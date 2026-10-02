package loader

import (
	"os"
	"path/filepath"
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
