package engine

const interferencePinDanger = 6

// shouldUseInterference is the simulator's decision to call in an ally: once
// per match, when the wrestler's PIN rating has become dangerous.
func (m *Match) shouldUseInterference(sideIdx int) bool {
	side := m.Sides[sideIdx]
	if side.Ally == nil || m.interferenceUsed[sideIdx] {
		return false
	}
	return side.Active().CurrentPIN >= interferencePinDanger
}

// interferenceCall is one roll on the Outside Interference chart. helped is
// the wrestler whose ally storms the ring ("you" in the chart text).
type interferenceCall struct {
	helped   *WrestlerState
	opponent *WrestlerState
	allyName string
	outcome  *InterferenceOutcome
}

// resolveInterference rolls on the Outside Interference chart for a side.
// fromPin says the interference was called while that side's wrestler was
// being pinned, in which case a broken-up pin still costs him fatigue.
func (m *Match) resolveInterference(sideIdx int, fromPin bool) {
	m.interferenceUsed[sideIdx] = true
	call := interferenceCall{
		helped:   m.Sides[sideIdx].Active(),
		opponent: m.Sides[1-sideIdx].Active(),
		allyName: m.interfererName(sideIdx),
	}

	roll := m.rollTwo().total()
	call.outcome = LookupInterference(roll)
	m.emit(newEvent(EventInterference, "%s storms the ring!", call.allyName))
	m.emit(newEvent(EventInterference, "[Interference Chart, roll %d] %s", roll, call.outcome.Text))

	if call.outcome.pinGoesAhead() {
		m.interferenceBackfires(call)
		return
	}
	if fromPin {
		m.addFatigue(call.helped)
	}
	m.interferenceHelps(call)
}

// interfererName names the ringside ally, or the tag partner when a pin save
// sends a tag match to the interference chart.
func (m *Match) interfererName(sideIdx int) string {
	side := m.Sides[sideIdx]
	if side.Ally != nil {
		return side.Ally.Name
	}
	if len(side.Wrestlers) > 1 {
		return side.Wrestlers[1-side.ActiveIndex].Card.Name
	}
	return "An ally"
}

func (o *InterferenceOutcome) pinGoesAhead() bool {
	return o.Type == InterfBackfire || o.Type == InterfBackfireFinish
}

// interferenceBackfires covers the rows where the opponent wins the brawl and
// the helped wrestler has to roll his PIN after all.
func (m *Match) interferenceBackfires(c interferenceCall) {
	m.emit(newEvent(EventInterference, "%s wins the brawl and throws %s out of the ring!", c.opponent.Card.Name, c.allyName))
	if c.outcome.Type == InterfBackfire {
		m.emit(newEvent(EventPin, "%s performs a big move and pins %s!", c.opponent.Card.Name, c.helped.Card.Name))
		m.resolvePIN(c.opponent, c.helped)
		return
	}
	m.emit(newEvent(EventFinisher, "%s motions to the crowd: %s time!", c.opponent.Card.Name, c.opponent.Card.Finisher.Name))
	m.resolvePINWithBonus(c.opponent, c.helped, c.opponent.Card.Finisher.Rating)
}

// interferenceHelps covers the rows where the ally's attack lands. Most carry
// a disqualification number the helped wrestler must beat first.
func (m *Match) interferenceHelps(c interferenceCall) {
	if c.outcome.Type == InterfDoubleTeam {
		m.addFatigue(c.opponent)
	}
	if c.outcome.DQThreshold > 0 && m.rollDQAgainst(c.helped, c.outcome.DQThreshold) {
		return
	}

	switch c.outcome.Type {
	case InterfDoubleTeam, InterfAttackAndPin:
		m.emit(newEvent(EventPin, "%s covers %s for the pin!", c.helped.Card.Name, c.opponent.Card.Name))
		m.resolvePIN(c.helped, c.opponent)
	case InterfFinisher:
		m.emit(newEvent(EventFinisher, "%s hits the %s on %s!", c.helped.Card.Name, c.helped.Card.Finisher.Name, c.opponent.Card.Name))
		m.resolvePINWithBonus(c.helped, c.opponent, c.helped.Card.Finisher.Rating)
	case InterfAttackAndL3:
		m.emit(newEvent(EventInterference, "%s recovers and attacks %s!", c.helped.Card.Name, c.opponent.Card.Name))
		m.setOffense(c.helped, 3)
	case InterfBrawl:
		m.brawl(c.helped, c.opponent)
	case InterfDistract:
		m.emit(newEvent(EventDistraction, "%s distracts the referee and is ordered to leave!", c.allyName))
		m.setOffense(c.opponent, 3)
	}
}
