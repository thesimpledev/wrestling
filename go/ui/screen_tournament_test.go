package ui

import (
	"fmt"
	"strings"
	"testing"

	"wrestling/engine"
)

func fakeCard(name string) *engine.WrestlerCard {
	return &engine.WrestlerCard{Name: name}
}

func newBracket(size int) *TournamentScreen {
	t := &TournamentScreen{bracketSize: size}
	for size > 1 {
		t.totalRounds++
		size /= 2
	}
	t.seeds = make([]*engine.WrestlerCard, t.bracketSize)
	t.results = make([][]*engine.WrestlerCard, t.totalRounds)
	for r := 0; r < t.totalRounds; r++ {
		t.results[r] = make([]*engine.WrestlerCard, t.bracketSize/(1<<(r+1)))
	}
	return t
}

func TestBracketAdvancement(tt *testing.T) {
	for _, size := range []int{4, 8, 16} {
		ts := newBracket(size)
		for i := range ts.seeds {
			ts.seeds[i] = fakeCard(fmt.Sprintf("W%02d", i))
		}
		// Always advance the first wrestler of each matchup
		for r := 0; r < ts.totalRounds; r++ {
			for m := 0; m < size/(1<<(r+1)); m++ {
				w1, w2 := ts.getMatchup(r, m)
				if w1 == nil || w2 == nil {
					tt.Fatalf("size %d round %d match %d: nil matchup", size, r, m)
				}
				if w1 == w2 {
					tt.Fatalf("size %d round %d match %d: wrestler faces themselves", size, r, m)
				}
				ts.results[r][m] = w1
			}
		}
		champ := ts.results[ts.totalRounds-1][0]
		if champ == nil || champ.Name != "W00" {
			tt.Fatalf("size %d: wrong champion %v", size, champ)
		}
	}
}

func TestConnectorAlignment(tt *testing.T) {
	ts := newBracket(8)
	// Round 1 (col 2) joins round-0 winner rows 1 and 5 (match 0), 9 and 13 (match 1)
	for _, row := range []int{1, 5, 9, 13} {
		if got := ts.getConnector(row, 2, 8); got != "---+" {
			tt.Errorf("col 2 row %d: got %q want ---+", row, got)
		}
	}
	if got := ts.getConnector(0, 2, 8); got != "" {
		tt.Errorf("col 2 row 0: got %q want empty", got)
	}
	// Round 2 (col 3) joins round-1 winner rows 3 and 11
	for _, row := range []int{3, 11} {
		if got := ts.getConnector(row, 3, 8); got != "---+" {
			tt.Errorf("col 3 row %d: got %q want ---+", row, got)
		}
	}
	if got := ts.getConnector(4, 3, 8); got != "   |" {
		tt.Errorf("col 3 row 4: got %q want |", got)
	}
	// Round 0 (col 1) joins seed rows 0 and 2
	for _, row := range []int{0, 2} {
		if got := ts.getConnector(row, 1, 8); got != "---+" {
			tt.Errorf("col 1 row %d: got %q want ---+", row, got)
		}
	}
}

func TestRenderBracketNoPanic(tt *testing.T) {
	for _, size := range []int{4, 8, 16} {
		ts := newBracket(size)
		for i := range ts.seeds {
			ts.seeds[i] = fakeCard(fmt.Sprintf("Wrestler %d", i))
		}
		lines := ts.renderBracket()
		if len(lines) == 0 {
			tt.Fatalf("size %d: empty bracket render", size)
		}
		for _, l := range lines {
			if strings.TrimSpace(l) == "" {
				tt.Fatalf("size %d: blank line survived filter", size)
			}
		}
	}
}

func TestAutoFillNoDuplicates(tt *testing.T) {
	roster := make([]*engine.WrestlerCard, 12)
	for i := range roster {
		roster[i] = fakeCard(fmt.Sprintf("R%02d", i))
	}
	g := &Game{Roster: roster}
	ts := newBracket(8)
	ts.roster = roster
	ts.seeds[0] = roster[3] // one manual pick
	ts.autoFillSeeds(g)
	seen := map[string]bool{}
	for i, s := range ts.seeds {
		if s == nil {
			tt.Fatalf("seed %d still empty", i)
		}
		if seen[s.Name] {
			tt.Fatalf("duplicate seed %s", s.Name)
		}
		seen[s.Name] = true
	}
}

func TestAvailableSizes(tt *testing.T) {
	ts := &TournamentScreen{}
	ts.roster = make([]*engine.WrestlerCard, 10)
	got := ts.availableSizes()
	if len(got) != 2 || got[0] != 4 || got[1] != 8 {
		tt.Fatalf("roster 10: got %v want [4 8]", got)
	}
	ts.roster = ts.roster[:3]
	got = ts.availableSizes()
	if len(got) != 1 || got[0] != 4 {
		tt.Fatalf("roster 3: got %v want [4]", got)
	}
}
