package ui

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
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
	lines := []string{
		"============================================================",
		"                  STANDINGS & RECORDS",
		"============================================================",
		"",
	}
	lines = append(lines, s.championshipLines()...)
	lines = append(lines, "")
	lines = append(lines, s.recordLines()...)
	lines = append(lines, s.rivalryLines()...)
	s.lines = lines
}

func (s *CareerStandingsScreen) championshipLines() []string {
	lines := make([]string, 0, len(s.fed.Championships))
	for _, ch := range s.fed.Championships {
		holder := ch.Champion
		if holder == "" {
			holder = "VACANT"
		}
		lines = append(lines, fmt.Sprintf("  %s: %s", ch.Name, holder))
	}
	return lines
}

type standingsEntry struct {
	name string
	rec  *engine.WrestlerRecord
	pct  float64
}

// rankedRecords orders wrestlers by win rate, then by number of wins.
func (s *CareerStandingsScreen) rankedRecords() []standingsEntry {
	entries := make([]standingsEntry, 0, len(s.fed.Records))
	for name, rec := range s.fed.Records {
		total := rec.Wins + rec.Losses + rec.Draws
		pct := 0.0
		if total > 0 {
			pct = float64(rec.Wins) / float64(total)
		}
		entries = append(entries, standingsEntry{name, rec, pct})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].pct != entries[j].pct {
			return entries[i].pct > entries[j].pct
		}
		return entries[i].rec.Wins > entries[j].rec.Wins
	})
	return entries
}

// recordLines is the ranked table of win-loss records, with current
// champions marked (c).
func (s *CareerStandingsScreen) recordLines() []string {
	champions := make(map[string]bool)
	for _, ch := range s.fed.Championships {
		if ch.Champion != "" {
			champions[ch.Champion] = true
		}
	}

	lines := []string{
		fmt.Sprintf(" %-3s %-22s %4s %4s %4s  %-7s %s", "#", "Name", "W", "L", "D", "Streak", "Titles"),
		" -----------------------------------------------------------",
	}
	for i, e := range s.rankedRecords() {
		name := e.name
		if champions[e.name] {
			name += " (c)"
		}
		if len(name) > 22 {
			name = name[:22]
		}
		lines = append(lines, fmt.Sprintf(" %-3d %-22s %4d %4d %4d  %-7s %d",
			i+1, name, e.rec.Wins, e.rec.Losses, e.rec.Draws, streakLabel(e.rec.CurrentStreak), e.rec.TitleReigns))
	}
	return lines
}

// streakLabel shows a winning streak as W<n> and a losing one as L<n>.
func streakLabel(streak int) string {
	switch {
	case streak > 0:
		return fmt.Sprintf("W%d", streak)
	case streak < 0:
		return fmt.Sprintf("L%d", -streak)
	default:
		return ""
	}
}

func (s *CareerStandingsScreen) rivalryLines() []string {
	rivals := s.fed.ActiveRivals()
	if len(rivals) == 0 {
		return nil
	}
	lines := []string{"", "ACTIVE RIVALRIES:", ""}
	for _, pair := range rivals {
		score := s.fed.RivalryScore(pair[0], pair[1])
		lines = append(lines, fmt.Sprintf("  %s vs %s (intensity: %d)", pair[0], pair[1], score))
	}
	return lines
}

func (s *CareerStandingsScreen) Update(g *Game) error {
	if g.in.JustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewCareerScreen(s.fed, s.save))
		return nil
	}

	if g.in.Pressed(ebiten.KeyUp) {
		if s.scroll > 0 {
			s.scroll--
		}
	}
	if g.in.Pressed(ebiten.KeyDown) {
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
