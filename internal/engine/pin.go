package engine

// resolvePIN handles a pin attempt on pinned by pinner.
func (m *Match) resolvePIN(pinner, pinned *WrestlerState) {
	if m.refDown {
		m.uncountedPin(pinner, pinned, "PIN ATTEMPT, but the referee is still down! No count!")
		return
	}
	m.pinAttempt(pinner, pinned, 0)
}

// pinAttempt rolls the pinned wrestler's PIN rating plus bonus. A kick-out
// leaves the wrestler on offense rolling on Level 3.
func (m *Match) pinAttempt(pinner, pinned *WrestlerState, bonus int) {
	if !m.pinRoll(pinned, bonus) {
		m.kickOut(pinner, pinned)
		return
	}
	if m.Type == MatchTag && m.tryPinSave(m.sideOf(pinned)) {
		return
	}
	if !m.over {
		m.endMatch(pinner, pinned, "pinfall")
	}
}

// pinRoll rolls two dice against the pinned wrestler's PIN rating plus bonus
// and reports whether he stays down.
func (m *Match) pinRoll(pinned *WrestlerState, bonus int) bool {
	roll := m.rollTwo().total()
	threshold := pinned.CurrentPIN + bonus
	isPinned := roll <= threshold
	m.emit(pinEvent(pinned.Card.Name, roll, threshold, isPinned))
	return isPinned
}

func (m *Match) kickOut(pinner, pinned *WrestlerState) {
	m.addFatigue(pinned)
	m.setOffense(pinner, 3)
}

// uncountedPin is a pin with no referee to count it: fatigue is still added
// and the wrestler on offense carries on at Level 3.
func (m *Match) uncountedPin(pinner, pinned *WrestlerState, text string) {
	m.emit(Event{Type: EventPin, Text: text})
	m.addFatigue(pinned)
	m.setOffense(pinner, 3)
}

// resolveFinisher handles a finisher: the opponent rolls his PIN rating plus
// the finisher rating.
func (m *Match) resolveFinisher(att, def *WrestlerState) {
	finisher := att.Card.Finisher
	m.emit(finisherEvent(att.Card.Name, finisher.Name))

	rating, connects := m.finisherRating(att, def)
	if !connects {
		return
	}
	if m.refDown {
		m.uncountedPin(att, def, att.Card.Name+" hits the "+finisher.Name+", but the referee is still down! No count!")
		return
	}
	m.pinAttempt(att, def, rating)
}

// finisherRating returns the rating to add to the opponent's PIN. For a roll
// finisher the die is the rating, and a roll outside its range is a miss that
// hands the opponent Level 3 offense.
func (m *Match) finisherRating(att, def *WrestlerState) (int, bool) {
	finisher := att.Card.Finisher
	if !finisher.IsRoll {
		return finisher.Rating, true
	}

	roll := m.rollOne()
	m.emit(newEvent(EventFinisher, "%s rolls for the finisher: %d! (needs %d-%d)",
		att.Card.Name, roll, finisher.RollMin, finisher.RollMax))
	if roll < finisher.RollMin || roll > finisher.RollMax {
		m.emit(newEvent(EventFinisher, "The %s misses! %s moves out of the way and takes over!", finisher.Name, def.Card.Name))
		m.setOffense(def, 3)
		return 0, false
	}
	m.emit(newEvent(EventFinisher, "The %s connects with a rating of +%d!", finisher.Name, roll))
	return roll, true
}
