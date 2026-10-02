package loader

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
	"wrestling/internal/engine"
	"wrestling/internal/storage"
)

const defaultDistractor = 5

// LoadAllCards loads every wrestler card from the store, sorted by name. A
// card that cannot be read is left out, and a card that uses an instruction
// the engine does not know is still loaded; both are described in problems.
// The error is for the store itself failing.
func LoadAllCards(store storage.Store) (cards []*engine.WrestlerCard, problems []string, err error) {
	allBytes, err := store.LoadAllCardBytes()
	if err != nil {
		return nil, nil, fmt.Errorf("loading card bytes: %w", err)
	}

	for _, file := range sortedKeys(allBytes) {
		card, err := ParseCard(allBytes[file])
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s was skipped: %v", file, err))
			continue
		}
		cards = append(cards, card)
		problems = append(problems, CardProblems(card)...)
	}

	sort.SliceStable(cards, func(i, j int) bool {
		return cards[i].Name < cards[j].Name
	})
	return cards, problems, nil
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SummarizeProblems fits a list of card problems on one line: the first one,
// and a count of the rest.
func SummarizeProblems(problems []string) string {
	switch len(problems) {
	case 0:
		return ""
	case 1:
		return problems[0]
	default:
		return fmt.Sprintf("%s (and %d more card problems)", problems[0], len(problems)-1)
	}
}

// cardYAML is the intermediate YAML structure before converting to engine types.
type cardYAML struct {
	Name    string         `yaml:"name"`
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
	Tags     []string `yaml:"tags,omitempty,flow"`
	Chart    string   `yaml:"chart,omitempty"`
	Choice   string   `yaml:"choice,omitempty"`
	DQNumber int      `yaml:"dis_number,omitempty"`
}

type defYAML struct {
	Type         string   `yaml:"type"`
	Power        int      `yaml:"power"`
	Tags         []string `yaml:"tags,omitempty,flow"`
	PINThreshold int      `yaml:"pin_threshold,omitempty"`
}

type finisherYAML struct {
	Name    string `yaml:"name"`
	Rating  int    `yaml:"rating"`
	IsRoll  bool   `yaml:"is_roll,omitempty"`
	RollMin int    `yaml:"roll_min,omitempty"`
	RollMax int    `yaml:"roll_max,omitempty"`
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
		Finisher:   engine.Finisher(raw.Finisher),
	}
	if card.Distractor == 0 {
		card.Distractor = defaultDistractor
	}

	for lvl := range raw.Offense {
		for slot := range raw.Offense[lvl] {
			card.Offense[lvl][slot] = parseMove(raw.Offense[lvl][slot])
			outcome, err := parseDefense(raw.Defense[lvl][slot])
			if err != nil {
				return nil, fmt.Errorf("defense level %d result %d: %w", lvl+1, slot+1, err)
			}
			card.Defense[lvl][slot] = outcome
		}
	}
	return card, nil
}

func parseMove(raw moveYAML) engine.Move {
	return engine.Move{
		Name:      raw.Name,
		Power:     raw.Power,
		DefLevel:  raw.DefLevel,
		Tags:      parseTags(raw.Tags),
		ChartType: raw.Chart,
		ChoiceKey: raw.Choice,
		DQNumber:  raw.DQNumber,
	}
}

func parseDefense(raw defYAML) (engine.DefenseOutcome, error) {
	kind, ok := ParseDefenseType(raw.Type)
	if !ok {
		return engine.DefenseOutcome{}, fmt.Errorf("unknown defense result %q", raw.Type)
	}
	return engine.DefenseOutcome{
		Type:         kind,
		Power:        raw.Power,
		Tags:         parseTags(raw.Tags),
		PINThreshold: raw.PINThreshold,
	}, nil
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

var defenseTypeNames = map[engine.DefenseType]string{
	engine.DefDazed:    "dazed",
	engine.DefHurt:     "hurt",
	engine.DefDown:     "down",
	engine.DefReversal: "reversal",
	engine.DefPIN:      "pin",
}

// DefenseTypeName is the word a defense result is written as on a card.
func DefenseTypeName(kind engine.DefenseType) string {
	return defenseTypeNames[kind]
}

// ParseDefenseType reads a defense result word as written on a card.
func ParseDefenseType(s string) (engine.DefenseType, bool) {
	word := strings.ToLower(strings.TrimSpace(s))
	for kind, name := range defenseTypeNames {
		if name == word {
			return kind, true
		}
	}
	return engine.DefDazed, false
}

func parseTags(tags []string) []engine.MoveTag {
	if len(tags) == 0 {
		return nil
	}
	result := make([]engine.MoveTag, 0, len(tags))
	for _, t := range tags {
		result = append(result, engine.MoveTag(strings.TrimSpace(t)))
	}
	return result
}

// MarshalCard writes a card as YAML that ParseCard reads back unchanged.
func MarshalCard(card *engine.WrestlerCard) ([]byte, error) {
	raw := cardYAML{
		Name:       card.Name,
		Ropes:      card.Ropes.String(),
		Turnbuckle: card.Turnbuckle.String(),
		Ring:       card.Ring.String(),
		Deathjump:  card.Deathjump.String(),
		PIN:        card.PIN,
		PINAdv:     card.PINAdv,
		Cage:       card.Cage,
		DQ:         card.DQ,
		Agility:    card.Agility,
		Power:      card.Power,
		Distractor: card.Distractor,
		Finisher:   finisherYAML(card.Finisher),
	}
	for lvl := range card.Offense {
		for slot := range card.Offense[lvl] {
			raw.Offense[lvl][slot] = marshalMove(card.Offense[lvl][slot])
			raw.Defense[lvl][slot] = marshalDefense(card.Defense[lvl][slot])
		}
	}
	return yaml.Marshal(raw)
}

func marshalMove(move engine.Move) moveYAML {
	return moveYAML{
		Name:     move.Name,
		Power:    move.Power,
		DefLevel: move.DefLevel,
		Tags:     tagStrings(move.Tags),
		Chart:    move.ChartType,
		Choice:   move.ChoiceKey,
		DQNumber: move.DQNumber,
	}
}

func marshalDefense(outcome engine.DefenseOutcome) defYAML {
	return defYAML{
		Type:         DefenseTypeName(outcome.Type),
		Power:        outcome.Power,
		Tags:         tagStrings(outcome.Tags),
		PINThreshold: outcome.PINThreshold,
	}
}

func tagStrings(tags []engine.MoveTag) []string {
	if len(tags) == 0 {
		return nil
	}
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		result = append(result, string(t))
	}
	return result
}
