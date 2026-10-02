package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
	"wrestling/internal/loader"
)

// richCard uses every kind of instruction the editor has to carry through.
func richCard() *engine.WrestlerCard {
	card := &engine.WrestlerCard{
		Name: "Rich Card", Ropes: engine.RatingA, Turnbuckle: engine.RatingB, Ring: engine.RatingC,
		Deathjump: engine.RatingA, PIN: 6, PINAdv: 3, Cage: 5, DQ: 4, Agility: -2, Power: 3, Distractor: 7,
		Finisher: engine.Finisher{Name: "RICH FINISH", Rating: 0, IsRoll: true, RollMin: 2, RollMax: 6},
	}
	plain := engine.Move{Name: "Jab", Power: 1, DefLevel: 1}
	for lvl := range card.Offense {
		for slot := range card.Offense[lvl] {
			card.Offense[lvl][slot] = plain
			card.Defense[lvl][slot] = engine.DefenseOutcome{Type: engine.DefDazed, Power: 1}
		}
	}
	card.Offense[0][0] = engine.Move{Name: "Into the Ropes", Power: 2, DefLevel: 1,
		Tags: tags(engine.TagChart, engine.TagChartChoice), ChartType: "ropes"}
	card.Offense[1][1] = engine.Move{Name: "Choice", Power: 2, DefLevel: 2, Tags: tags(engine.TagChoice), ChoiceKey: "C"}
	card.Offense[1][2] = engine.Move{Name: "Flying Elbow", Power: 2, DefLevel: 2, Tags: tags(engine.TagAgility)}
	card.Offense[1][3] = engine.Move{Name: "Chair Shot", Power: 3, DefLevel: 3, Tags: tags(engine.TagDQ), DQNumber: 7}
	card.Offense[2][0] = engine.Move{Name: "Big Slam", Power: 3, DefLevel: 3, Tags: tags(engine.TagPower, engine.TagAdd1)}
	card.Offense[2][4] = engine.Move{Name: "Double Team", Power: 2, DefLevel: 2, Tags: tags(engine.TagTagTeam)}
	card.Offense[2][5] = engine.Move{Name: "RICH FINISH", Power: 3, DefLevel: 3}
	card.Defense[1][5] = engine.DefenseOutcome{Type: engine.DefReversal, Power: 2}
	card.Defense[2][2] = engine.DefenseOutcome{Type: engine.DefDown, Power: 3, Tags: tags(engine.TagLeave)}
	card.Defense[2][3] = engine.DefenseOutcome{Type: engine.DefHurt, Power: 2, Tags: tags(engine.TagTagTeam)}
	card.Defense[2][4] = engine.DefenseOutcome{Type: engine.DefPIN, PINThreshold: 6}
	return card
}

func ctrlS(t *testing.T, g *Game, in *fakeInput) {
	t.Helper()
	in.set([]ebiten.Key{ebiten.KeyS}, "")
	in.held = map[ebiten.Key]bool{ebiten.KeyControl: true}
	if err := g.Update(); err != nil {
		t.Fatalf("Update returned %v", err)
	}
	in.held = nil
	in.set(nil, "")
}

func openEditor(card *engine.WrestlerCard, roster ...string) (*Game, *fakeInput, *CardEditorScreen) {
	g, in := testGame(roster...)
	editor := NewCardEditorScreen(card)
	g.SetScreen(editor)
	return g, in, editor
}

func savedCard(t *testing.T, g *Game, file string) *engine.WrestlerCard {
	t.Helper()
	data, ok := g.Store.(*memStore).cards[file]
	if !ok {
		t.Fatalf("%s was not saved; saved files: %v", file, reflect.ValueOf(g.Store.(*memStore).cards).MapKeys())
	}
	card, err := loader.ParseCard(data)
	if err != nil {
		t.Fatalf("saved card does not parse: %v\n%s", err, data)
	}
	return card
}

func TestEditorSavesAnUnchangedCardWithoutLosingAnything(t *testing.T) {
	original := richCard()
	g, in, editor := openEditor(original)
	ctrlS(t, g, in)
	if !strings.HasPrefix(editor.message, "Saved") {
		t.Fatalf("message: %q", editor.message)
	}
	if got := savedCard(t, g, "rich_card.yaml"); !reflect.DeepEqual(got, original) {
		t.Fatalf("card changed on save\nbefore: %+v\nafter:  %+v", original, got)
	}
}

func TestEditorShowsInstructionsOnTheMoveLines(t *testing.T) {
	_, _, editor := openEditor(richCard())
	checks := map[string]string{
		moveLabel(1, 1):           "Into the Ropes,2,1,ropes c",
		moveLabel(2, 2):           "Choice,2,2,ch C",
		moveLabel(2, 4):           "Chair Shot,3,3,dis 7",
		moveLabel(3, 1):           "Big Slam,3,3,pw add1",
		defenseLabel(3, 3):        "down,3,lv",
		"Finisher Roll (min-max)": "2-6",
	}
	for label, want := range checks {
		if got := editor.fieldValue(label); got != want {
			t.Errorf("%s: %q, want %q", label, got, want)
		}
	}
}

func TestEditorNewCardSavesAndJoinsTheRoster(t *testing.T) {
	g, in, _ := openEditor(nil, "A", "B")
	ctrlS(t, g, in)
	card := savedCard(t, g, "new_wrestler.yaml")
	if card.Name != "New Wrestler" {
		t.Fatalf("saved name %q", card.Name)
	}
	found := false
	for _, w := range g.Roster {
		if w.Name == "New Wrestler" {
			found = true
		}
	}
	if !found {
		t.Fatal("new card is not in the roster after saving")
	}
}

func TestEditorTypingChangesAField(t *testing.T) {
	g, in, editor := openEditor(nil)
	enter(t, g, in)
	typeText(t, g, in, " Jr")
	press(t, g, in, ebiten.KeyBackspace)
	enter(t, g, in)
	if got := editor.fieldValue("Name"); got != "New Wrestler J" {
		t.Fatalf("name: %q", got)
	}
	if editor.editing {
		t.Fatal("ENTER should finish editing")
	}
}

func TestEditorRatingFieldTakesALetter(t *testing.T) {
	g, in, editor := openEditor(nil)
	down(t, g, in, 1)
	if got := editor.fields[editor.cursor].Label; got != "Ropes" {
		t.Fatalf("cursor on %q, want Ropes (section headings are skipped)", got)
	}
	enter(t, g, in)
	press(t, g, in, ebiten.KeyA)
	if got := editor.fieldValue("Ropes"); got != "A" {
		t.Fatalf("Ropes: %q", got)
	}
}

func TestEditorCursorWrapsPastHeadings(t *testing.T) {
	g, in, editor := openEditor(nil)
	press(t, g, in, ebiten.KeyUp)
	if editor.cursor != len(editor.fields)-1 {
		t.Fatalf("UP from the first field: cursor %d, want the last field", editor.cursor)
	}
	down(t, g, in, 1)
	if editor.cursor != 0 {
		t.Fatalf("DOWN from the last field: cursor %d, want 0", editor.cursor)
	}
}

func TestEditorValidation(t *testing.T) {
	cases := []struct {
		label string
		value string
		want  string
	}{
		{"Name", "  ", "Name cannot be empty"},
		{"Ropes", "D", "Ropes must be A, B, or C"},
		{"PIN", "13", "PIN must be 1 to 12 (got 13)"},
		{"PIN", "abc", "PIN must be 1 to 12 (got 0)"},
		{"PIN Adv", "0", "PIN Adv must be 1 to 12"},
		{"Cage", "13", "Cage must be 1 to 12"},
		{"DQ", "0", "DQ must be 1 to 12"},
		{"Agility", "-6", "Agility must be -5 to 5 (got -6)"},
		{"Power", "6", "Power must be -5 to 5"},
		{"Distractor", "13", "Distractor must be 1 to 12"},
		{"Finisher Name", "", "Finisher Name cannot be empty"},
		{"Finisher Rating", "9", "Finisher Rating must be 0 to 8"},
		{"Finisher Roll (min-max)", "7", "Finisher Roll"},
		{moveLabel(1, 1), "Move 1", "L1 Move 1: use the format name,power,deflvl"},
		{moveLabel(2, 3), "Slam,4,1", "L2 Move 3: power must be 1-3 (got 4)"},
		{moveLabel(1, 1), "Slam,1,1,dq", `L1 Move 1: unknown instruction "dq"`},
		{defenseLabel(1, 1), "stun,1", "L1 Def 1: type must be dazed/hurt/down/reversal/pin"},
		{defenseLabel(3, 6), "down,0", "L3 Def 6: power must be 1-3 (got 0)"},
	}
	for _, tc := range cases {
		t.Run(tc.label+"="+tc.value, func(t *testing.T) {
			g, in, editor := openEditor(nil)
			editor.setFieldValue(tc.label, tc.value)
			ctrlS(t, g, in)
			if !strings.Contains(editor.message, tc.want) {
				t.Fatalf("message %q does not contain %q", editor.message, tc.want)
			}
			if saved := g.Store.(*memStore).cards; len(saved) != 0 {
				t.Fatalf("an invalid card was saved: %v", reflect.ValueOf(saved).MapKeys())
			}
		})
	}
}

func TestEditorAcceptsTheCardFormatFromTheReadme(t *testing.T) {
	g, in, editor := openEditor(nil)
	editor.setFieldValue(defenseLabel(2, 6), "pin,0")
	editor.setFieldValue(moveLabel(2, 3), "Into the Ropes,0,1,ropes")
	ctrlS(t, g, in)
	if !strings.HasPrefix(editor.message, "Saved") {
		t.Fatalf("message: %q", editor.message)
	}
	card := savedCard(t, g, "new_wrestler.yaml")
	if card.Defense[1][5].Type != engine.DefPIN {
		t.Errorf("defense: %+v", card.Defense[1][5])
	}
	if !card.Offense[1][2].HasTag(engine.TagChart) || card.Offense[1][2].ChartType != "ropes" {
		t.Errorf("move: %+v", card.Offense[1][2])
	}
}

func TestEditorRenamingSavesANewCardAndSaysSo(t *testing.T) {
	g, in, editor := openEditor(richCard())
	editor.setFieldValue("Name", "Second Card")
	ctrlS(t, g, in)
	if !strings.Contains(editor.message, "new card") || !strings.Contains(editor.message, "Rich Card") {
		t.Fatalf("message: %q", editor.message)
	}
	savedCard(t, g, "second_card.yaml")

	ctrlS(t, g, in)
	if strings.Contains(editor.message, "new card") {
		t.Fatalf("saving again under the same name: %q", editor.message)
	}
}

func TestEditorEscapeLeavesWithoutSaving(t *testing.T) {
	g, in, editor := openEditor(nil)
	editor.setFieldValue("Name", "Unsaved")
	press(t, g, in, ebiten.KeyEscape)
	if _, ok := g.screen.(*MenuScreen); !ok {
		t.Fatalf("screen is %T, want the menu", g.screen)
	}
	if saved := g.Store.(*memStore).cards; len(saved) != 0 {
		t.Fatal("ESC saved the card")
	}
}

func TestEditorListStopsAboveTheMessageLine(t *testing.T) {
	for _, screenH := range []int{360, 368, 720, 737} {
		listBottom := editorListTop + editorVisibleLines(screenH)*LineHeight
		if messageLine := editorMessageY(screenH); listBottom > messageLine {
			t.Errorf("screen height %d: list ends at %d, message line starts at %d", screenH, listBottom, messageLine)
		}
	}
}

func TestReloadRosterReportsCardProblems(t *testing.T) {
	g, _ := testGame("A", "B")
	g.Store.(*memStore).cards["broken.yaml"] = []byte("name: \"Broken\noffense: [[[")
	reloadRoster(g)
	if !strings.Contains(g.notice, "broken.yaml") {
		t.Fatalf("notice: %q", g.notice)
	}
}
