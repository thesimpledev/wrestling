package ui

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
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

const (
	historyBattleRoyal = "BATTLE ROYAL"
	historyTournament  = "TOURNAMENT"
	showDivider        = "============================================================"
)

var leavePromptLines = []string{
	showDivider,
	"  LEAVE THE SHOW?",
	showDivider,
	"",
	"  [C] Cancel the show: nothing from it is recorded",
	"  [S] Skip to the results: the rest of the card is simulated",
	"",
	"  [ESC] Keep watching",
}

type CareerShowScreen struct {
	fed  *engine.Federation
	save *engine.FederationSave
	card []engine.BookedMatch
	mode ShowMode

	// The federation as it stood before the show, for cancelling the show,
	// and the show's name and week, which move on when the show ends.
	snapshot *engine.Federation
	showName string
	showWeek int

	currentIdx    int
	phase         ShowPhase
	askingToLeave bool

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

	// Results of every match on the show, for closing out the fight card.
	cardResults []*engine.MatchResult

	// For battle royals embedded in career
	brScreen *BattleRoyalScreen
	inBR     bool

	// For tournaments embedded in career
	tournScreen *TournamentScreen
	inTourn     bool

	cards map[string]*engine.WrestlerCard
}

func NewCareerShowScreen(fed *engine.Federation, save *engine.FederationSave, card []engine.BookedMatch, mode ShowMode, g *Game) *CareerShowScreen {
	cs := &CareerShowScreen{
		fed:      fed,
		save:     save,
		card:     card,
		mode:     mode,
		showName: fed.ShowName(),
		showWeek: fed.Week,
		speed:    30,
		results:  []string{},
		cards:    make(map[string]*engine.WrestlerCard, len(g.Roster)),
	}
	for _, w := range g.Roster {
		cs.cards[w.Name] = w
	}
	snapshot, err := fed.Clone()
	if err != nil {
		g.SetNotice("This show cannot be cancelled once it starts: " + err.Error())
	}
	cs.snapshot = snapshot
	cs.startMatch(g)
	return cs
}

// ─── Starting matches ───────────────────────────────────────────────────────

// startMatch begins the current match on the card, passing over any that
// cannot take place, and closes the show when the card is finished.
func (cs *CareerShowScreen) startMatch(g *Game) {
	for cs.currentIdx < len(cs.card) {
		if cs.begin(g, cs.card[cs.currentIdx]) {
			return
		}
		cs.results = append(cs.results, fmt.Sprintf("%d. CANCELLED: wrestlers not available", cs.currentIdx+1))
		cs.currentIdx++
	}
	cs.finishShow(g)
}

func (cs *CareerShowScreen) nextMatch(g *Game) {
	cs.currentIdx++
	cs.startMatch(g)
}

func (cs *CareerShowScreen) begin(g *Game, booked engine.BookedMatch) bool {
	cs.inBR, cs.inTourn = false, false
	switch {
	case len(booked.BREntrants) > 0:
		return cs.beginBattleRoyal(g, booked)
	case booked.IsTournament:
		return cs.beginTournament(g, booked)
	default:
		return cs.beginMatch(g, booked)
	}
}

func (cs *CareerShowScreen) beginBattleRoyal(g *Game, booked engine.BookedMatch) bool {
	var entrants []*engine.WrestlerCard
	for _, name := range booked.BREntrants {
		if w, ok := cs.cards[name]; ok {
			entrants = append(entrants, w)
		}
	}
	if len(entrants) < minBattleRoyalField {
		return false
	}

	br := NewBattleRoyalScreen(entrants, g)
	br.embedded = true
	br.champion = br.wrestlers[0]
	br.phase = BRShowingBracket
	cs.brScreen, cs.inBR = br, true

	if cs.shouldSimulate() {
		br.runToEnd(g)
		cs.recordBattleRoyal()
		cs.showSummary(historyBattleRoyal)
	}
	return true
}

func (cs *CareerShowScreen) beginTournament(g *Game, booked engine.BookedMatch) bool {
	if booked.TournSize < 2 {
		return false
	}
	seeds := make([]*engine.WrestlerCard, booked.TournSize)
	for i, name := range booked.TournSeeds {
		if i < len(seeds) {
			seeds[i] = cs.cards[name]
		}
	}

	ts := newSeededTournament(g, seeds)
	ts.embedded = true
	ts.onMatchDone = cs.recordTournamentMatch
	cs.tournScreen, cs.inTourn = ts, true

	if cs.shouldSimulate() {
		ts.runToEnd(g)
		cs.recordTournament()
		cs.showSummary(historyTournament)
	}
	return true
}

// showSummary replaces the match log with the outcome of a battle royal or
// tournament that was simulated rather than watched.
func (cs *CareerShowScreen) showSummary(kind string) {
	cs.inBR, cs.inTourn = false, false
	cs.lines = []string{
		showDivider,
		fmt.Sprintf("  Match %d of %d: %s", cs.currentIdx+1, len(cs.card), kind),
		showDivider,
		"",
		"  " + cs.results[len(cs.results)-1],
	}
	cs.scroll = 0
	cs.phase = ShowMatchResult
}

func (cs *CareerShowScreen) beginMatch(g *Game, booked engine.BookedMatch) bool {
	match, ok := cs.buildMatch(booked)
	if !ok {
		return false
	}
	match.Rules = g.Rules
	match.ApplyInjuries(g.Injuries.IsInjured)

	cs.match = match
	cs.events = match.Run()
	cs.shown = 0
	cs.autoPlay = false
	cs.ticker = 0
	cs.scroll = 0
	cs.lines = cs.matchHeader(booked)
	cs.phase = ShowRunning

	if cs.shouldSimulate() {
		cs.simulateCurrentMatch(g)
	}
	return true
}

func (cs *CareerShowScreen) cardsFor(names []string) ([]*engine.WrestlerCard, bool) {
	found := make([]*engine.WrestlerCard, 0, len(names))
	for _, name := range names {
		w, ok := cs.cards[name]
		if !ok {
			return nil, false
		}
		found = append(found, w)
	}
	return found, len(found) > 0
}

func (cs *CareerShowScreen) buildMatch(booked engine.BookedMatch) (*engine.Match, bool) {
	side1, ok1 := cs.cardsFor(booked.Side1)
	side2, ok2 := cs.cardsFor(booked.Side2)
	if !ok1 || !ok2 {
		return nil, false
	}

	if booked.Type == engine.MatchTag {
		if len(side1) < 2 || len(side2) < 2 {
			return nil, false
		}
		match := engine.NewTagMatch(side1[0], side1[1], side2[0], side2[1])
		match.Sides[0].RegularPartners = booked.Side1Regular
		match.Sides[1].RegularPartners = booked.Side2Regular
		return match, true
	}

	match := engine.NewMatch(side1[0], side2[0])
	match.Type = booked.Type
	match.InitForMatchType()
	match.IsFeud = cs.fed.IsRival(side1[0].Name, side2[0].Name)
	return match, true
}

func (cs *CareerShowScreen) titleOnTheLine(booked engine.BookedMatch) bool {
	return booked.IsTitle && booked.TitleIndex >= 0 && booked.TitleIndex < len(cs.fed.Championships)
}

func (cs *CareerShowScreen) matchHeader(booked engine.BookedMatch) []string {
	title := ""
	if cs.titleOnTheLine(booked) {
		title = fmt.Sprintf(" (%s)", cs.fed.Championships[booked.TitleIndex].Name)
	}
	return []string{
		showDivider,
		fmt.Sprintf("  Match %d of %d: %s%s", cs.currentIdx+1, len(cs.card), engine.MatchTypeString(booked.Type), title),
		fmt.Sprintf("  %s  vs  %s", strings.Join(booked.Side1, " & "), strings.Join(booked.Side2, " & ")),
		showDivider,
		"",
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
	for cs.shown < len(cs.events) {
		cs.lines = append(cs.lines, cs.events[cs.shown].Text)
		cs.shown++
	}
	cs.processMatchResult(g)
	cs.phase = ShowMatchResult
}

// ─── Recording results ──────────────────────────────────────────────────────

func (cs *CareerShowScreen) recordBattleRoyal() {
	br := cs.brScreen
	for _, result := range br.cardResults {
		if result == nil || result.Draw() {
			continue
		}
		cs.fed.RecordResult(engine.MatchHistoryEntry{
			Winner: result.Winner, Loser: result.Loser, Method: result.Method, MatchType: historyBattleRoyal,
		})
		cs.fed.AddRivalry(result.Winner, result.Loser, 1)
	}
	cs.cardResults = append(cs.cardResults, br.cardResults...)
	cs.fed.TitleShotEarned = br.champion.Name
	cs.results = append(cs.results, fmt.Sprintf("%d. [BATTLE ROYAL] Winner: %s", cs.currentIdx+1, br.champion.Name))
}

func (cs *CareerShowScreen) recordTournamentMatch(result *engine.MatchResult) {
	cs.fed.RecordResult(engine.MatchHistoryEntry{
		Winner: result.Winner, Loser: result.Loser, Method: result.Method, MatchType: historyTournament,
	})
	cs.fed.AddRivalry(result.Winner, result.Loser, 1)
}

func (cs *CareerShowScreen) recordTournament() {
	ts := cs.tournScreen
	cs.cardResults = append(cs.cardResults, ts.cardResults...)

	winner := ts.results[ts.totalRounds-1][0]
	if winner == nil {
		cs.results = append(cs.results, fmt.Sprintf("%d. [TOURNAMENT] No winner", cs.currentIdx+1))
		return
	}
	booked := cs.card[cs.currentIdx]
	if cs.titleOnTheLine(booked) && cs.fed.ChampionOf(booked.TitleIndex) == "" {
		cs.fed.ChangeTitleHolder(booked.TitleIndex, winner.Name, "", "tournament")
	}
	cs.results = append(cs.results, fmt.Sprintf("%d. [TOURNAMENT] Winner: %s", cs.currentIdx+1, winner.Name))
}

func (cs *CareerShowScreen) processMatchResult(g *Game) {
	result := cs.match.Result()
	booked := cs.card[cs.currentIdx]
	cs.cardResults = append(cs.cardResults, result)

	if result == nil || result.Draw() {
		cs.recordDraw(booked)
	} else {
		cs.recordWin(booked, result)
	}
	cs.settleTitle(booked, result)

	cs.lines = append(cs.lines, "", showDivider, matchResultBanner(result), showDivider)
	cs.scrollToBottom(g)
}

func (cs *CareerShowScreen) recordWin(booked engine.BookedMatch, result *engine.MatchResult) {
	typeStr := engine.MatchTypeString(booked.Type)
	cs.fed.RecordResult(engine.MatchHistoryEntry{
		Winner: result.Winner, Loser: result.Loser, Method: result.Method,
		MatchType: typeStr, IsTitle: cs.titleOnTheLine(booked),
	})

	cs.fed.AddRivalry(result.Winner, result.Loser, 1)
	if result.InjuredWrestler != "" {
		cs.fed.AddRivalry(result.Winner, result.Loser, 2)
	}
	if result.FeudText != "" {
		cs.fed.AddRivalry(result.Winner, result.Loser, 2)
	}

	titleTag := ""
	if cs.titleOnTheLine(booked) {
		titleTag = fmt.Sprintf(" [%s]", cs.fed.Championships[booked.TitleIndex].Name)
	}
	cs.results = append(cs.results, fmt.Sprintf("%d. [%s%s] %s def. %s by %s",
		cs.currentIdx+1, typeStr, titleTag, result.Winner, result.Loser, result.Method))
}

func (cs *CareerShowScreen) recordDraw(booked engine.BookedMatch) {
	for i := 0; i < len(booked.Side1) && i < len(booked.Side2); i++ {
		cs.fed.RecordDraw(booked.Side1[i], booked.Side2[i])
	}
	cs.results = append(cs.results, fmt.Sprintf("%d. DRAW", cs.currentIdx+1))
}

// settleTitle counts the match as the title's match for this PPV cycle, uses
// up an earned title shot, and moves the title if the champion lost.
func (cs *CareerShowScreen) settleTitle(booked engine.BookedMatch, result *engine.MatchResult) {
	if !cs.titleOnTheLine(booked) {
		return
	}
	cs.fed.RecordTitleMatch(booked.TitleIndex)
	if booked.TitleIndex == 0 && bookedIn(booked, cs.fed.TitleShotEarned) {
		cs.fed.TitleShotEarned = ""
	}
	if result == nil || result.Draw() {
		return
	}
	if result.Winner != cs.fed.ChampionOf(booked.TitleIndex) {
		cs.fed.ChangeTitleHolder(booked.TitleIndex, result.Winner, result.Loser, result.Method)
	}
}

func bookedIn(booked engine.BookedMatch, name string) bool {
	if name == "" {
		return false
	}
	for _, side := range [][]string{booked.Side1, booked.Side2} {
		for _, wrestler := range side {
			if wrestler == name {
				return true
			}
		}
	}
	return false
}

func (cs *CareerShowScreen) finishShow(g *Game) {
	g.EndFightCard(cs.cardResults)
	cs.fed.AdvanceWeek()
	g.SaveFederations(cs.save)
	cs.phase = ShowComplete
}

// ─── Leaving a show part way through ────────────────────────────────────────

func (cs *CareerShowScreen) updateLeavePrompt(g *Game) {
	switch {
	case g.in.JustPressed(ebiten.KeyEscape):
		cs.askingToLeave = false
	case g.in.JustPressed(ebiten.KeyC):
		cs.cancelShow(g)
	case g.in.JustPressed(ebiten.KeyS):
		cs.skipToResults(g)
	}
}

// cancelShow puts the federation back as it was before the show. Nothing has
// been saved or counted against injuries yet, so there is nothing else to undo.
func (cs *CareerShowScreen) cancelShow(g *Game) {
	if cs.snapshot == nil {
		return
	}
	*cs.fed = *cs.snapshot
	g.SetScreen(NewCareerBookScreen(cs.fed, cs.save, cs.card, g))
}

func (cs *CareerShowScreen) skipToResults(g *Game) {
	cs.askingToLeave = false
	cs.mode = ShowModeSimulate
	cs.finishCurrent(g)
	for cs.phase != ShowComplete && cs.currentIdx < len(cs.card) {
		cs.nextMatch(g)
	}
}

// finishCurrent simulates what is left of the match on screen, keeping
// whatever part of it has already been shown.
func (cs *CareerShowScreen) finishCurrent(g *Game) {
	switch {
	case cs.inBR:
		if cs.brScreen.phase != BRFinished {
			cs.brScreen.runToEnd(g)
			cs.recordBattleRoyal()
		}
	case cs.inTourn:
		if cs.tournScreen.phase != TournFinished {
			cs.tournScreen.runToEnd(g)
			cs.recordTournament()
		}
	case cs.phase == ShowRunning:
		cs.simulateCurrentMatch(g)
	}
}

// ─── Update ─────────────────────────────────────────────────────────────────

func (cs *CareerShowScreen) Update(g *Game) error {
	if cs.phase == ShowComplete {
		if confirmPressed(g.in) || g.in.JustPressed(ebiten.KeyEscape) {
			g.SetScreen(NewCareerScreen(cs.fed, cs.save))
		}
		return nil
	}
	if cs.askingToLeave {
		cs.updateLeavePrompt(g)
		return nil
	}
	if g.in.JustPressed(ebiten.KeyEscape) {
		cs.askingToLeave = true
		return nil
	}

	switch {
	case cs.inBR:
		return cs.updateBR(g)
	case cs.inTourn:
		return cs.updateTournament(g)
	case cs.phase == ShowRunning:
		cs.updateRunning(g)
	case confirmPressed(g.in):
		cs.nextMatch(g)
	}
	return nil
}

func (cs *CareerShowScreen) updateBR(g *Game) error {
	br := cs.brScreen
	if br.phase == BRFinished {
		if confirmPressed(g.in) {
			cs.nextMatch(g)
		}
		return nil
	}
	if err := br.Update(g); err != nil {
		return err
	}
	if br.phase == BRFinished {
		cs.recordBattleRoyal()
	}
	return nil
}

func (cs *CareerShowScreen) updateTournament(g *Game) error {
	ts := cs.tournScreen
	if ts.phase == TournFinished {
		if confirmPressed(g.in) {
			cs.nextMatch(g)
		}
		return nil
	}
	if err := ts.Update(g); err != nil {
		return err
	}
	if ts.phase == TournFinished {
		cs.recordTournament()
	}
	return nil
}

func (cs *CareerShowScreen) updateRunning(g *Game) {
	if g.in.JustPressed(ebiten.KeyA) {
		cs.autoPlay = !cs.autoPlay
	}
	if g.in.JustPressed(ebiten.KeyEqual) || g.in.JustPressed(ebiten.KeyNumpadAdd) {
		if cs.speed > 5 {
			cs.speed -= 5
		}
	}
	if g.in.JustPressed(ebiten.KeyMinus) || g.in.JustPressed(ebiten.KeyNumpadSubtract) {
		cs.speed += 5
	}

	if g.in.Pressed(ebiten.KeyUp) {
		if cs.scroll > 0 {
			cs.scroll--
		}
	}
	if g.in.Pressed(ebiten.KeyDown) {
		max := cs.maxScroll(g)
		if cs.scroll < max {
			cs.scroll++
		}
	}

	advance := false
	if g.in.JustPressed(ebiten.KeySpace) || g.in.JustPressed(ebiten.KeyEnter) {
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

// ─── Drawing ────────────────────────────────────────────────────────────────

func (cs *CareerShowScreen) Draw(screen *ebiten.Image, g *Game) {
	switch {
	case cs.phase == ShowComplete:
		screen.Fill(Background)
		drawLines(screen, cs.completeLines())
		DrawText(screen, "[SPACE] Continue  [ESC] Federation Dashboard", Margin, g.screenH-LineHeight-Margin)
	case cs.askingToLeave:
		screen.Fill(Background)
		drawLines(screen, leavePromptLines)
	case cs.inBR:
		cs.brScreen.Draw(screen, g)
	case cs.inTourn:
		cs.tournScreen.Draw(screen, g)
	default:
		screen.Fill(Background)
		cs.drawMatch(screen, g)
	}
}

func drawLines(screen *ebiten.Image, lines []string) {
	y := Margin
	for _, line := range lines {
		DrawText(screen, line, Margin, y)
		y += LineHeight
	}
}

func (cs *CareerShowScreen) drawMatch(screen *ebiten.Image, g *Game) {
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
	DrawText(screen, cs.statusLine(), Margin, statusBarY)
}

func (cs *CareerShowScreen) statusLine() string {
	switch {
	case cs.phase == ShowRunning && cs.autoPlay:
		return "AUTO-PLAY ON  [A] Stop  [+/-] Speed  [ESC] Leave"
	case cs.phase == ShowRunning:
		return "[SPACE] Step  [A] Auto-play  [+/-] Speed  [ESC] Leave"
	}
	if remaining := len(cs.card) - cs.currentIdx - 1; remaining > 0 {
		return fmt.Sprintf("[SPACE] Next Match (%d remaining)  [ESC] Leave", remaining)
	}
	return "[SPACE] Show Results  [ESC] Leave"
}

// completeLines is the results screen: it names the show and week that were
// just completed, not the ones the federation has moved on to.
func (cs *CareerShowScreen) completeLines() []string {
	lines := []string{showDivider, fmt.Sprintf("  %s: RESULTS", cs.showName), showDivider, ""}
	for _, r := range cs.results {
		lines = append(lines, "  "+r)
	}
	lines = append(lines, "")
	for i, ch := range cs.fed.Championships {
		lines = append(lines, cs.titleLine(i, ch))
	}
	return append(lines, "", fmt.Sprintf("Week %d complete. Federation saved.", cs.showWeek))
}

func (cs *CareerShowScreen) titleLine(index int, ch engine.Championship) string {
	if ch.Champion != "" {
		return fmt.Sprintf("%s: %s", ch.Name, ch.Champion)
	}
	if stripped := cs.strippedOnThisShow(index); stripped != "" {
		return fmt.Sprintf("%s: VACANT (%s stripped: no title match on this PPV)", ch.Name, stripped)
	}
	return fmt.Sprintf("%s: VACANT", ch.Name)
}

func (cs *CareerShowScreen) strippedOnThisShow(index int) string {
	history := cs.fed.Championships[index].History
	if len(history) == 0 {
		return ""
	}
	last := history[len(history)-1]
	if last.Method != "vacated" || last.Week != cs.showWeek {
		return ""
	}
	return last.Loser
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
