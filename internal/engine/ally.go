package engine

const (
	defaultDistractorRating = 5
	distractionMinPin       = 4
	ringsideAllyAttackRoll  = 6
	ringsideAllyDQNumber    = 6
)

func distractorRating(ally *WrestlerCard) int {
	if ally.Distractor <= 0 {
		return defaultDistractorRating
	}
	return ally.Distractor
}

// shouldDistract is the simulator's decision to have the ringside ally
// distract the referee: once per match, when a pin is a real danger, and
// keeping the distraction back while interference is still the better call.
func (m *Match) shouldDistract(sideIdx int) bool {
	side := m.Sides[sideIdx]
	if side.Ally == nil || m.distractionUsed[sideIdx] {
		return false
	}
	pin := side.Active().CurrentPIN
	if pin < distractionMinPin {
		return false
	}
	return pin < interferencePinDanger || m.interferenceUsed[sideIdx]
}

// tryDistraction has the ringside ally try to distract the referee before a
// pin is rolled, on the ally's own Distractor rating. It reports whether the
// pin was avoided.
func (m *Match) tryDistraction(pinnedIdx int) bool {
	if !m.shouldDistract(pinnedIdx) {
		return false
	}
	m.distractionUsed[pinnedIdx] = true

	side := m.Sides[pinnedIdx]
	rating := distractorRating(side.Ally)
	roll := m.rollTwo().total()
	m.emit(newEvent(EventDistraction, "%s tries to distract the referee! (roll %d, needs %d or lower)", side.Ally.Name, roll, rating))

	if roll > rating {
		m.emit(newEvent(EventDistraction, "The distraction fails! The referee orders %s to leave!", side.Ally.Name))
		return false
	}
	m.emit(newEvent(EventDistraction, "The distraction works! The referee is distracted and the pin count is broken!"))
	m.addFatigue(side.Active())
	m.setOffense(m.Sides[1-pinnedIdx].Active(), 3)
	return true
}

// ringsideAllyInterferes handles the thrower's ally at ringside when the
// opponent goes out of the ring. On 6 or less the ally attacks; it reports
// whether that happened, which replaces the Out of the Ring chart result.
func (m *Match) ringsideAllyInterferes(thrower, outside *WrestlerState, roll int) bool {
	ally := m.Sides[m.sideOf(thrower)].Ally
	if ally == nil {
		return false
	}
	if roll > ringsideAllyAttackRoll {
		m.emit(newEvent(EventInterference, "The referee prevents the ringside ally from interfering!"))
		return false
	}

	m.emit(newEvent(EventInterference, "%s is attacked outside the ring by %s!", outside.Card.Name, ally.Name))
	m.emit(newEvent(EventInterference, "%s smashes %s into the steel post!", ally.Name, outside.Card.Name))
	m.emit(newEvent(EventDQ, "%s and %s may be disqualified!", thrower.Card.Name, ally.Name))
	if m.rollDQAgainst(thrower, ringsideAllyDQNumber) {
		return true
	}
	m.emit(newEvent(EventInterference, "%s tosses %s back into the ring to the waiting hands of %s!", ally.Name, outside.Card.Name, thrower.Card.Name))
	m.setOffense(thrower, 3)
	return true
}
