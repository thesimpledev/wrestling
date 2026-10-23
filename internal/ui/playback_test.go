package ui

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
)

// playbackView points into the match playback fields of a screen, so the
// same tests cover every screen that steps through match events.
type playbackView struct {
	autoPlay *bool
	speed    *int
	ticker   *int
	scroll   *int
	shown    *int
	lines    *[]string
}

type playbackScreen struct {
	name string
	// build puts a screen in its running-match phase with the given events.
	build func(g *Game, events []engine.Event) playbackView
}

var playbackScreens = []playbackScreen{
	{"tournament", func(g *Game, events []engine.Event) playbackView {
		ts := newBracket(4)
		ts.speed = 30
		ts.phase = TournRunningMatch
		ts.events = events
		g.SetScreen(ts)
		return playbackView{&ts.autoPlay, &ts.speed, &ts.ticker, &ts.scroll, &ts.shown, &ts.lines}
	}},
	{"battle royal", func(g *Game, events []engine.Event) playbackView {
		br := NewBattleRoyalScreen(g.Roster, g)
		br.phase = BRRunningMatch
		br.events = events
		g.SetScreen(br)
		return playbackView{&br.autoPlay, &br.speed, &br.ticker, &br.scroll, &br.shown, &br.lines}
	}},
}

func fakeEvents(n int) []engine.Event {
	events := make([]engine.Event, n)
	for i := range events {
		events[i] = engine.Event{Text: fmt.Sprintf("event %d", i)}
	}
	return events
}

func TestPlaybackAutoPlayToggle(t *testing.T) {
	for _, ps := range playbackScreens {
		g, in := testGame("A", "B")
		v := ps.build(g, fakeEvents(5))
		press(t, g, in, ebiten.KeyA)
		if !*v.autoPlay {
			t.Errorf("%s: A did not turn auto-play on", ps.name)
		}
		press(t, g, in, ebiten.KeyA)
		if *v.autoPlay {
			t.Errorf("%s: A did not turn auto-play off", ps.name)
		}
	}
}

func TestPlaybackSpeedKeys(t *testing.T) {
	cases := []struct {
		key        ebiten.Key
		from, want int
	}{
		{ebiten.KeyEqual, 30, 25},
		{ebiten.KeyNumpadAdd, 30, 25},
		{ebiten.KeyEqual, 10, 5},
		{ebiten.KeyEqual, 5, 5},
		{ebiten.KeyMinus, 30, 35},
		{ebiten.KeyNumpadSubtract, 5, 10},
	}
	for _, ps := range playbackScreens {
		for _, c := range cases {
			g, in := testGame("A", "B")
			v := ps.build(g, fakeEvents(5))
			*v.speed = c.from
			press(t, g, in, c.key)
			if *v.speed != c.want {
				t.Errorf("%s: key %v from %d: got %d want %d", ps.name, c.key, c.from, *v.speed, c.want)
			}
		}
	}
}

func TestPlaybackScrollStaysInRange(t *testing.T) {
	// With no screen size the view holds 20 lines, so 30 lines scroll to 10.
	cases := []struct {
		key        ebiten.Key
		from, want int
	}{
		{ebiten.KeyUp, 0, 0},
		{ebiten.KeyUp, 4, 3},
		{ebiten.KeyDown, 0, 1},
		{ebiten.KeyDown, 10, 10},
	}
	for _, ps := range playbackScreens {
		for _, c := range cases {
			g, in := testGame("A", "B")
			v := ps.build(g, fakeEvents(5))
			*v.lines = make([]string, 30)
			*v.scroll = c.from
			press(t, g, in, c.key)
			if *v.scroll != c.want {
				t.Errorf("%s: key %v from %d: got %d want %d", ps.name, c.key, c.from, *v.scroll, c.want)
			}
		}
	}
}

func TestPlaybackStepShowsNextEvent(t *testing.T) {
	for _, ps := range playbackScreens {
		for _, key := range []ebiten.Key{ebiten.KeySpace, ebiten.KeyEnter} {
			g, in := testGame("A", "B")
			v := ps.build(g, fakeEvents(5))
			*v.lines = make([]string, 25)
			press(t, g, in, key)
			lines := *v.lines
			if *v.shown != 1 || lines[len(lines)-1] != "event 0" {
				t.Errorf("%s: key %v: shown %d, last line %q", ps.name, key, *v.shown, lines[len(lines)-1])
			}
			if *v.scroll != 6 {
				t.Errorf("%s: key %v: scroll %d, want 6 (bottom)", ps.name, key, *v.scroll)
			}
		}
	}
}

func TestPlaybackAutoPlayStepsOnTimer(t *testing.T) {
	for _, ps := range playbackScreens {
		g, in := testGame("A", "B")
		v := ps.build(g, fakeEvents(5))
		*v.autoPlay = true
		*v.speed = 3
		press(t, g, in)
		press(t, g, in)
		if *v.shown != 0 {
			t.Fatalf("%s: event shown before the timer ran out", ps.name)
		}
		press(t, g, in)
		if *v.shown != 1 || *v.ticker != 0 {
			t.Errorf("%s: after 3 ticks: shown %d ticker %d, want 1 and 0", ps.name, *v.shown, *v.ticker)
		}
	}
}

func TestTournamentLastEventFinishesMatch(t *testing.T) {
	g, in := testGame("A", "B", "C", "D")
	ts := newBracket(4)
	copy(ts.seeds, g.Roster)
	ts.startCurrentMatch(g)
	g.SetScreen(ts)
	for range ts.events {
		press(t, g, in, ebiten.KeySpace)
	}
	if ts.phase != TournMatchResult || ts.results[0][0] == nil {
		t.Fatalf("phase %v winner %v: match did not finish", ts.phase, ts.results[0][0])
	}
}

func TestBattleRoyalLastEventFinishesMatch(t *testing.T) {
	g, in := testGame("A", "B", "C")
	br := NewBattleRoyalScreen(g.Roster, g)
	br.champion = br.wrestlers[0]
	br.startNextMatch(g)
	g.SetScreen(br)
	for range br.events {
		press(t, g, in, ebiten.KeySpace)
	}
	if br.phase != BRMatchResult || br.nextIdx != 2 {
		t.Fatalf("phase %v next %d: match did not finish", br.phase, br.nextIdx)
	}
}

func TestAdjustSpeed(t *testing.T) {
	in := &fakeInput{}
	in.set([]ebiten.Key{ebiten.KeyEqual}, "")
	if got := adjustSpeed(in, 7); got != 2 {
		t.Errorf("faster from 7: got %d want 2", got)
	}
	in.set([]ebiten.Key{ebiten.KeyMinus}, "")
	if got := adjustSpeed(in, 7); got != 12 {
		t.Errorf("slower from 7: got %d want 12", got)
	}
	in.set(nil, "")
	if got := adjustSpeed(in, 7); got != 7 {
		t.Errorf("no key: got %d want 7", got)
	}
}

func TestScrollWithKeys(t *testing.T) {
	in := &fakeInput{held: map[ebiten.Key]bool{ebiten.KeyUp: true, ebiten.KeyDown: true}}
	if got := scrollWithKeys(in, 0, 5); got != 1 {
		t.Errorf("both held from 0: got %d want 1", got)
	}
	in.held = nil
	if got := scrollWithKeys(in, 3, 5); got != 3 {
		t.Errorf("no key: got %d want 3", got)
	}
}

func TestStepDue(t *testing.T) {
	in := &fakeInput{}
	if due, ticker := stepDue(in, false, 0, 3); due || ticker != 0 {
		t.Errorf("idle: got %v %d", due, ticker)
	}
	if due, ticker := stepDue(in, true, 1, 3); due || ticker != 2 {
		t.Errorf("auto mid-wait: got %v %d want false 2", due, ticker)
	}
	if due, ticker := stepDue(in, true, 2, 3); !due || ticker != 0 {
		t.Errorf("auto timer done: got %v %d want true 0", due, ticker)
	}
	in.set([]ebiten.Key{ebiten.KeySpace}, "")
	if due, _ := stepDue(in, false, 0, 3); !due {
		t.Error("space did not step")
	}
}
