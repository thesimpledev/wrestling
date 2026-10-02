package loader

import (
	"strings"
	"testing"

	"wrestling/internal/engine"
)

func (s *memStore) LoadSettingsJSON() ([]byte, error) { return s.settings, nil }
func (s *memStore) SaveSettingsJSON(data []byte) error {
	s.settings = data
	return nil
}

// cardYAMLWith builds a complete card whose first two Level 1 moves are the
// given YAML entries and whose other slots are plain.
func cardYAMLWith(firstMove, secondMove string) string {
	const plainMove = `{name: "Jab", power: 1, def_level: 1}`
	const plainDefense = `{type: "dazed", power: 1}`
	var b strings.Builder
	b.WriteString("name: \"Some Wrestler\"\noffense:\n")
	for lvl := 0; lvl < 3; lvl++ {
		for slot := 0; slot < 6; slot++ {
			move := plainMove
			if lvl == 0 && slot == 0 {
				move = firstMove
			}
			if lvl == 0 && slot == 1 {
				move = secondMove
			}
			b.WriteString(gridPrefix(slot) + move + "\n")
		}
	}
	b.WriteString("defense:\n")
	for lvl := 0; lvl < 3; lvl++ {
		for slot := 0; slot < 6; slot++ {
			b.WriteString(gridPrefix(slot) + plainDefense + "\n")
		}
	}
	b.WriteString("ropes: \"B\"\nturnbuckle: \"B\"\nring: \"B\"\ndeathjump: \"B\"\n")
	b.WriteString("pin: 5\npin_adv: 3\ncage: 5\ndq: 4\nagility: 0\npower: 0\n")
	b.WriteString("finisher:\n  name: \"BIG FINISH\"\n  rating: 3\n")
	return b.String()
}

func gridPrefix(slot int) string {
	if slot == 0 {
		return "  - - "
	}
	return "    - "
}

func TestParseCardReadsOptionalRuleData(t *testing.T) {
	card, err := ParseCard([]byte(cardYAMLWith(
		`{name: "Into the Ropes", power: 2, def_level: 1, tags: ["chart", "c"], chart: "ropes"}`,
		`{name: "Chair Shot", power: 3, def_level: 3, tags: ["dis"], dis_number: 7}`,
	)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ropes := card.Offense[0][0]
	if !ropes.HasTag(engine.TagChart) || !ropes.HasTag(engine.TagChartChoice) {
		t.Errorf("ropes move tags: %v", ropes.Tags)
	}
	chair := card.Offense[0][1]
	if !chair.HasTag(engine.TagDQ) || chair.DQNumber != 7 {
		t.Errorf("chair shot: tags %v dis number %d", chair.Tags, chair.DQNumber)
	}
}

func TestLoadRulesDefaultsToEverythingOn(t *testing.T) {
	rules, err := LoadRules(&memStore{})
	if err != nil || rules != engine.DefaultRules() {
		t.Fatalf("got %+v, %v", rules, err)
	}
}

func TestRulesSaveAndLoad(t *testing.T) {
	store := &memStore{}
	want := engine.Rules{ChartChoice: false, AvoidDisMoves: true, CustomDisNumbers: false}
	if err := SaveRules(store, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := LoadRules(store)
	if err != nil || got != want {
		t.Fatalf("got %+v, %v, want %+v", got, err, want)
	}
}

func TestLoadRulesKeepsDefaultsForSettingsNotSaved(t *testing.T) {
	store := &memStore{settings: []byte(`{"chart_choice": false}`)}
	got, err := LoadRules(store)
	want := engine.Rules{ChartChoice: false, AvoidDisMoves: true, CustomDisNumbers: true}
	if err != nil || got != want {
		t.Fatalf("got %+v, %v, want %+v", got, err, want)
	}
}

func TestLoadRulesReportsUnreadableSettings(t *testing.T) {
	store := &memStore{settings: []byte(`{broken`)}
	got, err := LoadRules(store)
	if err == nil {
		t.Fatal("want an error for unreadable settings")
	}
	if got != engine.DefaultRules() {
		t.Fatalf("want default rules alongside the error, got %+v", got)
	}
}
