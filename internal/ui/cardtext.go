package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"wrestling/internal/engine"
	"wrestling/internal/loader"
)

// The card editor shows each move and defense result as one line of text:
//
//	name,power,deflvl[,extras]      Flying Elbow,2,2,ag
//	type,power[,extras]             down,3,lv
//
// The extras are the instructions printed in parentheses on a card.

// moveExtras is the parsed form of a move line's extras.
type moveExtras struct {
	chart       string
	chartChoice bool
	choice      string
	flags       map[engine.MoveTag]bool
	dqNumber    int
}

var simpleMoveInstructions = []engine.MoveTag{
	engine.TagAgility, engine.TagPower, engine.TagDQ, engine.TagAdd1, engine.TagTagTeam, engine.TagSingles,
}

func isSimpleMoveInstruction(word string) bool {
	for _, tag := range simpleMoveInstructions {
		if string(tag) == word {
			return true
		}
	}
	return false
}

// parseMoveExtras reads the extras part of a move line.
func parseMoveExtras(text string) (moveExtras, error) {
	extras := moveExtras{flags: map[engine.MoveTag]bool{}}
	words := strings.Fields(strings.ToLower(text))
	for i := 0; i < len(words); i++ {
		used, err := extras.take(words[i], nextWord(words, i))
		if err != nil {
			return moveExtras{}, err
		}
		if used {
			i++
		}
	}
	if extras.chartChoice && extras.chart == "" {
		return moveExtras{}, errors.New("(c) only applies to a chart move")
	}
	return extras, nil
}

func nextWord(words []string, i int) string {
	if i+1 < len(words) {
		return words[i+1]
	}
	return ""
}

// take applies one word of the extras. It reports whether the following word
// was used up as part of the same instruction ("ch E", "dis 7").
func (e *moveExtras) take(word, next string) (usedNext bool, err error) {
	switch {
	case loader.KnownCharts[word]:
		if e.chart != "" {
			return false, errors.New("only one chart per move")
		}
		e.chart = word
	case word == string(engine.TagChartChoice):
		e.chartChoice = true
	case word == string(engine.TagChoice):
		return true, e.takeChoice(next)
	case word == string(engine.TagDQ):
		e.flags[engine.TagDQ] = true
		return e.takeDisNumber(next)
	case isSimpleMoveInstruction(word):
		e.flags[engine.MoveTag(word)] = true
	default:
		return false, fmt.Errorf("unknown instruction %q", word)
	}
	return false, nil
}

func (e *moveExtras) takeChoice(letter string) error {
	if letter == "" {
		return errors.New("ch needs a letter (A to H)")
	}
	key := strings.ToUpper(letter)
	if _, known := engine.ChoiceSituations[key]; !known {
		return fmt.Errorf("unknown choice %q", key)
	}
	e.choice = key
	return nil
}

const (
	lowestDisNumber  = 2
	highestDisNumber = 12
)

func (e *moveExtras) takeDisNumber(next string) (usedNext bool, err error) {
	number, convErr := strconv.Atoi(next)
	if convErr != nil {
		return false, nil
	}
	if number < lowestDisNumber || number > highestDisNumber {
		return true, fmt.Errorf("dis number must be %d-%d (got %d)", lowestDisNumber, highestDisNumber, number)
	}
	e.dqNumber = number
	return true, nil
}

// tags lists the move's instructions in the order the editor writes them.
func (e moveExtras) tags() []engine.MoveTag {
	var tags []engine.MoveTag
	if e.chart != "" {
		tags = append(tags, engine.TagChart)
	}
	if e.chartChoice {
		tags = append(tags, engine.TagChartChoice)
	}
	if e.choice != "" {
		tags = append(tags, engine.TagChoice)
	}
	for _, tag := range simpleMoveInstructions {
		if e.flags[tag] {
			tags = append(tags, tag)
		}
	}
	return tags
}

func parseMoveLine(line string) (engine.Move, error) {
	parts := strings.SplitN(line, ",", 4)
	if len(parts) < 3 {
		return engine.Move{}, errors.New("use the format name,power,deflvl")
	}
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return engine.Move{}, errors.New("name cannot be empty")
	}
	power, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return engine.Move{}, errors.New("power must be a number")
	}
	defLevel, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return engine.Move{}, errors.New("def level must be a number")
	}

	extrasText := ""
	if len(parts) == 4 {
		extrasText = parts[3]
	}
	extras, err := parseMoveExtras(extrasText)
	if err != nil {
		return engine.Move{}, err
	}

	// A chart or choice move has no number of its own on a card.
	lowest := 1
	if extras.chart != "" || extras.choice != "" {
		lowest = 0
	}
	if power < lowest || power > 3 {
		return engine.Move{}, fmt.Errorf("power must be 1-3 (got %d)", power)
	}
	if defLevel < lowest || defLevel > 3 {
		return engine.Move{}, fmt.Errorf("def level must be 1-3 (got %d)", defLevel)
	}

	return engine.Move{
		Name:      name,
		Power:     power,
		DefLevel:  defLevel,
		Tags:      extras.tags(),
		ChartType: extras.chart,
		ChoiceKey: extras.choice,
		DQNumber:  extras.dqNumber,
	}, nil
}

func formatMoveLine(move engine.Move) string {
	line := fmt.Sprintf("%s,%d,%d", move.Name, move.Power, move.DefLevel)
	if extras := formatMoveExtras(move); extras != "" {
		line += "," + extras
	}
	return line
}

func formatMoveExtras(move engine.Move) string {
	var words []string
	if move.ChartType != "" {
		words = append(words, move.ChartType)
	}
	if move.HasTag(engine.TagChartChoice) {
		words = append(words, string(engine.TagChartChoice))
	}
	if move.ChoiceKey != "" {
		words = append(words, string(engine.TagChoice)+" "+move.ChoiceKey)
	}
	for _, tag := range simpleMoveInstructions {
		if !move.HasTag(tag) {
			continue
		}
		word := string(tag)
		if tag == engine.TagDQ && move.DQNumber > 0 {
			word += " " + strconv.Itoa(move.DQNumber)
		}
		words = append(words, word)
	}
	return strings.Join(words, " ")
}

var defenseInstructions = []engine.MoveTag{engine.TagTagTeam, engine.TagLeave}

func parseDefenseLine(line string) (engine.DefenseOutcome, error) {
	parts := strings.SplitN(line, ",", 3)
	kind, known := loader.ParseDefenseType(parts[0])
	if !known {
		if len(parts) < 2 {
			return engine.DefenseOutcome{}, errors.New("use the format type,power")
		}
		return engine.DefenseOutcome{}, errors.New("type must be dazed/hurt/down/reversal/pin")
	}

	power, err := defensePower(kind, parts)
	if err != nil {
		return engine.DefenseOutcome{}, err
	}
	outcome := engine.DefenseOutcome{Type: kind, Power: power}
	if len(parts) == 3 {
		outcome.Tags, err = parseDefenseExtras(parts[2])
	}
	return outcome, err
}

// defensePower reads the number after a defense result. PIN has no number on
// a card, so it may be left out or given as 0.
func defensePower(kind engine.DefenseType, parts []string) (int, error) {
	if len(parts) < 2 {
		if kind == engine.DefPIN {
			return 0, nil
		}
		return 0, errors.New("use the format type,power")
	}
	power, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, errors.New("power must be a number")
	}
	lowest := 1
	if kind == engine.DefPIN {
		lowest = 0
	}
	if power < lowest || power > 3 {
		return 0, fmt.Errorf("power must be 1-3 (got %d)", power)
	}
	return power, nil
}

func parseDefenseExtras(text string) ([]engine.MoveTag, error) {
	wanted := map[engine.MoveTag]bool{}
	for _, word := range strings.Fields(strings.ToLower(text)) {
		tag := engine.MoveTag(word)
		if tag != engine.TagTagTeam && tag != engine.TagLeave {
			return nil, fmt.Errorf("unknown instruction %q", word)
		}
		wanted[tag] = true
	}
	var tags []engine.MoveTag
	for _, tag := range defenseInstructions {
		if wanted[tag] {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func formatDefenseLine(outcome engine.DefenseOutcome) string {
	line := fmt.Sprintf("%s,%d", loader.DefenseTypeName(outcome.Type), outcome.Power)
	var words []string
	for _, tag := range defenseInstructions {
		if outcome.HasTag(tag) {
			words = append(words, string(tag))
		}
	}
	if len(words) > 0 {
		line += "," + strings.Join(words, " ")
	}
	return line
}

// applyFinisherRoll reads the Finisher Roll field: blank for a plain
// finisher, or "min-max" for a roll finisher that connects on that range of
// one die.
func applyFinisherRoll(f *engine.Finisher, text string) error {
	f.IsRoll, f.RollMin, f.RollMax = false, 0, 0
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	parts := strings.Split(text, "-")
	if len(parts) != 2 {
		return errors.New("use min-max, for example 2-6, or leave it blank")
	}
	low, lowErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	high, highErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if lowErr != nil || highErr != nil {
		return errors.New("use min-max, for example 2-6, or leave it blank")
	}
	if low < 1 || high > 6 {
		return errors.New("roll numbers must be 1 to 6")
	}
	if low > high {
		return errors.New("put the smaller number first")
	}
	f.IsRoll, f.RollMin, f.RollMax = true, low, high
	return nil
}

func formatFinisherRoll(f engine.Finisher) string {
	if !f.IsRoll {
		return ""
	}
	return fmt.Sprintf("%d-%d", f.RollMin, f.RollMax)
}
