package ui

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
	"wrestling/loader"
)

type TournPhase int

const (
	TournSelectSize TournPhase = iota
	TournFillBracket
	TournShowBracket
	TournRunningMatch
	TournMatchResult
	TournFinished
)

var bracketSizes = []int{4, 8, 16}

type TournamentScreen struct {
	phase       TournPhase
	bracketSize int
	seeds       []*engine.WrestlerCard
	results     [][]*engine.WrestlerCard // [round][matchIdx] = winner

	currentRound int
	currentMatch int
	totalRounds  int

	// Sub-match display
	match    *engine.Match
	events   []engine.Event
	shown    int
	lines    []string
	scroll   int
	autoPlay bool
	speed    int
	ticker   int

	// Bracket fill
	fillCursor   int
	rosterCursor int

	// Size select
	sizeCursor int

	// Bracket view
	bracketLines []string
	bracketScroll int

	roster []*engine.WrestlerCard
}

func NewTournamentScreen(g *Game) *TournamentScreen {
	return &TournamentScreen{
		phase:  TournSelectSize,
		speed:  30,
		roster: g.Roster,
	}
}

func (t *TournamentScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if t.phase == TournSelectSize {
			g.SetScreen(NewMenuScreen())
			return nil
		}
		if t.phase == TournFillBracket {
			t.phase = TournSelectSize
			t.fillCursor = 0
			t.rosterCursor = 0
			return nil
		}
		// Any other phase — quit tournament
		g.SetScreen(NewMenuScreen())
		return nil
	}

	switch t.phase {
	case TournSelectSize:
		t.sizeCursor = handleListInput(t.sizeCursor, len(bracketSizes))
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			t.bracketSize = bracketSizes[t.sizeCursor]
			t.totalRounds = int(math.Log2(float64(t.bracketSize)))
			t.seeds = make([]*engine.WrestlerCard, t.bracketSize)
			t.results = make([][]*engine.WrestlerCard, t.totalRounds)
			for r := 0; r < t.totalRounds; r++ {
				matchesInRound := t.bracketSize / (1 << (r + 1))
				t.results[r] = make([]*engine.WrestlerCard, matchesInRound)
			}
			t.phase = TournFillBracket
			t.fillCursor = 0
			t.rosterCursor = 0
		}

	case TournFillBracket:
		t.updateFillBracket(g)

	case TournShowBracket:
		t.updateShowBracket(g)

	case TournRunningMatch:
		t.updateRunningMatch(g)

	case TournMatchResult:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			t.advanceToNext(g)
		}

	case TournFinished:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.SetScreen(NewMenuScreen())
		}
	}

	return nil
}

func (t *TournamentScreen) updateFillBracket(g *Game) {
	// rosterCursor navigates the roster; last option is "Auto-Fill Remaining"
	listLen := len(g.Roster) + 1
	t.rosterCursor = handleListInput(t.rosterCursor, listLen)

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if t.rosterCursor == len(g.Roster) {
			// Auto-fill remaining slots
			for i := range t.seeds {
				if t.seeds[i] == nil {
					t.seeds[i] = g.Roster[rand.Intn(len(g.Roster))]
				}
			}
			t.startBracket()
			return
		}
		t.seeds[t.fillCursor] = g.Roster[t.rosterCursor]
		t.fillCursor++
		if t.fillCursor >= t.bracketSize {
			t.startBracket()
		}
	}
}

func (t *TournamentScreen) startBracket() {
	t.currentRound = 0
	t.currentMatch = 0
	t.buildBracketLines()
	t.phase = TournShowBracket
}

func (t *TournamentScreen) updateShowBracket(g *Game) {
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		if t.bracketScroll > 0 {
			t.bracketScroll--
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		t.bracketScroll++
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		t.startCurrentMatch(g)
	}
}

func (t *TournamentScreen) startCurrentMatch(g *Game) {
	w1, w2 := t.getMatchup(t.currentRound, t.currentMatch)
	if w1 == nil || w2 == nil {
		return
	}

	match := engine.NewMatch(w1, w2)
	match.Type = engine.MatchSingles
	match.InitForMatchType()
	match.ApplyInjuries(g.Injuries.IsInjured)

	t.match = match
	t.events = match.Run()
	t.shown = 0
	t.autoPlay = false
	t.ticker = 0
	t.scroll = 0
	t.lines = []string{
		"============================================================",
		fmt.Sprintf("  TOURNAMENT — Round %d, Match %d", t.currentRound+1, t.currentMatch+1),
		fmt.Sprintf("  %s  vs  %s", w1.Name, w2.Name),
		"============================================================",
		"",
	}
	t.phase = TournRunningMatch
}

func (t *TournamentScreen) getMatchup(round, matchIdx int) (*engine.WrestlerCard, *engine.WrestlerCard) {
	if round == 0 {
		return t.seeds[matchIdx*2], t.seeds[matchIdx*2+1]
	}
	prev := t.results[round-1]
	return prev[matchIdx*2], prev[matchIdx*2+1]
}

func (t *TournamentScreen) updateRunningMatch(g *Game) {
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		t.autoPlay = !t.autoPlay
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		if t.speed > 5 {
			t.speed -= 5
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		t.speed += 5
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		if t.scroll > 0 {
			t.scroll--
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		max := t.maxScroll(g)
		if t.scroll < max {
			t.scroll++
		}
	}

	advance := false
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		advance = true
	}
	if t.autoPlay {
		t.ticker++
		if t.ticker >= t.speed {
			t.ticker = 0
			advance = true
		}
	}

	if advance && t.shown < len(t.events) {
		e := t.events[t.shown]
		t.lines = append(t.lines, e.Text)
		t.shown++
		t.scrollToBottom(g)

		if t.shown >= len(t.events) {
			t.finishSubMatch(g)
		}
	}
}

func (t *TournamentScreen) finishSubMatch(g *Game) {
	result := t.match.Result()

	// Record injuries
	if result != nil && result.InjuredWrestler != "" && result.InjuryCards > 0 {
		g.Injuries.RecordInjury(result.InjuredWrestler, result.InjuryCards)
	}
	loader.SaveInjuries(g.Store, g.Injuries)

	if result != nil {
		// Find the winner card
		var winner *engine.WrestlerCard
		w1, w2 := t.getMatchup(t.currentRound, t.currentMatch)
		if result.Winner == w1.Name {
			winner = w1
		} else {
			winner = w2
		}
		t.results[t.currentRound][t.currentMatch] = winner

		t.lines = append(t.lines, "")
		t.lines = append(t.lines, "============================================================")
		t.lines = append(t.lines, fmt.Sprintf("  %s WINS by %s!", result.Winner, result.Method))
		t.lines = append(t.lines, "============================================================")
	} else {
		// Draw — first wrestler advances by default
		w1, _ := t.getMatchup(t.currentRound, t.currentMatch)
		t.results[t.currentRound][t.currentMatch] = w1

		t.lines = append(t.lines, "")
		t.lines = append(t.lines, "  Draw! First-seeded wrestler advances.")
	}

	t.scrollToBottom(g)
	t.phase = TournMatchResult
}

func (t *TournamentScreen) advanceToNext(g *Game) {
	matchesInRound := t.bracketSize / (1 << (t.currentRound + 1))
	t.currentMatch++

	if t.currentMatch >= matchesInRound {
		// Decrement injuries once per round
		g.Injuries.DecrementAll()
		loader.SaveInjuries(g.Store, g.Injuries)

		t.currentRound++
		t.currentMatch = 0

		if t.currentRound >= t.totalRounds {
			t.buildBracketLines()
			t.phase = TournFinished
			return
		}
	}

	t.buildBracketLines()
	t.bracketScroll = 0
	t.phase = TournShowBracket
}

// ─── BRACKET RENDERING ──────────────────────────────────────────────────────

func (t *TournamentScreen) buildBracketLines() {
	var lines []string

	roundLabel := ""
	if t.currentRound < t.totalRounds {
		roundLabel = fmt.Sprintf("Round %d of %d", t.currentRound+1, t.totalRounds)
	} else {
		roundLabel = "COMPLETE"
	}
	lines = append(lines, fmt.Sprintf("RING WARS TOURNAMENT - %s", roundLabel))
	lines = append(lines, "============================================================")
	lines = append(lines, "")

	// Build bracket column by column
	bracketStr := t.renderBracket()
	lines = append(lines, bracketStr...)

	// Show next matchup info
	lines = append(lines, "")
	if t.currentRound < t.totalRounds {
		w1, w2 := t.getMatchup(t.currentRound, t.currentMatch)
		if w1 != nil && w2 != nil {
			lines = append(lines, fmt.Sprintf("  Next: %s vs %s", w1.Name, w2.Name))
		}
		lines = append(lines, "  [SPACE] Start Match  [ESC] Quit Tournament")
	} else {
		// Tournament complete
		winner := t.results[t.totalRounds-1][0]
		if winner != nil {
			lines = append(lines, fmt.Sprintf("  TOURNAMENT CHAMPION: %s", winner.Name))
		}
		lines = append(lines, "  [SPACE] Return to Menu  [ESC] Quit")
	}

	t.bracketLines = lines
}

func (t *TournamentScreen) renderBracket() []string {
	// For each seed slot, build a line-based bracket representation
	// Use a column-based approach where each round adds connectors

	nameWidth := 20

	// Calculate total height: bracketSize lines for seeds + spacing
	totalSlots := t.bracketSize
	height := totalSlots * 2

	// Build grid of characters
	cols := t.totalRounds + 1
	grid := make([][]string, height)
	for i := range grid {
		grid[i] = make([]string, cols)
		for j := range grid[i] {
			grid[i][j] = ""
		}
	}

	// Column 0: seeds
	for i := 0; i < totalSlots; i++ {
		row := i * 2
		name := "???"
		if t.seeds[i] != nil {
			name = truncName(t.seeds[i].Name, nameWidth-5)
		}
		grid[row][0] = fmt.Sprintf("[%2d] %-*s", i+1, nameWidth-5, name)
	}

	// Subsequent columns: round results
	for r := 0; r < t.totalRounds; r++ {
		matchesInRound := totalSlots / (1 << (r + 1))
		spacing := 1 << (r + 1)
		for m := 0; m < matchesInRound; m++ {
			row := m*spacing*2 + spacing - 1
			if row >= height {
				row = height - 1
			}
			winner := t.results[r][m]
			name := "???"
			if winner != nil {
				name = truncName(winner.Name, nameWidth-2)
			}
			// Highlight current match
			marker := " "
			if r == t.currentRound && m == t.currentMatch && t.currentRound < t.totalRounds {
				marker = ">"
			}
			grid[row][r+1] = fmt.Sprintf("%s%-*s", marker, nameWidth-1, name)
		}
	}

	// Convert grid to string lines with connectors
	var lines []string
	colWidth := nameWidth + 2

	for row := 0; row < height; row++ {
		var sb strings.Builder
		sb.WriteString("  ")
		for col := 0; col < cols; col++ {
			cell := grid[row][col]
			if cell == "" {
				// Check if we should draw a connector
				connector := t.getConnector(row, col, height, totalSlots)
				sb.WriteString(fmt.Sprintf("%-*s", colWidth, connector))
			} else {
				sb.WriteString(fmt.Sprintf("%-*s", colWidth, cell))
			}
		}
		line := sb.String()
		// Only include non-empty lines
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

func (t *TournamentScreen) getConnector(row, col, height, totalSlots int) string {
	if col == 0 {
		return ""
	}

	round := col - 1
	spacing := 1 << (round + 1)
	matchesInRound := totalSlots / (1 << (round + 1))

	for m := 0; m < matchesInRound; m++ {
		topRow := m * spacing * 2
		botRow := topRow + spacing
		midRow := topRow + spacing - 1

		if row == topRow || row == botRow {
			return "---+"
		}
		if row > topRow && row < midRow {
			return "   |"
		}
		if row == midRow {
			return "---+"
		}
		if row > midRow && row < botRow {
			return "   |"
		}
	}

	return ""
}

func truncName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-1] + "."
}

// ─── DRAW ────────────────────────────────────────────────────────────────────

func (t *TournamentScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)

	switch t.phase {
	case TournSelectSize:
		t.drawSelectSize(screen, g)
	case TournFillBracket:
		t.drawFillBracket(screen, g)
	case TournShowBracket, TournFinished:
		t.drawBracketView(screen, g)
	case TournRunningMatch, TournMatchResult:
		t.drawRunningMatch(screen, g)
	}
}

func (t *TournamentScreen) drawSelectSize(screen *ebiten.Image, g *Game) {
	y := Margin
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                    TOURNAMENT", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	DrawText(screen, "SELECT BRACKET SIZE:", Margin, y)
	y += LineHeight * 2

	for i, size := range bracketSizes {
		prefix := "  "
		if i == t.sizeCursor {
			prefix = "> "
		}
		DrawText(screen, fmt.Sprintf("%s%d Wrestlers", prefix, size), Margin, y)
		y += LineHeight
	}

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Back", Margin, statusY)
}

func (t *TournamentScreen) drawFillBracket(screen *ebiten.Image, g *Game) {
	y := Margin
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                  FILL TOURNAMENT BRACKET", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	// Show filled seeds so far
	DrawText(screen, fmt.Sprintf("Filling seed %d of %d:", t.fillCursor+1, t.bracketSize), Margin, y)
	y += LineHeight
	for i := 0; i < t.bracketSize; i++ {
		name := "(empty)"
		if t.seeds[i] != nil {
			name = t.seeds[i].Name
		}
		marker := "  "
		if i == t.fillCursor {
			marker = ">>"
		}
		DrawText(screen, fmt.Sprintf("  %s [%2d] %s", marker, i+1, name), Margin, y)
		y += LineHeight
	}

	y += LineHeight
	DrawText(screen, "SELECT WRESTLER:", Margin, y)
	y += LineHeight

	// Show roster list
	for i, card := range t.roster {
		prefix := "  "
		if i == t.rosterCursor {
			prefix = "> "
		}
		DrawText(screen, prefix+card.Name, Margin, y)
		y += LineHeight
	}
	// Auto-fill option
	prefix := "  "
	if t.rosterCursor == len(t.roster) {
		prefix = "> "
	}
	DrawText(screen, prefix+"[Auto-Fill Remaining]", Margin, y)

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Back", Margin, statusY)
}

func (t *TournamentScreen) drawBracketView(screen *ebiten.Image, g *Game) {
	if t.bracketLines == nil {
		return
	}

	statusBarY := g.screenH - LineHeight - Margin

	startLine := t.bracketScroll
	if startLine >= len(t.bracketLines) {
		startLine = len(t.bracketLines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	y := Margin
	for i := startLine; i < len(t.bracketLines) && y < statusBarY; i++ {
		DrawText(screen, t.bracketLines[i], Margin, y)
		y += LineHeight
	}

	var status string
	if t.phase == TournFinished {
		status = "[SPACE] Return to Menu  [UP/DOWN] Scroll  [ESC] Quit"
	} else {
		status = "[SPACE] Start Match  [UP/DOWN] Scroll  [ESC] Quit"
	}
	DrawText(screen, status, Margin, statusBarY)
}

func (t *TournamentScreen) drawRunningMatch(screen *ebiten.Image, g *Game) {
	statusBarY := g.screenH - LineHeight - Margin

	startLine := t.scroll
	if startLine > len(t.lines)-1 {
		startLine = len(t.lines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	y := Margin
	for i := startLine; i < len(t.lines) && y < statusBarY; i++ {
		DrawText(screen, t.lines[i], Margin, y)
		y += LineHeight
	}

	var status string
	if t.phase == TournMatchResult {
		status = "[SPACE] Back to Bracket  [ESC] Quit"
	} else if t.autoPlay {
		status = "AUTO-PLAY ON  [A] Stop  [+/-] Speed  [ESC] Quit"
	} else {
		status = "[SPACE] Step  [A] Auto-play  [+/-] Speed  [ESC] Quit"
	}
	DrawText(screen, status, Margin, statusBarY)
}

// ─── SCROLL HELPERS ──────────────────────────────────────────────────────────

func (t *TournamentScreen) visibleLines(g *Game) int {
	if g.screenH == 0 {
		return 20
	}
	return (g.screenH - Margin*2 - LineHeight) / LineHeight
}

func (t *TournamentScreen) maxScroll(g *Game) int {
	max := len(t.lines) - t.visibleLines(g)
	if max < 0 {
		return 0
	}
	return max
}

func (t *TournamentScreen) scrollToBottom(g *Game) {
	t.scroll = t.maxScroll(g)
}
