package engine

import "fmt"

// chartCall is one result row being applied. thrower sent roller to the chart;
// roller is the wrestler who rolled on it ("you" in the chart text).
type chartCall struct {
	thrower   *WrestlerState
	roller    *WrestlerState
	outcome   *ChartOutcome
	chartType string
}

var chartHandlers map[ChartOutcomeType]func(*Match, chartCall)

func init() {
	chartHandlers = map[ChartOutcomeType]func(*Match, chartCall){
		ChartRollOnOffense:    (*Match).chartRollerOffense,
		ChartOppRollOnOffense: (*Match).chartThrowerOffense,
		ChartRollOnDefense:    (*Match).chartThrowerDefends,
		ChartRollPIN:          (*Match).chartThrowerPinned,
		ChartRollYourPIN:      (*Match).chartRollerPinned,
		ChartRollAgain:        (*Match).chartRollAgain,
		ChartRollDQ:           (*Match).chartRollerDQ,
		ChartOppRollDQ:        (*Match).chartThrowerDQ,
		ChartBothRollDQ:       (*Match).chartBothDQ,
		ChartRollCountOut:     (*Match).chartCountOut,
		ChartPowerCheck:       (*Match).chartPowerCheck,
		ChartBetterRating:     (*Match).chartBetterRating,
		ChartAgilityCheck:     (*Match).chartAgilityCheck,
		ChartRefDown:          (*Match).chartRefDown,
		ChartOppRollOnChart:   (*Match).chartThrowerOnChart,
	}
}

func (m *Match) resolveChartOutcome(att, def *WrestlerState, outcome *ChartOutcome, chartType string) {
	handler, ok := chartHandlers[outcome.Type]
	if !ok {
		panic(fmt.Sprintf("engine: no handler for chart outcome type %d", outcome.Type))
	}
	handler(m, chartCall{thrower: att, roller: def, outcome: outcome, chartType: chartType})
}

func (m *Match) chartRollerOffense(c chartCall) {
	m.setOffense(c.roller, c.outcome.Level)
}

func (m *Match) chartThrowerOffense(c chartCall) {
	m.setOffense(c.thrower, c.outcome.Level)
}

// chartThrowerDefends makes the thrower roll on his defense with the chart
// roller as the attacker.
func (m *Match) chartThrowerDefends(c chartCall) {
	level := levelIndex(c.outcome.Level)
	defRoll := m.rollOne()
	result := c.thrower.Card.Defense[level][defRoll-1]
	m.emit(Event{
		Type:     EventDefense,
		Text:     fmt.Sprintf("%s (Defense Level %d, roll %d): %s", c.thrower.Card.Name, level+1, defRoll, result.Type),
		Defender: c.thrower.Card.Name,
		Roll:     defRoll,
		Level:    level + 1,
	})
	m.onOffense = m.sideOf(c.roller)
	m.resolveDefense(result)
}

func (m *Match) chartThrowerPinned(c chartCall) {
	m.emit(newEvent(EventPin, "%s is in a pinning predicament!", c.thrower.Card.Name))
	m.resolvePIN(c.roller, c.thrower)
}

func (m *Match) chartRollerPinned(c chartCall) {
	m.emit(newEvent(EventPin, "%s is in a pinning predicament!", c.roller.Card.Name))
	m.resolvePIN(c.thrower, c.roller)
}

func (m *Match) chartRollAgain(c chartCall) {
	m.resolveChart(c.thrower, c.roller, c.chartType)
}

func (m *Match) chartRollerDQ(c chartCall) {
	if m.rollDQ(c.roller) {
		return
	}
	if c.outcome.ThenType == ChartRollOnOffense {
		m.setOffense(c.roller, c.outcome.ThenLevel)
	}
}

func (m *Match) chartThrowerDQ(c chartCall) {
	if m.rollDQ(c.thrower) {
		return
	}
	if c.outcome.ThenType == ChartOppRollOnOffense {
		m.setOffense(c.thrower, c.outcome.ThenLevel)
	}
}

func (m *Match) chartBothDQ(c chartCall) {
	m.emit(newEvent(EventDQ, "Both wrestlers may be disqualified!"))
	if m.rollDQ(c.thrower) {
		return
	}
	if m.rollDQ(c.roller) {
		return
	}
	m.brawl(c.roller, c.thrower)
}

// brawl rolls one die for a brawl: an even roll goes to evenWinner, an odd
// roll to oddWinner, and the winner rolls on Level 3 offense.
func (m *Match) brawl(evenWinner, oddWinner *WrestlerState) {
	roll := m.rollOne()
	winner := oddWinner
	if roll%2 == 0 {
		winner = evenWinner
	}
	m.emit(newEvent(EventChart, "%s wins the brawl! (roll %d)", winner.Card.Name, roll))
	m.setOffense(winner, 3)
}

func (m *Match) chartCountOut(c chartCall) {
	m.emit(newEvent(EventCountOut, "%s may be counted out!", c.roller.Card.Name))
	m.resolveCountOut(c.thrower, c.roller)
	if m.over {
		return
	}
	if c.outcome.AddFatigue {
		c.roller.CurrentPIN++
		m.emit(newEvent(EventFatigue, "%s's PIN rating increases to %d!", c.roller.Card.Name, c.roller.CurrentPIN))
	}
	m.setOffense(c.thrower, 3)
}

func (m *Match) chartPowerCheck(c chartCall) {
	if betterRating(c.roller.Card.Power, c.thrower.Card.Power) {
		m.emit(newEvent(EventChart, "%s is more powerful and knocks the opponent down with a shoulder tackle!", c.roller.Card.Name))
		m.setOffense(c.roller, 2)
		return
	}
	m.emit(newEvent(EventChart, "%s is overpowered! The opponent knocks him down with a shoulder tackle!", c.roller.Card.Name))
	m.setOffense(c.thrower, 2)
}

func (m *Match) chartAgilityCheck(c chartCall) {
	if betterRating(c.roller.Card.Agility, c.thrower.Card.Agility) {
		m.emit(newEvent(EventChart, "%s wins the struggle on the top rope with superior agility!", c.roller.Card.Name))
		m.setOffense(c.roller, 3)
		return
	}
	m.emit(newEvent(EventChart, "%s pushes %s off the top rope!", c.thrower.Card.Name, c.roller.Card.Name))
	m.setOffense(c.thrower, 3)
}

func (m *Match) chartBetterRating(c chartCall) {
	rollerRating := m.getRating(c.roller, c.outcome.RatingType)
	throwerRating := m.getRating(c.thrower, c.outcome.RatingType)
	if rollerRating < throwerRating {
		m.emit(newEvent(EventChart, "%s has the better %s rating and recovers first!", c.roller.Card.Name, c.outcome.RatingType))
		m.setOffense(c.roller, 3)
		return
	}
	m.emit(newEvent(EventChart, "%s recovers first!", c.thrower.Card.Name))
	m.setOffense(c.thrower, 3)
}

func (m *Match) chartRefDown(c chartCall) {
	m.refDown = true
	m.refDownTurns = m.rollTwo().total()
	m.emit(newEvent(EventRefDown, "THE REFEREE IS DOWN! He'll be out for %d moves!", m.refDownTurns))

	roll := m.rollOne()
	if roll%2 == 0 {
		m.emit(newEvent(EventChart, "%s takes advantage of the chaos! (roll %d)", c.roller.Card.Name, roll))
		m.setOffense(c.roller, 3)
		return
	}
	m.emit(newEvent(EventChart, "%s is still down and %s goes for the kill! (roll %d)", c.roller.Card.Name, c.thrower.Card.Name, roll))
	m.setOffense(c.thrower, 3)
}

func (m *Match) chartThrowerOnChart(c chartCall) {
	m.resolveChart(c.roller, c.thrower, c.outcome.ChartRef)
}
