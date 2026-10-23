package ui

import (
	"reflect"
	"strings"
	"testing"

	"wrestling/internal/engine"
)

func standingsFed() *engine.Federation {
	return &engine.Federation{
		Championships: []engine.Championship{
			{Name: "World Title", Champion: "A Very Long Wrestler Name"},
			{Name: "Tag Titles"},
		},
		Records: map[string]*engine.WrestlerRecord{
			"A Very Long Wrestler Name": {Wins: 3, Losses: 1, CurrentStreak: 2, TitleReigns: 1},
			"Bruiser":                   {Wins: 1, Losses: 2, Draws: 1, CurrentStreak: -1},
			"Rookie":                    {},
		},
		Rivalries: map[string]map[string]int{
			"Bruiser": {"Rookie": 4},
			"Rookie":  {"Bruiser": 4, "A Very Long Wrestler Name": 1},
		},
	}
}

func TestStandingsLines(t *testing.T) {
	s := NewCareerStandingsScreen(standingsFed(), nil)
	want := []string{
		"============================================================",
		"                  STANDINGS & RECORDS",
		"============================================================",
		"",
		"  World Title: A Very Long Wrestler Name",
		"  Tag Titles: VACANT",
		"",
		" #   Name                      W    L    D  Streak  Titles",
		" -----------------------------------------------------------",
		" 1   A Very Long Wrestler N    3    1    0  W2      1",
		" 2   Bruiser                   1    2    1  L1      0",
		" 3   Rookie                    0    0    0          0",
		"",
		"ACTIVE RIVALRIES:",
		"",
		"  Bruiser vs Rookie (intensity: 4)",
	}
	if !reflect.DeepEqual(s.lines, want) {
		t.Fatalf("standings lines:\n got %q\nwant %q", s.lines, want)
	}
}

func TestStandingsWithoutRivalriesHasNoRivalrySection(t *testing.T) {
	fed := standingsFed()
	fed.Rivalries = nil
	s := NewCareerStandingsScreen(fed, nil)
	if last := s.lines[len(s.lines)-1]; last != " 3   Rookie                    0    0    0          0" {
		t.Fatalf("last line: got %q, want the last record row", last)
	}
}

func TestStandingsBreaksFullTiesOnName(t *testing.T) {
	fed := &engine.Federation{Records: map[string]*engine.WrestlerRecord{
		"Delta":   {Wins: 2, Losses: 1},
		"Alpha":   {Wins: 2, Losses: 1},
		"Charlie": {Wins: 2, Losses: 1},
		"Bravo":   {Wins: 2, Losses: 1},
	}}
	want := []string{"Alpha", "Bravo", "Charlie", "Delta"}
	// Map order changes from run to run, so check the order many times.
	for range 50 {
		s := NewCareerStandingsScreen(fed, nil)
		for i, name := range want {
			if got := s.lines[7+i]; !strings.Contains(got, " "+name+" ") {
				t.Fatalf("row %d: got %q, want %s", i+1, got, name)
			}
		}
	}
}

func TestStandingsBreaksEqualWinRateOnWins(t *testing.T) {
	fed := &engine.Federation{Records: map[string]*engine.WrestlerRecord{
		"Few":  {Wins: 1, Losses: 1},
		"Many": {Wins: 3, Losses: 3},
	}}
	s := NewCareerStandingsScreen(fed, nil)
	if got := s.lines[7]; got[:10] != " 1   Many " {
		t.Fatalf("first ranked row: got %q, want Many first", got)
	}
}
