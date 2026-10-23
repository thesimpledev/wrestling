package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"wrestling/internal/engine"
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
		for r := 0; r < ts.totalRounds; r++ {
			advanceFirstSeeds(tt, ts, r)
		}
		champ := ts.results[ts.totalRounds-1][0]
		if champ == nil || champ.Name != "W00" {
			tt.Fatalf("size %d: wrong champion %v", size, champ)
		}
	}
}

// advanceFirstSeeds plays a round in which the first wrestler of every
// matchup wins, checking each matchup has two different wrestlers.
func advanceFirstSeeds(tt *testing.T, ts *TournamentScreen, round int) {
	tt.Helper()
	for m := range ts.results[round] {
		w1, w2 := ts.getMatchup(round, m)
		if w1 == nil || w2 == nil {
			tt.Fatalf("size %d round %d match %d: nil matchup", ts.bracketSize, round, m)
		}
		if w1 == w2 {
			tt.Fatalf("size %d round %d match %d: wrestler faces themselves", ts.bracketSize, round, m)
		}
		ts.results[round][m] = w1
	}
}

func TestRenderBracketLayout(tt *testing.T) {
	ts := newBracket(8)
	for i := range ts.seeds {
		ts.seeds[i] = fakeCard(fmt.Sprintf("Wrestler Number %d Long", i))
	}
	ts.seeds[5] = nil
	ts.results[0][0] = ts.seeds[1]
	ts.results[0][1] = ts.seeds[2]
	ts.results[1][0] = ts.seeds[2]
	ts.currentMatch = 2
	want := []string{
		"  [ 1] Wrestler Numbe.  ---+                                                              ",
		"                         Wrestler Number 1.   ---+                                        ",
		"  [ 2] Wrestler Numbe.  ---+                     |                                        ",
		"                                               Wrestler Number 2.   ---+                  ",
		"  [ 3] Wrestler Numbe.  ---+                     |                     |                  ",
		"                         Wrestler Number 2.   ---+                     |                  ",
		"  [ 4] Wrestler Numbe.  ---+                                           |                  ",
		"                                                                     ???                  ",
		"  [ 5] Wrestler Numbe.  ---+                                           |                  ",
		"                        >???                  ---+                     |                  ",
		"  [ 6] ???              ---+                     |                     |                  ",
		"                                               ???                  ---+                  ",
		"  [ 7] Wrestler Numbe.  ---+                     |                                        ",
		"                         ???                  ---+                                        ",
		"  [ 8] Wrestler Numbe.  ---+                                                              ",
	}
	if got := ts.renderBracket(); !reflect.DeepEqual(got, want) {
		tt.Fatalf("bracket:\n got %q\nwant %q", got, want)
	}
}

func TestFillBracketLines(tt *testing.T) {
	ts := newBracket(8)
	ts.roster = []*engine.WrestlerCard{fakeCard("A"), fakeCard("B"), fakeCard("C")}
	ts.seeds[0] = ts.roster[1]
	ts.fillCursor = 1
	ts.rosterCursor = 1
	want := []string{
		"============================================================",
		"                  FILL TOURNAMENT BRACKET",
		"============================================================",
		"",
		"Filling seed 2 of 8:",
		"     [ 1] B",
		"  >> [ 2] (empty)",
		"     [ 3] (empty)",
		"     [ 4] (empty)",
		"     [ 5] (empty)",
		"     [ 6] (empty)",
		"     ...",
		"",
		"SELECT WRESTLER:",
		"  A",
		"> B  (already seeded)",
		"  C",
		"  [Auto-Fill Remaining]",
	}
	if got := ts.fillBracketLines(600); !reflect.DeepEqual(got, want) {
		tt.Fatalf("fill lines:\n got %q\nwant %q", got, want)
	}
}

func TestFillBracketLinesScrollWindows(tt *testing.T) {
	ts := newBracket(8)
	ts.roster = []*engine.WrestlerCard{fakeCard("A"), fakeCard("B"), fakeCard("C")}
	ts.fillCursor = 7
	ts.rosterCursor = 3
	// Room for exactly two roster rows below the 14 lines above them.
	screenH := 17*LineHeight + 2*Margin
	got := ts.fillBracketLines(screenH)
	if got[5] != "     ..." || got[6] != "     [ 3] (empty)" || got[11] != "  >> [ 8] (empty)" {
		tt.Fatalf("seed window: got %q", got[5:12])
	}
	tail := got[len(got)-3:]
	want := []string{"SELECT WRESTLER:", "  C", "> [Auto-Fill Remaining]"}
	if !reflect.DeepEqual(tail, want) {
		tt.Fatalf("roster window: got %q want %q", tail, want)
	}
}

func TestFillBracketLinesShowsOneRosterRowWithNoRoom(tt *testing.T) {
	ts := newBracket(4)
	ts.roster = []*engine.WrestlerCard{fakeCard("A"), fakeCard("B")}
	ts.rosterCursor = 1
	got := ts.fillBracketLines(0)
	if last := got[len(got)-1]; last != "> B" || got[len(got)-2] != "SELECT WRESTLER:" {
		tt.Fatalf("got %q, want only the cursor row", got)
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
