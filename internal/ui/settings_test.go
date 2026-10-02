package ui

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
)

func (s *memStore) LoadSettingsJSON() ([]byte, error) { return s.settings, nil }
func (s *memStore) SaveSettingsJSON(data []byte) error {
	s.settings = data
	return nil
}

func TestSettingsScreenTogglesEachRuleAndSaves(t *testing.T) {
	g, in := testGame("A", "B")
	g.SetScreen(NewSettingsScreen())
	if g.Rules != engine.DefaultRules() {
		t.Fatalf("starting rules: %+v", g.Rules)
	}

	press(t, g, in, ebiten.KeyEnter)
	if g.Rules.ChartChoice {
		t.Fatal("ENTER on the first item should switch chart choice off")
	}
	if saved := string(g.Store.(*memStore).settings); !strings.Contains(saved, `"chart_choice": false`) {
		t.Fatalf("settings not saved: %s", saved)
	}

	press(t, g, in, ebiten.KeyDown)
	press(t, g, in, ebiten.KeySpace)
	if g.Rules.AvoidDisMoves {
		t.Fatal("SPACE on the second item should switch dis avoidance off")
	}

	press(t, g, in, ebiten.KeyDown)
	press(t, g, in, ebiten.KeyEnter)
	if g.Rules.CustomDisNumbers {
		t.Fatal("ENTER on the third item should switch custom dis numbers off")
	}
	press(t, g, in, ebiten.KeyEnter)
	if !g.Rules.CustomDisNumbers {
		t.Fatal("a second ENTER should switch it back on")
	}
}

func TestSettingsScreenEscapeReturnsToMenu(t *testing.T) {
	g, in := testGame("A", "B")
	g.SetScreen(NewSettingsScreen())
	press(t, g, in, ebiten.KeyEscape)
	if _, ok := g.screen.(*MenuScreen); !ok {
		t.Fatalf("screen is %T, want *MenuScreen", g.screen)
	}
}

func TestMainMenuOpensSettings(t *testing.T) {
	g, in := testGame("A", "B")
	for i := 0; i < menuSettings; i++ {
		press(t, g, in, ebiten.KeyDown)
	}
	press(t, g, in, ebiten.KeyEnter)
	if _, ok := g.screen.(*SettingsScreen); !ok {
		t.Fatalf("screen is %T, want *SettingsScreen", g.screen)
	}
}

func TestNewGameLoadsSavedRules(t *testing.T) {
	store := newMemStore()
	store.settings = []byte(`{"chart_choice": false}`)
	g := NewGame(nil, store)
	want := engine.Rules{ChartChoice: false, AvoidDisMoves: true, CustomDisNumbers: true}
	if g.Rules != want {
		t.Fatalf("rules: %+v, want %+v", g.Rules, want)
	}
}

func TestNewGameWarnsAboutUnreadableSettings(t *testing.T) {
	store := newMemStore()
	store.settings = []byte(`{broken`)
	g := NewGame(nil, store)
	if !strings.Contains(g.notice, "settings") {
		t.Fatalf("notice: %q", g.notice)
	}
	if g.Rules != engine.DefaultRules() {
		t.Fatalf("rules should fall back to defaults, got %+v", g.Rules)
	}
}

func TestEveryMatchUsesTheGameRules(t *testing.T) {
	g, _ := testGame("A", "B", "C", "D")
	g.Rules = engine.Rules{AvoidDisMoves: true}
	a, b, c, d := g.Roster[0], g.Roster[1], g.Roster[2], g.Roster[3]

	if got := NewMatchScreen(a, b, engine.MatchSingles, g).match.Rules; got != g.Rules {
		t.Errorf("singles match rules: %+v", got)
	}
	if got := NewTagMatchScreen(engine.NewTagMatch(a, b, c, d), g).match.Rules; got != g.Rules {
		t.Errorf("tag match rules: %+v", got)
	}

	br := NewBattleRoyalScreen(g.Roster, g)
	br.champion = br.wrestlers[0]
	br.startNextMatch(g)
	if br.match.Rules != g.Rules {
		t.Errorf("battle royal match rules: %+v", br.match.Rules)
	}

	ts := newBracket(4)
	copy(ts.seeds, g.Roster)
	ts.startCurrentMatch(g)
	if ts.match.Rules != g.Rules {
		t.Errorf("tournament match rules: %+v", ts.match.Rules)
	}
}
