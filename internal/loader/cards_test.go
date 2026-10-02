package loader

import (
	"reflect"
	"strings"
	"testing"

	"wrestling/internal/engine"
)

const plainMoveYAML = `{name: "Jab", power: 1, def_level: 1}`

func TestLoadAllCardsSkipsUnreadableCardsAndReportsThem(t *testing.T) {
	store := &memStore{cards: map[string][]byte{
		"good.yaml":   []byte(cardYAMLWith(plainMoveYAML, plainMoveYAML)),
		"broken.yaml": []byte("name: \"Broken\noffense: [[[ not yaml"),
	}}
	cards, problems, err := LoadAllCards(store)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cards) != 1 || cards[0].Name != "Some Wrestler" {
		t.Fatalf("cards: %v", cards)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "broken.yaml") {
		t.Fatalf("problems: %v", problems)
	}
}

func TestLoadAllCardsSortsByName(t *testing.T) {
	second := strings.Replace(cardYAMLWith(plainMoveYAML, plainMoveYAML), "Some Wrestler", "Another Wrestler", 1)
	store := &memStore{cards: map[string][]byte{
		"a.yaml": []byte(cardYAMLWith(plainMoveYAML, plainMoveYAML)),
		"b.yaml": []byte(second),
	}}
	cards, problems, err := LoadAllCards(store)
	if err != nil || len(problems) != 0 {
		t.Fatalf("load: %v, problems %v", err, problems)
	}
	if len(cards) != 2 || cards[0].Name != "Another Wrestler" || cards[1].Name != "Some Wrestler" {
		t.Fatalf("order: %s, %s", cards[0].Name, cards[1].Name)
	}
}

func TestCardProblemsNameInstructionsTheEngineDoesNotKnow(t *testing.T) {
	cases := []struct {
		name        string
		first       string
		second      string
		wantProblem string
	}{
		{"unknown move instruction",
			`{name: "Low Blow", power: 2, def_level: 2, tags: ["dq"]}`, plainMoveYAML, `"dq"`},
		{"choice move with no letter",
			`{name: "Ropes", power: 0, def_level: 1, tags: ["ch"], chart: "ropes"}`, plainMoveYAML, "choice"},
		{"choice letter that is not on the chart",
			`{name: "Choice", power: 2, def_level: 2, tags: ["ch"], choice: "Z"}`, plainMoveYAML, `"Z"`},
		{"chart move with no chart",
			`{name: "Into Something", power: 2, def_level: 2, tags: ["chart"]}`, plainMoveYAML, "chart"},
		{"chart that does not exist",
			`{name: "Into the Crowd", power: 2, def_level: 2, tags: ["chart"], chart: "crowd"}`, plainMoveYAML, `"crowd"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			card, err := ParseCard([]byte(cardYAMLWith(tc.first, tc.second)))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			problems := CardProblems(card)
			if len(problems) != 1 || !strings.Contains(problems[0], tc.wantProblem) {
				t.Fatalf("problems: %v, want one mentioning %s", problems, tc.wantProblem)
			}
			if !strings.Contains(problems[0], "Some Wrestler") {
				t.Fatalf("problem does not name the card: %s", problems[0])
			}
		})
	}
}

func TestCardProblemsReportsUnknownDefenseResult(t *testing.T) {
	text := strings.Replace(cardYAMLWith(plainMoveYAML, plainMoveYAML),
		`{type: "dazed", power: 1}`, `{type: "stunned", power: 1}`, 1)
	_, err := ParseCard([]byte(text))
	if err == nil || !strings.Contains(err.Error(), "stunned") {
		t.Fatalf("want a parse error naming the defense result, got %v", err)
	}
}

func TestBundledCardsHaveNoProblems(t *testing.T) {
	for name, card := range loadBundled(t) {
		if problems := CardProblems(card); len(problems) != 0 {
			t.Errorf("%s: %v", name, problems)
		}
	}
}

func TestMarshalCardRoundTripsEveryBundledCard(t *testing.T) {
	for name, card := range loadBundled(t) {
		data, err := MarshalCard(card)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		again, err := ParseCard(data)
		if err != nil {
			t.Fatalf("%s: parsing what was saved: %v\n%s", name, err, data)
		}
		if !reflect.DeepEqual(card, again) {
			t.Errorf("%s: card changed after save and load\nbefore: %+v\nafter:  %+v", name, card, again)
		}
	}
}

func TestMarshalCardKeepsOptionalRuleData(t *testing.T) {
	card, err := ParseCard([]byte(cardYAMLWith(
		`{name: "Into the Ropes", power: 2, def_level: 1, tags: ["chart", "c"], chart: "ropes"}`,
		`{name: "Chair Shot", power: 3, def_level: 3, tags: ["dis"], dis_number: 7}`,
	)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	card.Finisher = engine.Finisher{Name: "RISKY FINISH", IsRoll: true, RollMin: 2, RollMax: 6}
	data, err := MarshalCard(card)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	again, err := ParseCard(data)
	if err != nil {
		t.Fatalf("parse saved card: %v", err)
	}
	if !reflect.DeepEqual(card, again) {
		t.Fatalf("card changed after save and load\nbefore: %+v\nafter:  %+v", card, again)
	}
}

func TestSummarizeProblems(t *testing.T) {
	if got := SummarizeProblems(nil); got != "" {
		t.Fatalf("no problems: %q", got)
	}
	if got := SummarizeProblems([]string{"only one"}); got != "only one" {
		t.Fatalf("one problem: %q", got)
	}
	got := SummarizeProblems([]string{"first", "second", "third"})
	if !strings.Contains(got, "first") || !strings.Contains(got, "2 more") {
		t.Fatalf("several problems: %q", got)
	}
}
