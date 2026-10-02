package engine

// choiceThreshold is the highest roll a choice move works on against this
// opponent: the chart number plus or minus the opponent's agility or power.
func choiceThreshold(opt ChoiceOption, opponent *WrestlerCard) int {
	if opt.StatType == "ag" {
		return opt.Threshold + opponent.Agility
	}
	return opt.Threshold + opponent.Power
}

const reliableChoiceThreshold = 8

// pickChoiceOption makes the simulator's decision in a choice situation: the
// roll move more likely to work, or a chart move when the roll move is risky.
func pickChoiceOption(opponent *WrestlerCard, choice ChoiceSituation) ChoiceOption {
	first, second := choice.Option1, choice.Option2
	switch {
	case first.IsChart:
		return rollMoveOrChart(second, first, opponent)
	case second.IsChart:
		return rollMoveOrChart(first, second, opponent)
	case choiceThreshold(first, opponent) >= choiceThreshold(second, opponent):
		return first
	default:
		return second
	}
}

func rollMoveOrChart(rollMove, chart ChoiceOption, opponent *WrestlerCard) ChoiceOption {
	if choiceThreshold(rollMove, opponent) >= reliableChoiceThreshold {
		return rollMove
	}
	return chart
}

func (m *Match) resolveChoice(att, def *WrestlerState, choiceKey string) {
	choice, ok := ChoiceSituations[choiceKey]
	if !ok {
		m.emit(newEvent(EventChart, "Unknown choice situation %q, continuing normally.", choiceKey))
		return
	}

	m.emit(newEvent(EventChart, "CHOICE SITUATION %s! %s must decide between %s or %s!",
		choiceKey, att.Card.Name, choice.Option1.Name, choice.Option2.Name))

	opt := pickChoiceOption(def.Card, choice)
	if opt.IsChart {
		m.emit(newEvent(EventChart, "%s chooses: %s!", att.Card.Name, opt.Name))
		m.resolveChartMove(att, def, opt.ChartRef)
		return
	}
	m.attemptChoiceMove(att, def, opt)
}

// attemptChoiceMove rolls a choice move. A success sends the opponent to the
// defense level of the move's number; a failure gives him Level 2 offense.
func (m *Match) attemptChoiceMove(att, def *WrestlerState, opt ChoiceOption) {
	roll := m.rollTwo().total()
	threshold := choiceThreshold(opt, def.Card)
	m.emit(newEvent(EventChart, "%s tries a %s! (roll %d, needs %d or lower, adjusted for opponent's %s)",
		att.Card.Name, opt.Name, roll, threshold, opt.StatType))

	if roll > threshold {
		m.emit(newEvent(EventDefense, "The %s fails! %s takes over!", opt.Name, def.Card.Name))
		m.setOffense(def, 2)
		return
	}
	m.emit(newEvent(EventMove, "%s hits the %s!", att.Card.Name, opt.Name))
	m.resolveNormalDefense(att, def, Move{Name: opt.Name, Power: opt.Power, DefLevel: opt.Power})
}
