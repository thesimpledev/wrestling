package loader

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
	"wrestling/engine"
	"wrestling/storage"
)

// LoadAllCards loads all wrestler cards from the store and returns them sorted by name.
func LoadAllCards(store storage.Store) ([]*engine.WrestlerCard, error) {
	allBytes, err := store.LoadAllCardBytes()
	if err != nil {
		return nil, fmt.Errorf("loading card bytes: %w", err)
	}

	var cards []*engine.WrestlerCard
	for name, data := range allBytes {
		card, err := ParseCard(data)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", name, err)
		}
		cards = append(cards, card)
	}

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Name < cards[j].Name
	})
	return cards, nil
}

// cardYAML is the intermediate YAML structure before converting to engine types.
type cardYAML struct {
	Name    string        `yaml:"name"`
	Offense [3][6]moveYAML `yaml:"offense"`
	Defense [3][6]defYAML  `yaml:"defense"`

	Ropes      string `yaml:"ropes"`
	Turnbuckle string `yaml:"turnbuckle"`
	Ring       string `yaml:"ring"`
	Deathjump  string `yaml:"deathjump"`

	PIN        int `yaml:"pin"`
	PINAdv     int `yaml:"pin_adv"`
	Cage       int `yaml:"cage"`
	DQ         int `yaml:"dq"`
	Agility    int `yaml:"agility"`
	Power      int `yaml:"power"`
	Distractor int `yaml:"distractor"`

	Finisher finisherYAML `yaml:"finisher"`
}

type moveYAML struct {
	Name     string   `yaml:"name"`
	Power    int      `yaml:"power"`
	DefLevel int      `yaml:"def_level"`
	Tags     []string `yaml:"tags"`
	Chart    string   `yaml:"chart"`
	Choice   string   `yaml:"choice"`
}

type defYAML struct {
	Type         string   `yaml:"type"`
	Power        int      `yaml:"power"`
	Tags         []string `yaml:"tags"`
	PINThreshold int      `yaml:"pin_threshold"`
}

type finisherYAML struct {
	Name    string `yaml:"name"`
	Rating  int    `yaml:"rating"`
	IsRoll  bool   `yaml:"is_roll"`
	RollMin int    `yaml:"roll_min"`
	RollMax int    `yaml:"roll_max"`
}

// ParseCard parses raw YAML bytes into a WrestlerCard.
func ParseCard(data []byte) (*engine.WrestlerCard, error) {
	var raw cardYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing card YAML: %w", err)
	}

	card := &engine.WrestlerCard{
		Name:       raw.Name,
		Ropes:      parseRating(raw.Ropes),
		Turnbuckle: parseRating(raw.Turnbuckle),
		Ring:       parseRating(raw.Ring),
		Deathjump:  parseRating(raw.Deathjump),
		PIN:        raw.PIN,
		PINAdv:     raw.PINAdv,
		Cage:       raw.Cage,
		DQ:         raw.DQ,
		Agility:    raw.Agility,
		Power:      raw.Power,
		Distractor: raw.Distractor,
		Finisher: engine.Finisher{
			Name:    raw.Finisher.Name,
			Rating:  raw.Finisher.Rating,
			IsRoll:  raw.Finisher.IsRoll,
			RollMin: raw.Finisher.RollMin,
			RollMax: raw.Finisher.RollMax,
		},
	}

	if card.Distractor == 0 {
		card.Distractor = 5
	}

	for lvl := 0; lvl < 3; lvl++ {
		for slot := 0; slot < 6; slot++ {
			rm := raw.Offense[lvl][slot]
			card.Offense[lvl][slot] = engine.Move{
				Name:      rm.Name,
				Power:     rm.Power,
				DefLevel:  rm.DefLevel,
				Tags:      parseTags(rm.Tags),
				ChartType: rm.Chart,
				ChoiceKey: rm.Choice,
			}

			rd := raw.Defense[lvl][slot]
			card.Defense[lvl][slot] = engine.DefenseOutcome{
				Type:         parseDefType(rd.Type),
				Power:        rd.Power,
				Tags:         parseTags(rd.Tags),
				PINThreshold: rd.PINThreshold,
			}
		}
	}

	return card, nil
}

func parseRating(s string) engine.Rating {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "A":
		return engine.RatingA
	case "B":
		return engine.RatingB
	default:
		return engine.RatingC
	}
}

func parseDefType(s string) engine.DefenseType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "dazed":
		return engine.DefDazed
	case "hurt":
		return engine.DefHurt
	case "down":
		return engine.DefDown
	case "reversal":
		return engine.DefReversal
	case "pin":
		return engine.DefPIN
	default:
		return engine.DefDazed
	}
}

func parseTags(tags []string) []engine.MoveTag {
	result := make([]engine.MoveTag, 0, len(tags))
	for _, t := range tags {
		result = append(result, engine.MoveTag(strings.TrimSpace(t)))
	}
	return result
}
