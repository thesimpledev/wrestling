package engine

// dqPossible reports whether anyone can be disqualified right now. There are
// no disqualifications in No DQ and cage matches, or while the referee is down.
func (m *Match) dqPossible() bool {
	return m.Type != MatchNoDQ && m.Type != MatchCage && !m.refDown
}

func (m *Match) explainNoDQ() {
	if m.refDown {
		m.emit(newEvent(EventDQ, "The referee is down, so nobody can be disqualified!"))
		return
	}
	m.emit(newEvent(EventDQ, "No disqualification in this match!"))
}

// dqCheck rolls two dice against a disqualification number and reports the
// roll and whether the wrestler is disqualified. It does not end the match.
func (m *Match) dqCheck(ws *WrestlerState, number int) (twoDice, bool) {
	roll := m.rollTwo()
	m.emit(newEvent(EventDQ, "%s rolls %d for disqualification (needs more than %d).", ws.Card.Name, roll.total(), number))
	if roll.total() > number {
		m.emit(newEvent(EventDQ, "%s avoids disqualification!", ws.Card.Name))
		return roll, false
	}
	m.emit(newEvent(EventDQ, "%s HAS BEEN DISQUALIFIED!", ws.Card.Name))
	return roll, true
}

// rollDQ rolls the wrestler's own Disqualification rating and ends the match
// if he is disqualified.
func (m *Match) rollDQ(ws *WrestlerState) bool {
	if !m.dqPossible() {
		m.explainNoDQ()
		return false
	}
	return m.rollDQAgainst(ws, ws.Card.DQ)
}

// rollDQAgainst rolls against a number given by a chart instead of the
// wrestler's own rating, and ends the match if he is disqualified.
func (m *Match) rollDQAgainst(ws *WrestlerState, number int) bool {
	if !m.dqPossible() {
		return false
	}
	roll, out := m.dqCheck(ws, number)
	if out {
		m.endMatch(m.otherWrestler(ws), ws, "dq", roll)
	}
	return out
}

// rollBothDQ makes both wrestlers roll their Disqualification rating. Both
// failing is a double disqualification. It reports whether the match ended.
func (m *Match) rollBothDQ(first, second *WrestlerState) bool {
	if !m.dqPossible() {
		m.explainNoDQ()
		return false
	}
	firstRoll, firstOut := m.dqCheck(first, first.Card.DQ)
	secondRoll, secondOut := m.dqCheck(second, second.Card.DQ)
	switch {
	case firstOut && secondOut:
		m.endDoubleDQ(secondRoll)
	case firstOut:
		m.endMatch(second, first, "dq", firstRoll)
	case secondOut:
		m.endMatch(first, second, "dq", secondRoll)
	}
	return firstOut || secondOut
}

// countOutsApply reports whether a wrestler can be counted out right now.
func (m *Match) countOutsApply() bool {
	return m.Type != MatchNoDQ && m.Type != MatchCage && !m.refDown
}

// resolveCountOut rolls the wrestler's PIN rating for a count-out and ends
// the match if he fails to beat the count.
func (m *Match) resolveCountOut(winner, outside *WrestlerState) {
	roll := m.rollTwo()
	number := outside.CurrentPIN
	m.emit(newEvent(EventCountOut, "%s rolls %d for count-out (PIN rating: %d).", outside.Card.Name, roll.total(), number))

	if roll.total() > number {
		m.emit(newEvent(EventCountOut, "%s beats the count!", outside.Card.Name))
		return
	}
	m.emit(newEvent(EventCountOut, "%s HAS BEEN COUNTED OUT!", outside.Card.Name))
	m.endMatch(winner, outside, "countout", roll)
}
