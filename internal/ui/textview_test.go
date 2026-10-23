package ui

import (
	"reflect"
	"testing"
)

func letterLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = string(rune('a' + i))
	}
	return lines
}

func TestFittingLinesStopsAboveTheStatusBar(t *testing.T) {
	// Five full rows between the top margin and the status bar.
	screenH := 2*Margin + 6*LineHeight
	got := fittingLines(letterLines(10), 2, screenH)
	want := []string{"c", "d", "e", "f", "g"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFittingLinesLeavesNoPartRowUnderTheStatusBar(t *testing.T) {
	// Room for four and a half rows: the half row stays empty.
	screenH := 2*Margin + 5*LineHeight + LineHeight/2
	if got := fittingLines(letterLines(10), 0, screenH); len(got) != 4 {
		t.Fatalf("got %d lines %q, want 4", len(got), got)
	}
}

func TestFittingLinesClampsTheStart(t *testing.T) {
	screenH := 2*Margin + 6*LineHeight
	if got := fittingLines(letterLines(3), 7, screenH); !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("start past the end: got %q, want the last line", got)
	}
	if got := fittingLines(letterLines(3), -4, screenH); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("negative start: got %q, want every line", got)
	}
	if got := fittingLines(nil, 0, screenH); len(got) != 0 {
		t.Fatalf("no lines: got %q", got)
	}
}

func TestFittingLinesWithNoRoom(t *testing.T) {
	if got := fittingLines(letterLines(3), 0, 0); len(got) != 0 {
		t.Fatalf("zero-height screen: got %q, want nothing", got)
	}
}
