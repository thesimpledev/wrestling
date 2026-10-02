package engine

const maxFeudRerolls = 100

// feudAfterMatch sends a feud match to the Feud Table when the roll that
// ended it was doubles. you is the wrestler the table text calls "you": the
// winner of a pin or count-out, or the wrestler who was disqualified.
func (m *Match) feudAfterMatch(endingRoll twoDice, you, opponent *WrestlerState) {
	if !m.IsFeud || !endingRoll.doubles() {
		return
	}
	m.emit(newEvent(EventMatchEnd, ""))
	m.emit(newEvent(EventMatchEnd, "The final roll was doubles (%d and %d)! THE FEUD CONTINUES AFTER THE BELL!",
		endingRoll.first, endingRoll.second))

	outcome, roll := m.rollFeudOutcome(you, opponent)
	m.emit(newEvent(EventMatchEnd, "[Feud Table, roll %d, \"you\" is %s] %s", roll, you.Card.Name, outcome.Text))
	m.result.FeudText = outcome.Text
	m.applyFeudOutcome(outcome, you, opponent)
}

// rollFeudOutcome rolls on the Feud Table, rolling again while the result
// involves an ally that the wrestler did not bring to ringside.
func (m *Match) rollFeudOutcome(you, opponent *WrestlerState) (*FeudOutcome, int) {
	yourAlly := m.Sides[m.sideOf(you)].Ally != nil
	theirAlly := m.Sides[m.sideOf(opponent)].Ally != nil

	for attempt := 0; attempt < maxFeudRerolls; attempt++ {
		roll := m.rollTwo().total()
		outcome := LookupFeud(roll)
		if outcome.usableWith(yourAlly, theirAlly) {
			return outcome, roll
		}
		m.emit(newEvent(EventMatchEnd, "[Feud Table, roll %d] No ally at ringside for that result, rolling again.", roll))
	}
	return &FeudTable[0], FeudTable[0].MinRoll
}

func (o *FeudOutcome) usableWith(yourAlly, theirAlly bool) bool {
	if o.NeedsYourAlly && !yourAlly {
		return false
	}
	return !o.NeedsOpponentAlly || theirAlly
}

func (m *Match) applyFeudOutcome(outcome *FeudOutcome, you, opponent *WrestlerState) {
	switch outcome.Type {
	case FeudAttackedByLoser, FeudOpponentAlly:
		m.injure(you, outcome.InjuryDays)
	case FeudPostMatchAttack:
		m.injure(opponent, outcome.InjuryDays)
	case FeudAllyDoubleTeam:
		m.emit(newEvent(EventMatchEnd, "A new rivalry is born!"))
	case FeudFourManBrawl:
		m.emit(newEvent(EventMatchEnd, "The commissioner books a tag team super match!"))
	case FeudGangAttack:
		m.gangAttack(you, opponent)
	}
}

func (m *Match) injure(ws *WrestlerState, cards int) {
	m.result.InjuredWrestler = ws.Card.Name
	m.result.InjuryCards = cards
	m.emit(newEvent(EventMatchEnd, "%s IS INJURED FOR %d FIGHT CARD(S)!", ws.Card.Name, cards))
}

// gangAttack rolls one die for the opponent's injury and one die for the
// suspension of the attacker and his ringside ally.
func (m *Match) gangAttack(you, opponent *WrestlerState) {
	m.injure(opponent, m.rollOne())

	suspended := []string{you.Card.Name}
	if ally := m.Sides[m.sideOf(you)].Ally; ally != nil {
		suspended = append(suspended, ally.Name)
	}
	m.result.SuspendedWrestlers = suspended
	m.result.SuspensionCards = m.rollOne()
	for _, name := range suspended {
		m.emit(newEvent(EventMatchEnd, "%s IS SUSPENDED FOR %d FIGHT CARD(S)!", name, m.result.SuspensionCards))
	}
}
