package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
)

type HistoryTab int

const (
	HistoryTabMatches HistoryTab = iota
	HistoryTabTitle
)

type CareerHistoryScreen struct {
	fed    *engine.Federation
	save   *engine.FederationSave
	tab    HistoryTab
	scroll int
	lines  []string
}

func NewCareerHistoryScreen(fed *engine.Federation, save *engine.FederationSave) *CareerHistoryScreen {
	h := &CareerHistoryScreen{fed: fed, save: save}
	h.buildLines()
	return h
}

func (h *CareerHistoryScreen) buildLines() {
	if h.tab == HistoryTabTitle {
		h.lines = titleHistoryLines(h.fed)
		return
	}
	h.lines = matchHistoryLines(h.fed)
}

func clipName(name string, width int) string {
	if len(name) > width {
		return name[:width]
	}
	return name
}

// matchHistoryLines lists the federation's matches, newest first.
func matchHistoryLines(fed *engine.Federation) []string {
	lines := []string{showDivider, "                    MATCH HISTORY", showDivider, ""}
	if len(fed.MatchHistory) == 0 {
		return append(lines, "  No matches played yet.")
	}

	lines = append(lines,
		fmt.Sprintf(" %-5s %-20s %-5s %-20s %-10s %s", "Week", "Winner", "", "Loser", "Method", "Match"),
		" ----------------------------------------------------------------------------")
	for i := len(fed.MatchHistory) - 1; i >= 0; i-- {
		m := fed.MatchHistory[i]
		titleTag := ""
		if m.IsTitle {
			titleTag = " [T]"
		}
		lines = append(lines, fmt.Sprintf(" W%-4d %-20s def. %-20s %-10s %s%s",
			m.Week, clipName(m.Winner, 20), clipName(m.Loser, 20), m.Method, m.MatchType, titleTag))
	}
	return lines
}

// titleHistoryLines lists every championship with its changes, newest first.
func titleHistoryLines(fed *engine.Federation) []string {
	lines := []string{showDivider, "                   TITLE HISTORY", showDivider, ""}
	if len(fed.Championships) == 0 {
		return append(lines, "  No championships configured.")
	}
	for i, ch := range fed.Championships {
		if i > 0 {
			lines = append(lines, "", "  --------------------------------------------------", "")
		}
		lines = append(lines, championshipLines(ch)...)
	}
	return lines
}

func championshipLines(ch engine.Championship) []string {
	status := "  Status: VACANT"
	if ch.Champion != "" {
		status = fmt.Sprintf("  Current Champion: %s", ch.Champion)
	}
	lines := []string{fmt.Sprintf("  %s", ch.Name), "", status, ""}
	if len(ch.History) == 0 {
		return append(lines, "  No title changes yet.")
	}

	lines = append(lines,
		fmt.Sprintf(" %-5s %-22s %-22s %s", "Week", "Winner", "Loser", "Method"),
		" -----------------------------------------------------------")
	for i := len(ch.History) - 1; i >= 0; i-- {
		tc := ch.History[i]
		winner := tc.Winner
		if winner == "" {
			winner = "(vacated)"
		}
		lines = append(lines, fmt.Sprintf(" W%-4d %-22s %-22s %s",
			tc.Week, clipName(winner, 22), clipName(tc.Loser, 22), tc.Method))
	}
	return lines
}

func (h *CareerHistoryScreen) Update(g *Game) error {
	if g.in.JustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewCareerScreen(h.fed, h.save))
		return nil
	}

	if g.in.JustPressed(ebiten.KeyTab) {
		if h.tab == HistoryTabMatches {
			h.tab = HistoryTabTitle
		} else {
			h.tab = HistoryTabMatches
		}
		h.scroll = 0
		h.buildLines()
	}

	if g.in.Pressed(ebiten.KeyUp) {
		if h.scroll > 0 {
			h.scroll--
		}
	}
	if g.in.Pressed(ebiten.KeyDown) {
		max := len(h.lines) - h.visibleLines(g)
		if max < 0 {
			max = 0
		}
		if h.scroll < max {
			h.scroll++
		}
	}

	return nil
}

func (h *CareerHistoryScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)

	statusBarY := g.screenH - LineHeight - Margin

	startLine := h.scroll
	if startLine >= len(h.lines) {
		startLine = len(h.lines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	y := Margin
	for i := startLine; i < len(h.lines) && y < statusBarY; i++ {
		DrawText(screen, h.lines[i], Margin, y)
		y += LineHeight
	}

	tabStr := "[TAB] Switch to Title History"
	if h.tab == HistoryTabTitle {
		tabStr = "[TAB] Switch to Match History"
	}
	DrawText(screen, fmt.Sprintf("%s  [UP/DOWN] Scroll  [ESC] Back", tabStr), Margin, statusBarY)
}

func (h *CareerHistoryScreen) visibleLines(g *Game) int {
	if g.screenH == 0 {
		return 20
	}
	return (g.screenH - Margin*2 - LineHeight) / LineHeight
}
