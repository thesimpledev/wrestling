package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
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
	var lines []string
	lines = append(lines, "============================================================")

	switch h.tab {
	case HistoryTabMatches:
		lines = append(lines, "                    MATCH HISTORY")
		lines = append(lines, "============================================================")
		lines = append(lines, "")

		if len(h.fed.MatchHistory) == 0 {
			lines = append(lines, "  No matches played yet.")
		} else {
			lines = append(lines, fmt.Sprintf(" %-5s %-20s %-5s %-20s %-10s %s",
				"Week", "Winner", "", "Loser", "Method", ""))
			lines = append(lines, " -----------------------------------------------------------")
			for i := len(h.fed.MatchHistory) - 1; i >= 0; i-- {
				m := h.fed.MatchHistory[i]
				titleTag := ""
				if m.IsTitle {
					titleTag = " [T]"
				}
				winnerName := m.Winner
				if len(winnerName) > 20 {
					winnerName = winnerName[:20]
				}
				loserName := m.Loser
				if len(loserName) > 20 {
					loserName = loserName[:20]
				}
				lines = append(lines, fmt.Sprintf(" W%-4d %-20s def. %-20s %-10s%s",
					m.Week, winnerName, loserName, m.Method, titleTag))
			}
		}

	case HistoryTabTitle:
		lines = append(lines, "                   TITLE HISTORY")
		lines = append(lines, "============================================================")
		lines = append(lines, "")

		if len(h.fed.Championships) == 0 {
			lines = append(lines, "  No championships configured.")
		} else {
			for ci, ch := range h.fed.Championships {
				if ci > 0 {
					lines = append(lines, "")
					lines = append(lines, "  --------------------------------------------------")
					lines = append(lines, "")
				}
				lines = append(lines, fmt.Sprintf("  %s", ch.Name))
				lines = append(lines, "")

				if ch.Champion == "" {
					lines = append(lines, "  Status: VACANT")
				} else {
					lines = append(lines, fmt.Sprintf("  Current Champion: %s", ch.Champion))
				}
				lines = append(lines, "")

				if len(ch.History) == 0 {
					lines = append(lines, "  No title changes yet.")
				} else {
					lines = append(lines, fmt.Sprintf(" %-5s %-22s %-22s %s", "Week", "Winner", "Loser", "Method"))
					lines = append(lines, " -----------------------------------------------------------")
					for i := len(ch.History) - 1; i >= 0; i-- {
						tc := ch.History[i]
						winner := tc.Winner
						if winner == "" {
							winner = "(vacated)"
						}
						if len(winner) > 22 {
							winner = winner[:22]
						}
						loser := tc.Loser
						if len(loser) > 22 {
							loser = loser[:22]
						}
						lines = append(lines, fmt.Sprintf(" W%-4d %-22s %-22s %s",
							tc.Week, winner, loser, tc.Method))
					}
				}
			}
		}
	}

	h.lines = lines
}

func (h *CareerHistoryScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewCareerScreen(h.fed, h.save))
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if h.tab == HistoryTabMatches {
			h.tab = HistoryTabTitle
		} else {
			h.tab = HistoryTabMatches
		}
		h.scroll = 0
		h.buildLines()
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		if h.scroll > 0 {
			h.scroll--
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
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
