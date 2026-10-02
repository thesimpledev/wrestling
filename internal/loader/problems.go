package loader

import (
	"fmt"

	"wrestling/internal/engine"
)

var knownMoveTags = map[engine.MoveTag]bool{
	engine.TagChart:       true,
	engine.TagChoice:      true,
	engine.TagAgility:     true,
	engine.TagPower:       true,
	engine.TagDQ:          true,
	engine.TagAdd1:        true,
	engine.TagTagTeam:     true,
	engine.TagSingles:     true,
	engine.TagChartChoice: true,
}

var knownDefenseTags = map[engine.MoveTag]bool{
	engine.TagTagTeam: true,
	engine.TagLeave:   true,
}

// KnownCharts are the chart names a chart move can send the opponent to.
var KnownCharts = map[string]bool{
	"ropes":      true,
	"turnbuckle": true,
	"ring":       true,
	"deathjump":  true,
}

// CardProblems describes what on a card the engine cannot act on: an
// instruction it does not know, or a chart or choice move that does not say
// which chart or choice. The card still plays, but those moves do nothing
// special, so the player should know.
func CardProblems(card *engine.WrestlerCard) []string {
	var problems []string
	for lvl := range card.Offense {
		for slot := range card.Offense[lvl] {
			for _, problem := range moveProblems(card.Offense[lvl][slot]) {
				problems = append(problems, fmt.Sprintf("%s, Level %d move %d: %s", card.Name, lvl+1, slot+1, problem))
			}
			for _, tag := range card.Defense[lvl][slot].Tags {
				if !knownDefenseTags[tag] {
					problems = append(problems, fmt.Sprintf("%s, Level %d defense %d: unknown instruction %q",
						card.Name, lvl+1, slot+1, tag))
				}
			}
		}
	}
	return problems
}

func moveProblems(move engine.Move) []string {
	var problems []string
	for _, tag := range move.Tags {
		if !knownMoveTags[tag] {
			problems = append(problems, fmt.Sprintf("unknown instruction %q", tag))
		}
	}
	if move.HasTag(engine.TagChart) {
		problems = append(problems, chartProblems(move)...)
	}
	if move.HasTag(engine.TagChoice) {
		problems = append(problems, choiceProblems(move)...)
	}
	return problems
}

func chartProblems(move engine.Move) []string {
	if move.ChartType == "" {
		return []string{"chart move does not say which chart"}
	}
	if !KnownCharts[move.ChartType] {
		return []string{fmt.Sprintf("unknown chart %q", move.ChartType)}
	}
	return nil
}

func choiceProblems(move engine.Move) []string {
	if move.ChoiceKey == "" {
		return []string{"choice move does not say which choice (A to H)"}
	}
	if _, known := engine.ChoiceSituations[move.ChoiceKey]; !known {
		return []string{fmt.Sprintf("unknown choice %q", move.ChoiceKey)}
	}
	return nil
}
