package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
	"wrestling/internal/storage"
)

type fakeInput struct {
	just  map[ebiten.Key]bool
	held  map[ebiten.Key]bool
	chars []rune
}

func (f *fakeInput) JustPressed(key ebiten.Key) bool { return f.just[key] }
func (f *fakeInput) Pressed(key ebiten.Key) bool     { return f.held[key] || f.just[key] }
func (f *fakeInput) Chars() []rune                   { return f.chars }
func (f *fakeInput) AnyJustPressed() bool            { return len(f.just) > 0 }

func (f *fakeInput) set(keys []ebiten.Key, chars string) {
	f.just = make(map[ebiten.Key]bool, len(keys))
	for _, k := range keys {
		f.just[k] = true
	}
	f.chars = []rune(chars)
}

type memStore struct {
	cards    map[string][]byte
	injuries []byte
	career   []byte
	settings []byte
}

func newMemStore() *memStore {
	return &memStore{cards: map[string][]byte{}}
}

func (s *memStore) LoadAllCardBytes() (map[string][]byte, error) { return s.cards, nil }
func (s *memStore) SaveCardBytes(name string, data []byte) error {
	s.cards[name] = data
	return nil
}
func (s *memStore) LoadInjuriesJSON() ([]byte, error) { return s.injuries, nil }
func (s *memStore) SaveInjuriesJSON(data []byte) error {
	s.injuries = data
	return nil
}
func (s *memStore) LoadCareerJSON() ([]byte, error) { return s.career, nil }
func (s *memStore) SaveCareerJSON(data []byte) error {
	s.career = data
	return nil
}

var _ storage.Store = (*memStore)(nil)

// testGame builds a Game on an in-memory store with a roster of named cards.
func testGame(names ...string) (*Game, *fakeInput) {
	roster := make([]*engine.WrestlerCard, len(names))
	for i, n := range names {
		roster[i] = fakeCard(n)
	}
	in := &fakeInput{}
	g := NewGame(roster, newMemStore())
	g.in = in
	return g, in
}

// press runs one game tick with the given keys just pressed.
func press(t *testing.T, g *Game, in *fakeInput, keys ...ebiten.Key) {
	t.Helper()
	in.set(keys, "")
	if err := g.Update(); err != nil {
		t.Fatalf("Update returned %v", err)
	}
	in.set(nil, "")
}

// typeText runs one game tick per character, as a keyboard would deliver them.
func typeText(t *testing.T, g *Game, in *fakeInput, text string) {
	t.Helper()
	for _, r := range text {
		in.set(nil, string(r))
		if err := g.Update(); err != nil {
			t.Fatalf("Update returned %v", err)
		}
	}
	in.set(nil, "")
}

func TestHandleListInputMovesAndWraps(t *testing.T) {
	in := &fakeInput{}
	in.set([]ebiten.Key{ebiten.KeyDown}, "")
	if got := handleListInput(in, 0, 3); got != 1 {
		t.Fatalf("down from 0: got %d want 1", got)
	}
	if got := handleListInput(in, 2, 3); got != 0 {
		t.Fatalf("down from last: got %d want 0", got)
	}
	in.set([]ebiten.Key{ebiten.KeyUp}, "")
	if got := handleListInput(in, 0, 3); got != 2 {
		t.Fatalf("up from 0: got %d want 2", got)
	}
}

func TestMenuCursorFollowsInput(t *testing.T) {
	g, in := testGame("A", "B")
	press(t, g, in, ebiten.KeyDown)
	menu, ok := g.screen.(*MenuScreen)
	if !ok {
		t.Fatalf("screen is %T, want *MenuScreen", g.screen)
	}
	if menu.cursor != 1 {
		t.Fatalf("cursor: got %d want 1", menu.cursor)
	}
}

func TestTextInputTakesCharsAndBackspace(t *testing.T) {
	in := &fakeInput{}
	ti := NewTextInput(3)
	in.set(nil, "abcd")
	ti.Update(in)
	if ti.Text != "abc" {
		t.Fatalf("text: got %q want %q (max length 3)", ti.Text, "abc")
	}
	in.set([]ebiten.Key{ebiten.KeyBackspace}, "")
	ti.Update(in)
	if ti.Text != "ab" {
		t.Fatalf("after backspace: got %q want %q", ti.Text, "ab")
	}
}

func TestNoticeClearsOnAnyKey(t *testing.T) {
	g, in := testGame("A", "B")
	g.SetNotice("something failed")
	press(t, g, in, ebiten.KeyDown)
	if g.notice != "" {
		t.Fatalf("notice still set: %q", g.notice)
	}
}
