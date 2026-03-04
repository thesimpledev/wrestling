package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
	"wrestling/loader"
)

type ShowMode int

const (
	ShowModeWatch     ShowMode = iota // Watch all matches
	ShowModeSimulate                  // Instant results
	ShowModeMainEvent                 // Simulate undercard, watch last match
)

type ShowPhase int

const (
	ShowRunning ShowPhase = iota
	ShowMatchResult
	ShowComplete
)

type CareerShowScreen struct {
	career *engine.CareerSave
	card   []engine.BookedMatch
	mode   ShowMode

	currentIdx int
	phase      ShowPhase

	// Match display
	match    *engine.Match
	events   []engine.Event
	shown    int
	lines    []string
	scroll   int
	autoPlay bool
	speed    int
	ticker   int

	// Results summary
	results []string

	// For battle royals embedded in career
	brScreen *BattleRoyalScreen
	inBR     bool

	// For tournaments embedded in career
	tournScreen *TournamentScreen
	inTourn     bool

	roster []*engine.WrestlerCard
}

func NewCareerShowScreen(career *engine.CareerSave, card []engine.BookedMatch, mode ShowMode, g *Game) *CareerShowScreen {
	cs := &CareerShowScreen{
		career:  career,
		card:    card,
		mode:    mode,
		speed:   30,
		results: []string{},
		roster:  g.Roster,
	}
	cs.startMatch(g)
	return cs
}

func (cs *CareerShowScreen) startMatch(g *Game) {
	if cs.currentIdx >= len(cs.card) {
		cs.finishShow(g)
		return
	}

	booked := cs.card[cs.currentIdx]

	// Battle Royal
	if len(booked.BREntrants) > 0 {
		var picks []*engine.WrestlerCard
		rosterMap := make(map[string]*engine.WrestlerCard)
		for _, w := range g.Roster {
			rosterMap[w.Name] = w
		}
		for _, name := range booked.BREntrants {
			if w, ok := rosterMap[name]; ok {
				picks = append(picks, w)
			}
		}
		if len(picks) >= 3 {
			cs.brScreen = NewBattleRoyalScreen(picks, g)
			// Skip intro in career mode — go straight to standings
			cs.brScreen.champion = cs.brScreen.wrestlers[0]
			cs.brScreen.phase = BRShowingBracket
			cs.inBR = true
			cs.inTourn = false

			if cs.shouldSimulate() {
				cs.simulateBR(g)
			}
			return
		}
	}

	// Tournament
	if booked.IsTournament {
		rosterMap := make(map[string]*engine.WrestlerCard)
		for _, w := range g.Roster {
			rosterMap[w.Name] = w
		}
		ts := NewTournamentScreen(g)
		ts.bracketSize = booked.TournSize
		ts.totalRounds = 0
		size := booked.TournSize
		for size > 1 {
			ts.totalRounds++
			size /= 2
		}
		ts.seeds = make([]*engine.WrestlerCard, booked.TournSize)
		for i, name := range booked.TournSeeds {
			if i < booked.TournSize {
				if w, ok := rosterMap[name]; ok {
					ts.seeds[i] = w
				}
			}
		}
		ts.results = make([][]*engine.WrestlerCard, ts.totalRounds)
		for r := 0; r < ts.totalRounds; r++ {
			ts.results[r] = make([]*engine.WrestlerCard, booked.TournSize/(1<<(r+1)))
		}
		ts.buildBracketLines()
		ts.phase = TournShowBracket
		cs.tournScreen = ts
		cs.inTourn = true
		cs.inBR = false

		if cs.shouldSimulate() {
			cs.simulateTournament(g)
		}
		return
	}

	// Standard match
	rosterMap := make(map[string]*engine.WrestlerCard)
	for _, w := range g.Roster {
		rosterMap[w.Name] = w
	}

	s1Name := ""
	s2Name := ""
	if len(booked.Side1) > 0 {
		s1Name = booked.Side1[0]
	}
	if len(booked.Side2) > 0 {
		s2Name = booked.Side2[0]
	}

	card1, ok1 := rosterMap[s1Name]
	card2, ok2 := rosterMap[s2Name]
	if !ok1 || !ok2 {
		// Skip invalid match
		cs.results = append(cs.results, fmt.Sprintf("%d. CANCELLED — invalid wrestlers", cs.currentIdx+1))
		cs.currentIdx++
		cs.startMatch(g)
		return
	}

	match := engine.NewMatch(card1, card2)
	match.Type = booked.Type
	match.InitForMatchType()
	match.ApplyInjuries(g.Injuries.IsInjured)

	// Auto-apply feud rules for rivals
	if cs.career.IsRival(s1Name, s2Name) {
		match.IsFeud = true
	}

	cs.match = match
	cs.events = match.Run()
	cs.shown = 0
	cs.autoPlay = false
	cs.ticker = 0
	cs.scroll = 0
	cs.inBR = false
	cs.inTourn = false

	typeStr := engine.MatchTypeString(booked.Type)
	titleStr := ""
	if booked.IsTitle {
		titleStr = " (TITLE MATCH)"
	}

	cs.lines = []string{
		"============================================================",
		fmt.Sprintf("  Match %d of %d — %s%s", cs.currentIdx+1, len(cs.card), typeStr, titleStr),
		fmt.Sprintf("  %s  vs  %s", s1Name, s2Name),
		"============================================================",
		"",
	}

	cs.phase = ShowRunning

	if cs.shouldSimulate() {
		cs.simulateCurrentMatch(g)
	}
}

func (cs *CareerShowScreen) shouldSimulate() bool {
	if cs.mode == ShowModeSimulate {
		return true
	}
	if cs.mode == ShowModeMainEvent && cs.currentIdx < len(cs.card)-1 {
		return true
	}
	return false
}

func (cs *CareerShowScreen) simulateCurrentMatch(g *Game) {
	// Show all events at once
	for cs.shown < len(cs.events) {
		cs.lines = append(cs.lines, cs.events[cs.shown].Text)
		cs.shown++
	}
	cs.processMatchResult(g)
	cs.phase = ShowMatchResult
}

func (cs *CareerShowScreen) simulateBR(g *Game) {
	// Run all BR matches
	for cs.brScreen.nextIdx < len(cs.brScreen.wrestlers) {
		cs.brScreen.startNextMatch(g)
		cs.brScreen.finishSubMatch(g)
	}
	// Record results
	for _, eliminated := range cs.brScreen.eliminated {
		cs.career.RecordResult(cs.brScreen.champion.Name, eliminated, "elimination", false)
		cs.career.AddRivalry(cs.brScreen.champion.Name, eliminated, 1)
	}
	// BR winner earns title shot
	cs.career.TitleShotEarned = cs.brScreen.champion.Name

	cs.results = append(cs.results, fmt.Sprintf("%d. [BATTLE ROYAL] Winner: %s", cs.currentIdx+1, cs.brScreen.champion.Name))
	cs.inBR = false
	cs.currentIdx++
	cs.phase = ShowMatchResult
}

func (cs *CareerShowScreen) simulateTournament(g *Game) {
	ts := cs.tournScreen
	for ts.currentRound < ts.totalRounds {
		matchesInRound := ts.bracketSize / (1 << (ts.currentRound + 1))
		for ts.currentMatch < matchesInRound {
			w1, w2 := ts.getMatchup(ts.currentRound, ts.currentMatch)
			if w1 == nil || w2 == nil {
				ts.currentMatch++
				continue
			}
			match := engine.NewMatch(w1, w2)
			match.Type = engine.MatchSingles
			match.InitForMatchType()
			match.ApplyInjuries(g.Injuries.IsInjured)
			ts.match = match
			ts.events = match.Run()
			ts.shown = len(ts.events)

			result := match.Result()
			if result != nil {
				if result.InjuredWrestler != "" && result.InjuryCards > 0 {
					g.Injuries.RecordInjury(result.InjuredWrestler, result.InjuryCards)
				}
				var winner *engine.WrestlerCard
				if result.Winner == w1.Name {
					winner = w1
				} else {
					winner = w2
				}
				ts.results[ts.currentRound][ts.currentMatch] = winner
				cs.career.RecordResult(result.Winner, result.Loser, result.Method, false)
				cs.career.AddRivalry(result.Winner, result.Loser, 1)
			} else {
				ts.results[ts.currentRound][ts.currentMatch] = w1
			}
			ts.currentMatch++
		}
		g.Injuries.DecrementAll()
		ts.currentRound++
		ts.currentMatch = 0
	}
	loader.SaveInjuries(g.Store, g.Injuries)

	// Determine tournament winner
	winner := ts.results[ts.totalRounds-1][0]
	winnerName := "Unknown"
	if winner != nil {
		winnerName = winner.Name
	}

	booked := cs.card[cs.currentIdx]
	if booked.IsTitle && cs.career.WorldChampion() == "" {
		cs.career.ChangeTitleHolder(winnerName, "", "tournament")
	}

	cs.results = append(cs.results, fmt.Sprintf("%d. [TOURNAMENT] Winner: %s", cs.currentIdx+1, winnerName))
	cs.inTourn = false
	cs.currentIdx++
	cs.phase = ShowMatchResult
}

func (cs *CareerShowScreen) processMatchResult(g *Game) {
	result := cs.match.Result()
	booked := cs.card[cs.currentIdx]

	if result != nil {
		// Record injuries
		if result.InjuredWrestler != "" && result.InjuryCards > 0 {
			g.Injuries.RecordInjury(result.InjuredWrestler, result.InjuryCards)
		}

		// Update career records
		cs.career.RecordResult(result.Winner, result.Loser, result.Method, booked.IsTitle)

		// Rivalry points
		cs.career.AddRivalry(result.Winner, result.Loser, 1)
		if result.InjuredWrestler != "" {
			cs.career.AddRivalry(result.Winner, result.Loser, 2) // +2 more for injury
		}
		if result.FeudText != "" {
			cs.career.AddRivalry(result.Winner, result.Loser, 2)
		}

		// Title change
		if booked.IsTitle {
			champ := cs.career.WorldChampion()
			if result.Winner != champ {
				cs.career.ChangeTitleHolder(result.Winner, result.Loser, result.Method)
			} else {
				// Successful defense — reset counter
				if len(cs.career.Championships) > 0 {
					cs.career.Championships[0].DefensesLeft = 4
				}
			}
		}

		typeStr := engine.MatchTypeString(booked.Type)
		titleTag := ""
		if booked.IsTitle {
			titleTag = " [TITLE]"
		}
		cs.results = append(cs.results, fmt.Sprintf("%d. [%s%s] %s def. %s by %s",
			cs.currentIdx+1, typeStr, titleTag, result.Winner, result.Loser, result.Method))
	} else {
		// Draw
		if len(booked.Side1) > 0 && len(booked.Side2) > 0 {
			cs.career.RecordDraw(booked.Side1[0], booked.Side2[0])
		}
		cs.results = append(cs.results, fmt.Sprintf("%d. DRAW", cs.currentIdx+1))
	}

	// Decrement injuries after each match
	g.Injuries.DecrementAll()
	loader.SaveInjuries(g.Store, g.Injuries)

	cs.lines = append(cs.lines, "")
	if result != nil {
		cs.lines = append(cs.lines, "============================================================")
		cs.lines = append(cs.lines, fmt.Sprintf("  WINNER: %s by %s", result.Winner, result.Method))
		cs.lines = append(cs.lines, "============================================================")
	}
	cs.scrollToBottom(g)
}

func (cs *CareerShowScreen) finishShow(g *Game) {
	// Advance week
	cs.career.AdvanceWeek()

	// Check if champion needs to vacate (defense overdue)
	if len(cs.career.Championships) > 0 && cs.career.Championships[0].DefensesLeft <= 0 {
		if cs.career.WorldChampion() != "" {
			cs.career.VacateTitle()
		}
	}

	// Save career
	loader.SaveCareer(g.Store, cs.career)

	cs.phase = ShowComplete
}

func (cs *CareerShowScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		// Save and return to career dashboard
		loader.SaveCareer(g.Store, cs.career)
		g.SetScreen(NewCareerScreen(cs.career))
		return nil
	}

	// Delegate to embedded BR/tournament screens if active
	if cs.inBR && !cs.shouldSimulate() {
		return cs.updateBR(g)
	}
	if cs.inTourn && !cs.shouldSimulate() {
		return cs.updateTournament(g)
	}

	switch cs.phase {
	case ShowRunning:
		cs.updateRunning(g)
	case ShowMatchResult:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			cs.currentIdx++
			cs.startMatch(g)
		}
	case ShowComplete:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.SetScreen(NewCareerScreen(cs.career))
		}
	}

	return nil
}

func (cs *CareerShowScreen) updateBR(g *Game) error {
	br := cs.brScreen
	oldPhase := br.phase

	// Let BR screen handle input
	br.Update(g)

	// If BR finished, record results and advance
	if br.phase == BRFinished && oldPhase != BRFinished {
		for _, eliminated := range br.eliminated {
			cs.career.RecordResult(br.champion.Name, eliminated, "elimination", false)
		}
		// BR winner earns title shot
		cs.career.TitleShotEarned = br.champion.Name
		cs.results = append(cs.results, fmt.Sprintf("%d. [BATTLE ROYAL] Winner: %s", cs.currentIdx+1, br.champion.Name))
	}

	// When user presses space on finished BR, move to next match
	if br.phase == BRFinished {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			cs.inBR = false
			cs.currentIdx++
			cs.phase = ShowMatchResult
		}
	}

	return nil
}

func (cs *CareerShowScreen) updateTournament(g *Game) error {
	ts := cs.tournScreen
	oldPhase := ts.phase

	ts.Update(g)

	if ts.phase == TournFinished && oldPhase != TournFinished {
		// Record all tournament results
		winner := ts.results[ts.totalRounds-1][0]
		if winner != nil {
			booked := cs.card[cs.currentIdx]
			if booked.IsTitle && cs.career.WorldChampion() == "" {
				cs.career.ChangeTitleHolder(winner.Name, "", "tournament")
			}
			cs.results = append(cs.results, fmt.Sprintf("%d. [TOURNAMENT] Winner: %s", cs.currentIdx+1, winner.Name))
		}
	}

	if ts.phase == TournFinished {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			cs.inTourn = false
			cs.currentIdx++
			cs.phase = ShowMatchResult
		}
	}

	return nil
}

func (cs *CareerShowScreen) updateRunning(g *Game) {
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		cs.autoPlay = !cs.autoPlay
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		if cs.speed > 5 {
			cs.speed -= 5
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		cs.speed += 5
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		if cs.scroll > 0 {
			cs.scroll--
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		max := cs.maxScroll(g)
		if cs.scroll < max {
			cs.scroll++
		}
	}

	advance := false
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		advance = true
	}
	if cs.autoPlay {
		cs.ticker++
		if cs.ticker >= cs.speed {
			cs.ticker = 0
			advance = true
		}
	}

	if advance && cs.shown < len(cs.events) {
		e := cs.events[cs.shown]
		cs.lines = append(cs.lines, e.Text)
		cs.shown++
		cs.scrollToBottom(g)

		if cs.shown >= len(cs.events) {
			cs.processMatchResult(g)
			cs.phase = ShowMatchResult
		}
	}
}

func (cs *CareerShowScreen) Draw(screen *ebiten.Image, g *Game) {
	// Delegate to embedded screens if active
	if cs.inBR && !cs.shouldSimulate() {
		cs.brScreen.Draw(screen, g)
		return
	}
	if cs.inTourn && !cs.shouldSimulate() {
		cs.tournScreen.Draw(screen, g)
		return
	}

	screen.Fill(Background)

	if cs.phase == ShowComplete {
		cs.drawComplete(screen, g)
		return
	}

	statusBarY := g.screenH - LineHeight - Margin

	startLine := cs.scroll
	if startLine > len(cs.lines)-1 {
		startLine = len(cs.lines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	y := Margin
	for i := startLine; i < len(cs.lines) && y < statusBarY; i++ {
		DrawText(screen, cs.lines[i], Margin, y)
		y += LineHeight
	}

	var status string
	switch cs.phase {
	case ShowRunning:
		if cs.autoPlay {
			status = "AUTO-PLAY ON  [A] Stop  [+/-] Speed  [ESC] Quit"
		} else {
			status = "[SPACE] Step  [A] Auto-play  [+/-] Speed  [ESC] Quit"
		}
	case ShowMatchResult:
		remaining := len(cs.card) - cs.currentIdx - 1
		if remaining > 0 {
			status = fmt.Sprintf("[SPACE] Next Match (%d remaining)  [ESC] Quit", remaining)
		} else {
			status = "[SPACE] Show Results  [ESC] Quit"
		}
	}
	DrawText(screen, status, Margin, statusBarY)
}

func (cs *CareerShowScreen) drawComplete(screen *ebiten.Image, g *Game) {
	y := Margin
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, fmt.Sprintf("  %s — RESULTS", cs.career.ShowName()), Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	for _, r := range cs.results {
		DrawText(screen, "  "+r, Margin, y)
		y += LineHeight
	}

	y += LineHeight

	// Show updated champion
	champ := cs.career.WorldChampion()
	if champ == "" {
		DrawText(screen, "World Heavyweight Championship: VACANT", Margin, y)
	} else {
		DrawText(screen, fmt.Sprintf("World Heavyweight Champion: %s", champ), Margin, y)
	}
	y += LineHeight

	DrawText(screen, fmt.Sprintf("Week %d complete. Career saved.", cs.career.Week), Margin, y)

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[SPACE] Continue  [ESC] Career Dashboard", Margin, statusY)
}

func (cs *CareerShowScreen) visibleLines(g *Game) int {
	if g.screenH == 0 {
		return 20
	}
	return (g.screenH - Margin*2 - LineHeight) / LineHeight
}

func (cs *CareerShowScreen) maxScroll(g *Game) int {
	max := len(cs.lines) - cs.visibleLines(g)
	if max < 0 {
		return 0
	}
	return max
}

func (cs *CareerShowScreen) scrollToBottom(g *Game) {
	cs.scroll = cs.maxScroll(g)
}
