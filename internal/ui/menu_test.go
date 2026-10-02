package ui

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
)

func down(t *testing.T, g *Game, in *fakeInput, times int) {
	t.Helper()
	for i := 0; i < times; i++ {
		press(t, g, in, ebiten.KeyDown)
	}
}

func enter(t *testing.T, g *Game, in *fakeInput) {
	t.Helper()
	press(t, g, in, ebiten.KeyEnter)
}

// openMenuOption moves from the top of the main menu to an option and selects it.
func openMenuOption(t *testing.T, g *Game, in *fakeInput, option int) {
	t.Helper()
	down(t, g, in, option)
	enter(t, g, in)
}

func matchScreen(t *testing.T, g *Game) *MatchScreen {
	t.Helper()
	ms, ok := g.screen.(*MatchScreen)
	if !ok {
		t.Fatalf("screen is %T, want *MatchScreen", g.screen)
	}
	return ms
}

func sideNames(side *engine.Side) []string {
	names := make([]string, len(side.Wrestlers))
	for i, w := range side.Wrestlers {
		names[i] = w.Card.Name
	}
	return names
}

func TestSinglesStyleMenuOptionsStartTheMatchTheyName(t *testing.T) {
	cases := []struct {
		name     string
		option   int
		wantType engine.MatchType
		wantFeud bool
	}{
		{"Singles Match", menuSingles, engine.MatchSingles, false},
		{"Cage Match", menuCage, engine.MatchCage, false},
		{"No DQ Match", menuNoDQ, engine.MatchNoDQ, false},
		{"Feud Match", menuFeud, engine.MatchSingles, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if menuOptions[tc.option] != tc.name {
				t.Fatalf("menu option %d is %q, want %q", tc.option, menuOptions[tc.option], tc.name)
			}
			g, in := testGame("A", "B", "C")
			openMenuOption(t, g, in, tc.option)
			enter(t, g, in) // wrestler 1: A
			enter(t, g, in) // wrestler 2: B (cursor skips A)
			enter(t, g, in) // no ally
			enter(t, g, in) // no ally

			ms := matchScreen(t, g)
			if ms.match.Type != tc.wantType || ms.match.IsFeud != tc.wantFeud {
				t.Fatalf("started type %d feud %v, want type %d feud %v",
					ms.match.Type, ms.match.IsFeud, tc.wantType, tc.wantFeud)
			}
			if got := sideNames(ms.match.Sides[0]); !reflect.DeepEqual(got, []string{"A"}) {
				t.Errorf("side 1: %v", got)
			}
			if got := sideNames(ms.match.Sides[1]); !reflect.DeepEqual(got, []string{"B"}) {
				t.Errorf("side 2: %v", got)
			}
		})
	}
}

func TestTagTeamMenuOptionStartsATagMatchAndAsksAboutPartners(t *testing.T) {
	if menuOptions[menuTag] != "Tag Team Match" {
		t.Fatalf("menu option %d is %q", menuTag, menuOptions[menuTag])
	}
	g, in := testGame("A", "B", "C", "D")
	openMenuOption(t, g, in, menuTag)
	for i := 0; i < 4; i++ {
		enter(t, g, in)
	}
	enter(t, g, in) // team 1: regular partners (first answer is yes)
	down(t, g, in, 1)
	enter(t, g, in) // team 2: not regular partners

	ms := matchScreen(t, g)
	if ms.match.Type != engine.MatchTag {
		t.Fatalf("match type %d, want tag", ms.match.Type)
	}
	if got := sideNames(ms.match.Sides[0]); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("team 1: %v", got)
	}
	if got := sideNames(ms.match.Sides[1]); !reflect.DeepEqual(got, []string{"C", "D"}) {
		t.Errorf("team 2: %v", got)
	}
	if !ms.match.Sides[0].RegularPartners || ms.match.Sides[1].RegularPartners {
		t.Errorf("regular partners: team 1 %v team 2 %v, want true and false",
			ms.match.Sides[0].RegularPartners, ms.match.Sides[1].RegularPartners)
	}
}

func TestSameWrestlerCannotBePickedTwice(t *testing.T) {
	g, in := testGame("A", "B", "C")
	openMenuOption(t, g, in, menuSingles)
	enter(t, g, in) // A
	menu := g.screen.(*MenuScreen)
	if menu.cursor != 1 {
		t.Fatalf("cursor should start on the first free wrestler, got %d", menu.cursor)
	}
	press(t, g, in, ebiten.KeyUp) // back onto A
	enter(t, g, in)
	if len(menu.answers) != 1 {
		t.Fatalf("picking A again was accepted: answers %v", menu.answers)
	}
}

func TestRingsideAllyCannotBeInTheMatch(t *testing.T) {
	g, in := testGame("A", "B", "C")
	openMenuOption(t, g, in, menuSingles)
	enter(t, g, in) // A
	enter(t, g, in) // B
	menu := g.screen.(*MenuScreen)
	down(t, g, in, 1) // "No Ally" is first, then A
	enter(t, g, in)
	if len(menu.answers) != 2 {
		t.Fatalf("a wrestler in the match was accepted as an ally: answers %v", menu.answers)
	}
	down(t, g, in, 2) // C
	enter(t, g, in)
	enter(t, g, in) // side 2: no ally
	ms := matchScreen(t, g)
	if ms.match.Sides[0].Ally == nil || ms.match.Sides[0].Ally.Name != "C" || ms.match.Sides[1].Ally != nil {
		t.Fatalf("allies: %+v and %+v", ms.match.Sides[0].Ally, ms.match.Sides[1].Ally)
	}
}

func TestEscapeStepsBackOneAnswerAtATime(t *testing.T) {
	g, in := testGame("A", "B", "C")
	openMenuOption(t, g, in, menuSingles)
	enter(t, g, in)
	enter(t, g, in)
	menu := g.screen.(*MenuScreen)
	press(t, g, in, ebiten.KeyEscape)
	if len(menu.answers) != 1 {
		t.Fatalf("after one ESC: answers %v, want one left", menu.answers)
	}
	press(t, g, in, ebiten.KeyEscape)
	press(t, g, in, ebiten.KeyEscape)
	if menu.step != stepMainMenu {
		t.Fatalf("after backing out: step %d, want main menu", menu.step)
	}
}

func TestEscapeOnMainMenuOnlyQuitsWhereQuittingMakesSense(t *testing.T) {
	g, in := testGame("A", "B")
	press(t, g, in, ebiten.KeyEscape)
	if _, ok := g.screen.(*MenuScreen); !ok {
		t.Fatalf("screen is %T, want the menu still", g.screen)
	}

	g.CanQuit = true
	in.set([]ebiten.Key{ebiten.KeyEscape}, "")
	if err := g.Update(); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("Update returned %v, want ebiten.Termination", err)
	}
}

func TestBattleRoyalNeedsThreeWrestlers(t *testing.T) {
	g, in := testGame("A", "B", "C")
	openMenuOption(t, g, in, menuBattleRoyal)
	press(t, g, in, ebiten.KeySpace)
	down(t, g, in, 1)
	press(t, g, in, ebiten.KeySpace)
	enter(t, g, in)
	if _, ok := g.screen.(*MenuScreen); !ok {
		t.Fatalf("two wrestlers were enough to start: screen %T", g.screen)
	}
	down(t, g, in, 1)
	press(t, g, in, ebiten.KeySpace)
	enter(t, g, in)
	if _, ok := g.screen.(*BattleRoyalScreen); !ok {
		t.Fatalf("screen is %T, want *BattleRoyalScreen", g.screen)
	}
}

func TestEditExistingCardOpensTheEditor(t *testing.T) {
	g, in := testGame("A", "B")
	openMenuOption(t, g, in, menuEditCard)
	down(t, g, in, 1)
	enter(t, g, in)
	editor, ok := g.screen.(*CardEditorScreen)
	if !ok {
		t.Fatalf("screen is %T, want *CardEditorScreen", g.screen)
	}
	if editor.fieldValue("Name") != "B" {
		t.Fatalf("editing %q, want B", editor.fieldValue("Name"))
	}
}

func TestRematchKeepsTagTeamsAndTheirPartnerStatus(t *testing.T) {
	g, in := testGame("A", "B", "C", "D")
	match := engine.NewTagMatch(g.Roster[0], g.Roster[1], g.Roster[2], g.Roster[3])
	match.Sides[1].RegularPartners = true
	ms := NewTagMatchScreen(match, g)
	ms.state = MatchFinished
	g.SetScreen(ms)

	press(t, g, in, ebiten.KeyR)
	next := matchScreen(t, g)
	if next == ms {
		t.Fatal("R did not start a new match")
	}
	if next.match.Type != engine.MatchTag {
		t.Fatalf("rematch type %d, want tag", next.match.Type)
	}
	if got := sideNames(next.match.Sides[0]); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("team 1: %v", got)
	}
	if got := sideNames(next.match.Sides[1]); !reflect.DeepEqual(got, []string{"C", "D"}) {
		t.Errorf("team 2: %v", got)
	}
	if next.match.Sides[0].RegularPartners || !next.match.Sides[1].RegularPartners {
		t.Error("regular partner status was not carried over")
	}
}

func TestRematchKeepsAlliesAndFeud(t *testing.T) {
	g, in := testGame("A", "B", "C")
	ms := NewMatchScreen(g.Roster[0], g.Roster[1], engine.MatchCage, g)
	ms.match.Sides[1].Ally = g.Roster[2]
	ms.match.IsFeud = true
	ms.state = MatchFinished
	g.SetScreen(ms)

	press(t, g, in, ebiten.KeyR)
	next := matchScreen(t, g)
	if next.match.Type != engine.MatchCage || !next.match.IsFeud {
		t.Fatalf("rematch type %d feud %v", next.match.Type, next.match.IsFeud)
	}
	if next.match.Sides[1].Ally == nil || next.match.Sides[1].Ally.Name != "C" {
		t.Fatalf("ally not carried over: %+v", next.match.Sides[1].Ally)
	}
}

func TestWindowedLinesKeepsTheCursorVisible(t *testing.T) {
	items := make([]string, 30)
	for i := range items {
		items[i] = string(rune('a' + i%26))
	}
	for cursor := range items {
		lines := windowedLines(items, cursor, 8)
		if len(lines) > 8 {
			t.Fatalf("cursor %d: %d lines for 8 rows", cursor, len(lines))
		}
		found := false
		for _, line := range lines {
			if line == "> "+items[cursor] {
				found = true
			}
		}
		if !found {
			t.Fatalf("cursor %d not visible in %v", cursor, lines)
		}
	}
}

func TestWindowedLinesShowsThereIsMoreToScroll(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	top := windowedLines(items, 0, 6)
	if top[0] != "> a" || top[len(top)-1] != listMoreBelow {
		t.Fatalf("top of list: %v", top)
	}
	middle := windowedLines(items, 5, 6)
	if middle[0] != listMoreAbove || middle[len(middle)-1] != listMoreBelow {
		t.Fatalf("middle of list: %v", middle)
	}
	bottom := windowedLines(items, 9, 6)
	if bottom[0] != listMoreAbove || bottom[len(bottom)-1] != "> j" {
		t.Fatalf("bottom of list: %v", bottom)
	}
	short := windowedLines(items[:3], 1, 6)
	if !reflect.DeepEqual(short, []string{"  a", "> b", "  c"}) {
		t.Fatalf("short list: %v", short)
	}
}
