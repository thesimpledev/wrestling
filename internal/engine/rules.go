package engine

// Rules switches the rulebook's optional player choices on and off. The
// simulator plays both sides, so each rule is a decision it makes itself.
type Rules struct {
	// ChartChoice lets a wrestler decline a chart move marked (c) and roll
	// again one level lower.
	ChartChoice bool `json:"chart_choice"`
	// AvoidDisMoves lets a wrestler decline a "dis" move that is likely to
	// get him disqualified and roll again one level lower.
	AvoidDisMoves bool `json:"avoid_dis_moves"`
	// CustomDisNumbers uses the number printed with a "dis" move, when the
	// card has one, in place of the wrestler's Disqualification rating.
	CustomDisNumbers bool `json:"custom_dis_numbers"`
}

// DefaultRules has every optional rule switched on.
func DefaultRules() Rules {
	return Rules{ChartChoice: true, AvoidDisMoves: true, CustomDisNumbers: true}
}

const riskyDisNumber = 6

// disNumber is the number a "dis" move is rolled against.
func (m *Match) disNumber(att *WrestlerState, move Move) int {
	if m.Rules.CustomDisNumbers && move.DQNumber > 0 {
		return move.DQNumber
	}
	return att.Card.DQ
}

// declinesMove is the simulator's decision to pass on the move just rolled:
// an optional chart move against an opponent rated A on that chart, or a
// "dis" move with a disqualification number of 6 or more.
func (m *Match) declinesMove(att, def *WrestlerState, move Move) bool {
	if m.Rules.ChartChoice && move.HasTag(TagChart) && move.HasTag(TagChartChoice) {
		return m.chartFavoursOpponent(def, move.ChartType)
	}
	if m.Rules.AvoidDisMoves && move.HasTag(TagDQ) && m.dqPossible() {
		return m.disNumber(att, move) >= riskyDisNumber
	}
	return false
}

func (m *Match) chartFavoursOpponent(def *WrestlerState, chartType string) bool {
	if m.Type == MatchCage && chartType == "ring" {
		return false
	}
	_, rating, known := chartFor(chartType, def.Card)
	return known && rating == RatingA
}
