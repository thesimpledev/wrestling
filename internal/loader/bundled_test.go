package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wrestling/internal/engine"
)

const bundledDir = "../../data/wrestlers"

func loadBundled(t *testing.T) map[string]*engine.WrestlerCard {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(bundledDir, "*.yaml"))
	if err != nil {
		t.Fatalf("listing bundled cards: %v", err)
	}
	if len(files) != 8 {
		t.Fatalf("bundled cards: got %d files, want 8", len(files))
	}
	cards := make(map[string]*engine.WrestlerCard, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file) // #nosec G304 -- test reads the repo's own card files
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		card, err := ParseCard(data)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		cards[card.Name] = card
	}
	return cards
}

func readmeExampleCard(t *testing.T) []byte {
	t.Helper()
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("reading the README: %v", err)
	}
	const open, closing = "```yaml\n", "\n```"
	start := strings.Index(string(readme), open)
	if start < 0 {
		t.Fatal("the README has no yaml example")
	}
	body := string(readme)[start+len(open):]
	end := strings.Index(body, closing)
	if end < 0 {
		t.Fatal("the README's yaml example is not closed")
	}
	return []byte(body[:end])
}

func TestReadmeExampleCardLoadsWithoutProblems(t *testing.T) {
	card, err := ParseCard(readmeExampleCard(t))
	if err != nil {
		t.Fatalf("the README's example card does not load: %v", err)
	}
	if problems := CardProblems(card); len(problems) != 0 {
		t.Fatalf("the README's example card uses instructions the game does not know: %v", problems)
	}
	if card.Offense[1][4].DQNumber != 7 || !card.Offense[0][3].HasTag(engine.TagChartChoice) {
		t.Fatalf("the README's example card lost its instructions: %+v, %+v", card.Offense[1][4], card.Offense[0][3])
	}
	if !card.Offense[2][5].IsFinisher() || card.Offense[2][5].Name != card.Finisher.Name {
		t.Fatalf("the README's finisher %q is not the capital-letter move on Level 3", card.Finisher.Name)
	}
}

// The rulebook rates agility and power from -5 (excellent) to +5 (poor).
func TestBundledCardsUseRulebookAgilityAndPower(t *testing.T) {
	cards := loadBundled(t)
	want := map[string][2]int{
		"Armand the Colossus": {3, -5},
		"Blake Harton":        {-2, 0},
		"Buck Stallion":       {2, -5},
		"Butcher Briggs":      {1, -4},
		"Rex Fontaine":        {-1, 0},
		"Ricky Rampage":       {-2, -1},
		"Rico Stormcloud":     {-4, 0},
		"The Gravedigger":     {1, -4},
	}
	for name, ratings := range want {
		card, ok := cards[name]
		if !ok {
			t.Errorf("%s: card not found", name)
			continue
		}
		if card.Agility != ratings[0] || card.Power != ratings[1] {
			t.Errorf("%s: agility %d power %d, want agility %d power %d",
				name, card.Agility, card.Power, ratings[0], ratings[1])
		}
	}
}
