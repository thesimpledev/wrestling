package ui

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
)

type CareerStandingsScreen struct {
	fed    *engine.Federation
	save   *engine.FederationSave
	scroll int
	lines  []string
}

func NewCareerStandingsScreen(fed *engine.Federation, save *engine.FederationSave) *CareerStandingsScreen {
	s := &CareerStandingsScreen{fed: fed, save: save}
	s.buildLines()
	return s
}

func (s *CareerStandingsScreen) buildLines() {
	var lines []string
	lines = append(lines, "============================================================")
	lines = append(lines, "                  STANDINGS & RECORDS")
	lines = append(lines, "============================================================")
	lines = append(lines, "")

	// Show all championships
	for _, ch := range s.fed.Championships {
		if ch.Champion == "" {
			lines = append(lines, fmt.Sprintf("  %s: VACANT", ch.Name))
		} else {
			lines = append(lines, fmt.Sprintf("  %s: %s", ch.Name, ch.Champion))
		}
	}
	lines = append(lines, "")

	// Collect all champion names for marking
	champSet := make(map[string]bool)
	for _, ch := range s.fed.Championships {
		if ch.Champion != "" {
			champSet[ch.Champion] = true
		}
	}

	// Build ranked list
	type entry struct {
		name string
		rec  *engine.WrestlerRecord
		pct  float64
	}
	var entries []entry
	for name, rec := range s.fed.Records {
		total := rec.Wins + rec.Losses + rec.Draws
		pct := 0.0
		if total > 0 {
			pct = float64(rec.Wins) / float64(total)
		}
		entries = append(entries, entry{name, rec, pct})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].pct != entries[j].pct {
			return entries[i].pct > entries[j].pct
		}
		return entries[i].rec.Wins > entries[j].rec.Wins
	})

	lines = append(lines, fmt.Sprintf(" %-3s %-22s %4s %4s %4s  %-7s %s",
		"#", "Name", "W", "L", "D", "Streak", "Titles"))
	lines = append(lines, " -----------------------------------------------------------")

	for i, e := range entries {
		nameStr := e.name
		if champSet[e.name] {
			nameStr += " (c)"
		}
		if len(nameStr) > 22 {
			nameStr = nameStr[:22]
		}

		streak := ""
		if e.rec.CurrentStreak > 0 {
			streak = fmt.Sprintf("W%d", e.rec.CurrentStreak)
		} else if e.rec.CurrentStreak < 0 {
			streak = fmt.Sprintf("L%d", -e.rec.CurrentStreak)
		}

		lines = append(lines, fmt.Sprintf(" %-3d %-22s %4d %4d %4d  %-7s %d",
			i+1, nameStr, e.rec.Wins, e.rec.Losses, e.rec.Draws, streak, e.rec.TitleReigns))
	}

	// Active rivalries
	rivals := s.fed.ActiveRivals()
	if len(rivals) > 0 {
		lines = append(lines, "")
		lines = append(lines, "ACTIVE RIVALRIES:")
		lines = append(lines, "")
		for _, pair := range rivals {
			score := s.fed.RivalryScore(pair[0], pair[1])
			lines = append(lines, fmt.Sprintf("  %s vs %s (intensity: %d)", pair[0], pair[1], score))
		}
	}

	s.lines = lines
}

func (s *CareerStandingsScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewCareerScreen(s.fed, s.save))
		return nil
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		if s.scroll > 0 {
			s.scroll--
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		max := len(s.lines) - s.visibleLines(g)
		if max < 0 {
			max = 0
		}
		if s.scroll < max {
			s.scroll++
		}
	}

	return nil
}

func (s *CareerStandingsScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)

	statusBarY := g.screenH - LineHeight - Margin

	startLine := s.scroll
	if startLine >= len(s.lines) {
		startLine = len(s.lines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	y := Margin
	for i := startLine; i < len(s.lines) && y < statusBarY; i++ {
		DrawText(screen, s.lines[i], Margin, y)
		y += LineHeight
	}

	DrawText(screen, "[UP/DOWN] Scroll  [ESC] Back", Margin, statusBarY)
}

func (s *CareerStandingsScreen) visibleLines(g *Game) int {
	if g.screenH == 0 {
		return 20
	}
	return (g.screenH - Margin*2 - LineHeight) / LineHeight
}
