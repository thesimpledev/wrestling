package engine

const (
	maxPinSaves             = 2
	fatigueBeforeTagOut     = 3
	tagRollRegularPartners  = 6
	tagRollOtherTeams       = 4
	doubleDQNumberOnPinSave = 4
)

// maybeTagOnOffense lets the attacking side tag in a fresh partner once the
// man in the ring has built up fatigue. The partner rolls on the same level.
func (m *Match) maybeTagOnOffense() {
	side := m.Sides[m.onOffense]
	if len(side.Wrestlers) < 2 {
		return
	}
	active := side.Active()
	if active.CurrentPIN < active.Card.PINAdv+fatigueBeforeTagOut {
		return
	}
	side.ActiveIndex = 1 - side.ActiveIndex
	m.emit(newEvent(EventTagIn, "%s tags out! %s enters the ring!", active.Card.Name, side.Active().Card.Name))
}

// tagOutOnDefense handles a defense result marked (tag) in a tag match: the
// defender tries to reach his partner, who comes in on Level 1 offense. It
// reports whether the tag was made, which replaces the defense result.
func (m *Match) tagOutOnDefense(def *WrestlerState, outcome DefenseOutcome) bool {
	side := m.Sides[m.sideOf(def)]
	if m.Type != MatchTag || len(side.Wrestlers) < 2 || !outcome.HasTag(TagTagTeam) {
		return false
	}

	number := tagRollOtherTeams
	if side.RegularPartners {
		number = tagRollRegularPartners
	}
	roll := m.rollTwo().total()
	m.emit(newEvent(EventTagAttempt, "%s reaches for a tag! Rolls %d (needs %d or less)...", def.Card.Name, roll, number))
	if roll > number {
		m.emit(newEvent(EventTagAttempt, "%s can't reach the tag!", def.Card.Name))
		return false
	}

	side.ActiveIndex = 1 - side.ActiveIndex
	m.emit(newEvent(EventTagIn, "TAG MADE! %s enters the ring fresh!", side.Active().Card.Name))
	m.setOffense(side.Active(), 1)
	return true
}

// tryPinSave lets the pinned wrestler's regular partner try to break up the
// pin. It reports whether the pin was broken up (or the match ended another
// way); false means the pin stands.
func (m *Match) tryPinSave(pinnedSide int) bool {
	side := m.Sides[pinnedSide]
	if len(side.Wrestlers) < 2 || !side.RegularPartners {
		return false
	}
	if side.PinSavesUsed >= maxPinSaves {
		m.emit(newEvent(EventPinSave, "No more pin saves available, both already used!"))
		return false
	}

	side.PinSavesUsed++
	roll := m.rollTwo().total()
	outcome := LookupPinSave(roll)
	m.emit(newEvent(EventPinSave, "[Pin Save, roll %d] %s", roll, outcome.Text))
	return m.applyPinSave(pinnedSide, outcome.Type)
}

func (m *Match) applyPinSave(pinnedSide int, kind PinSaveOutcomeType) bool {
	pinned := m.Sides[pinnedSide].Active()
	opponent := m.Sides[1-pinnedSide].Active()

	switch kind {
	case PinSaveFailed:
		return false
	case PinSaveSaved:
		m.addFatigue(pinned)
		m.setOffense(opponent, 3)
	case PinSaveReversed:
		m.addFatigue(pinned)
		m.reversedPin(pinned, opponent)
	case PinSaveBrawl:
		m.pinSaveBrawl(pinned, opponent)
	case PinSaveInterference:
		m.resolveInterference(pinnedSide, true)
	}
	return true
}

// reversedPin is the pin save that turns the pin over: the opponent rolls his
// PIN with no save of his own.
func (m *Match) reversedPin(nowOnTop, nowPinned *WrestlerState) {
	m.emit(newEvent(EventPin, "%s is now in a pinning predicament!", nowPinned.Card.Name))
	roll := m.rollTwo()
	number := nowPinned.CurrentPIN
	isPinned := roll.total() <= number
	m.emit(pinEvent(nowPinned.Card.Name, roll.total(), number, isPinned))
	if isPinned {
		m.endMatch(nowOnTop, nowPinned, "pinfall", roll)
		return
	}
	m.kickOut(nowOnTop, nowPinned)
}

// pinSaveBrawl is the pin save where all four wrestlers brawl: a double
// disqualification on 4 or less, otherwise one die decides who takes over.
func (m *Match) pinSaveBrawl(pinned, opponent *WrestlerState) {
	m.emit(newEvent(EventDQ, "A wild brawl erupts with all the wrestlers! The referee may disqualify both teams."))
	if m.dqPossible() {
		roll := m.rollTwo()
		m.emit(newEvent(EventDQ, "Double disqualification roll: %d (needs more than %d).", roll.total(), doubleDQNumberOnPinSave))
		if roll.total() <= doubleDQNumberOnPinSave {
			m.endDoubleDQ(roll)
			return
		}
	}
	m.addFatigue(pinned)
	m.brawl(pinned, opponent)
}
