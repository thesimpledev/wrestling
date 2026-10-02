package engine

import "fmt"

// MatchType defines the kind of match being simulated.
type MatchType int

const (
	MatchSingles MatchType = iota
	MatchTag
	MatchCage
	MatchNoDQ
)

// WrestlerState tracks runtime state for a wrestler during a match.
type WrestlerState struct {
	Card       *WrestlerCard
	CurrentPIN int // Starts at card's PINAdv, increases with fatigue
	Injured    bool
	InjuryLeft int // Fight cards remaining on injury
}

// Side represents one corner of the match (one or more wrestlers in tag).
type Side struct {
	Wrestlers    []*WrestlerState
	ActiveIndex  int           // Which wrestler is currently in the ring
	PinSavesUsed int           // Max 2 per match in tag matches
	Ally         *WrestlerCard // Optional ringside ally (enables interference/distraction)

	// RegularPartners marks a tag team as regular partners, who tag out on
	// defense more easily and may attempt pin saves.
	RegularPartners bool
}

// Active returns the wrestler currently in the ring for this side.
func (s *Side) Active() *WrestlerState {
	return s.Wrestlers[s.ActiveIndex]
}

// MatchResult describes how a match ended.
type MatchResult struct {
	WinningSide     int    // 0 or 1, or -1 when there is no winner
	Winner          string // Wrestler name, empty when there is no winner
	Loser           string // Wrestler name
	Method          string // "pinfall", "dq", "countout", "double dq"
	FeudText        string // Post-match feud narration (if any)
	InjuredWrestler string // Name of wrestler injured in feud
	InjuryCards     int    // Fight cards of injury

	SuspendedWrestlers []string // Names suspended after a feud gang attack
	SuspensionCards    int      // Fight cards of suspension
}

// Match holds all state for a match in progress.
type Match struct {
	Type   MatchType
	Sides  [2]*Side
	Events []Event

	onOffense int // 0 or 1 — which side is attacking
	offLevel  int // Current offense level (0, 1, or 2 — maps to Level 1, 2, 3)

	refDown      bool
	refDownTurns int

	turnCount int
	over      bool
	result    *MatchResult

	interferenceUsed [2]bool // Each side can use interference once per match
	distractionUsed  [2]bool // Each side can use distraction once per match
	IsFeud           bool    // Whether this is a feud match

	dice Dice
}

// NewMatch creates a match between two wrestlers (singles).
func NewMatch(card1, card2 *WrestlerCard) *Match {
	m := &Match{
		Type: MatchSingles,
		dice: randomDice{},
		Sides: [2]*Side{
			{Wrestlers: []*WrestlerState{{Card: card1, CurrentPIN: card1.PINAdv}}},
			{Wrestlers: []*WrestlerState{{Card: card2, CurrentPIN: card2.PINAdv}}},
		},
	}
	return m
}

// NewTagMatch creates a tag team match.
func NewTagMatch(team1a, team1b, team2a, team2b *WrestlerCard) *Match {
	m := &Match{
		Type: MatchTag,
		dice: randomDice{},
		Sides: [2]*Side{
			{Wrestlers: []*WrestlerState{
				{Card: team1a, CurrentPIN: team1a.PINAdv},
				{Card: team1b, CurrentPIN: team1b.PINAdv},
			}},
			{Wrestlers: []*WrestlerState{
				{Card: team2a, CurrentPIN: team2a.PINAdv},
				{Card: team2b, CurrentPIN: team2b.PINAdv},
			}},
		},
	}
	return m
}

// InitForMatchType adjusts wrestler state based on match type.
// Call after setting match.Type if not singles.
func (m *Match) InitForMatchType() {
	if m.Type == MatchCage {
		for _, side := range m.Sides {
			for _, ws := range side.Wrestlers {
				ws.CurrentPIN = ws.Card.Cage
			}
		}
	}
}

// ApplyInjuries adds +2 to PIN for injured wrestlers. Called before Run().
func (m *Match) ApplyInjuries(isInjured func(name string) bool) {
	for _, side := range m.Sides {
		for _, ws := range side.Wrestlers {
			if isInjured(ws.Card.Name) {
				ws.CurrentPIN += 2
				ws.Injured = true
			}
		}
	}
}

func (m *Match) attacker() *WrestlerState { return m.Sides[m.onOffense].Active() }
func (m *Match) defender() *WrestlerState { return m.Sides[1-m.onOffense].Active() }
func (m *Match) Over() bool               { return m.over }
func (m *Match) Result() *MatchResult     { return m.result }

func (m *Match) emit(e Event) {
	m.Events = append(m.Events, e)
}

// Run simulates the entire match from start to finish.
func (m *Match) Run() []Event {
	m.rollForInitiative()

	for !m.over {
		m.turnCount++
		if m.turnCount > 500 {
			m.emit(newEvent(EventMatchEnd, "Match ends in a draw: time limit exceeded!"))
			m.over = true
			break
		}
		m.executeTurn()
	}

	return m.Events
}

func (m *Match) rollForInitiative() {
	r1 := m.rollOne()
	r2 := m.rollOne()
	name1 := m.Sides[0].Active().Card.Name
	name2 := m.Sides[1].Active().Card.Name

	m.emit(Event{Type: EventMatchStart, Text: CommentaryMatchStart(name1, name2)})
	m.emit(newEvent(EventRoll, "%s rolls %d, %s rolls %d for initiative.", name1, r1, name2, r2))

	if r2 > r1 {
		m.onOffense = 1
	} else {
		m.onOffense = 0
	}
	m.offLevel = 0 // Start at Level 1

	m.emit(newEvent(EventMatchStart, "%s starts on offense!", m.attacker().Card.Name))
}

// levelIndex converts a card level number (1 to 3) to an offense or defense grid index.
func levelIndex(level int) int {
	switch {
	case level <= 1:
		return 0
	case level >= 3:
		return 2
	default:
		return 1
	}
}

// betterRating reports whether agility or power rating a beats rating b.
// The rulebook rates -5 as excellent and +5 as poor.
func betterRating(a, b int) bool {
	return a < b
}

func (m *Match) setOffense(ws *WrestlerState, level int) {
	m.onOffense = m.sideOf(ws)
	m.offLevel = levelIndex(level)
}

func (m *Match) addFatigue(ws *WrestlerState) {
	ws.CurrentPIN++
	m.emit(newEvent(EventFatigue, "%s's PIN rating increases to %d from fatigue.", ws.Card.Name, ws.CurrentPIN))
}

// tickReferee counts down the moves a downed referee misses and brings him
// back once he has missed them all.
func (m *Match) tickReferee() {
	if !m.refDown {
		return
	}
	if m.refDownTurns > 0 {
		m.refDownTurns--
		return
	}
	m.refDown = false
	m.emit(newEvent(EventRefRecover, "The referee recovers and is back on his feet!"))
}

const maxMoveRerolls = 50

func (m *Match) moveAllowed(move Move) bool {
	if move.HasTag(TagTagTeam) {
		return m.Type == MatchTag
	}
	if move.HasTag(TagSingles) {
		return m.Type != MatchTag
	}
	return true
}

func (m *Match) firstAllowedSlot(moves [6]Move) (int, bool) {
	for slot, move := range moves {
		if m.moveAllowed(move) {
			return slot, true
		}
	}
	return 0, false
}

// rollOffenseMove rolls on the attacker's current offense level, rolling again
// while the result is a tag-only move in a singles match or the reverse.
func (m *Match) rollOffenseMove(att *WrestlerState) (Move, int) {
	moves := att.Card.Offense[m.offLevel]
	roll := m.rollOne()
	fallback, anyAllowed := m.firstAllowedSlot(moves)
	if !anyAllowed {
		return moves[roll-1], roll
	}
	for attempt := 0; attempt < maxMoveRerolls && !m.moveAllowed(moves[roll-1]); attempt++ {
		m.emit(newEvent(EventRoll, "%s rolls %s, which does not apply in this match, and rolls again.",
			att.Card.Name, moves[roll-1].Name))
		roll = m.rollOne()
	}
	if !m.moveAllowed(moves[roll-1]) {
		roll = fallback + 1
	}
	return moves[roll-1], roll
}

func (m *Match) executeTurn() {
	m.tickReferee()

	if m.Type == MatchTag {
		m.maybeTagOnOffense()
	}

	att := m.attacker()
	def := m.defender()
	move, offRoll := m.rollOffenseMove(att)

	m.emit(Event{
		Type:     EventMove,
		Text:     fmt.Sprintf("%s (Level %d, roll %d): %s!", att.Card.Name, m.offLevel+1, offRoll, move.Name),
		Attacker: att.Card.Name,
		Defender: def.Card.Name,
		Roll:     offRoll,
		Level:    m.offLevel + 1,
	})

	m.resolveMove(att, def, move)
}

func (m *Match) resolveMove(att, def *WrestlerState, move Move) {
	if !m.passesStatChecks(att, def, move) {
		return
	}
	if move.HasTag(TagDQ) {
		m.emit(newEvent(EventDQ, "%s goes for a dirty move while the referee is watching!", att.Card.Name))
		if m.rollDQ(att) {
			return
		}
	}
	if move.HasTag(TagAdd1) {
		def.CurrentPIN++
		m.emit(newEvent(EventFatigue, "Devastating move! %s's PIN rating increases to %d!", def.Card.Name, def.CurrentPIN))
	}

	switch {
	case move.HasTag(TagChart):
		m.resolveChartMove(att, def, move.ChartType)
	case move.HasTag(TagChoice):
		m.resolveChoice(att, def, move.ChoiceKey)
	case m.offLevel == 2 && move.IsFinisher():
		m.resolveFinisher(att, def)
	default:
		m.resolveNormalDefense(att, def, move)
	}
}

// passesStatChecks applies the (ag) and (pw) instructions: the move works only
// when the attacker's rating is the same as or better than the opponent's.
func (m *Match) passesStatChecks(att, def *WrestlerState, move Move) bool {
	if move.HasTag(TagAgility) && betterRating(def.Card.Agility, att.Card.Agility) {
		m.emit(newEvent(EventDefense, "%s's agility isn't good enough and %s counters!", att.Card.Name, def.Card.Name))
		m.setOffense(def, 2)
		return false
	}
	if move.HasTag(TagAgility) {
		m.emit(newEvent(EventMove, "%s is agile enough and the move connects!", att.Card.Name))
	}
	if move.HasTag(TagPower) && betterRating(def.Card.Power, att.Card.Power) {
		m.emit(newEvent(EventDefense, "%s isn't powerful enough and %s overpowers him!", att.Card.Name, def.Card.Name))
		m.setOffense(def, 2)
		return false
	}
	if move.HasTag(TagPower) {
		m.emit(newEvent(EventMove, "%s is powerful enough and the move connects!", att.Card.Name))
	}
	return true
}

// resolveChartMove sends the defender to a chart. In a cage match nobody can
// leave the ring, so out of the ring becomes "face into cage 3" and the
// defender rolls on his Level 3 defense.
func (m *Match) resolveChartMove(att, def *WrestlerState, chartType string) {
	if m.Type == MatchCage && chartType == "ring" {
		m.emit(newEvent(EventMove, "%s smashes %s face-first into the cage! - 3", att.Card.Name, def.Card.Name))
		m.onOffense = m.sideOf(att)
		m.resolveNormalDefense(att, def, Move{Name: "Face into Cage", Power: 3, DefLevel: 3})
		return
	}
	m.resolveChart(att, def, chartType)
}

func (m *Match) resolveNormalDefense(att, def *WrestlerState, move Move) {
	defLevel := levelIndex(move.DefLevel)

	defRoll := m.rollOne()
	outcome := def.Card.Defense[defLevel][defRoll-1]

	m.emit(Event{
		Type:     EventDefense,
		Text:     fmt.Sprintf("%s (Defense Level %d, roll %d): %s", def.Card.Name, defLevel+1, defRoll, outcome.Type),
		Defender: def.Card.Name,
		Roll:     defRoll,
		Level:    defLevel + 1,
	})

	m.resolveDefense(outcome)
}

// resolveDefense applies a defense result. The number after dazed, hurt or
// down is the offense level the attacker rolls on next.
func (m *Match) resolveDefense(outcome DefenseOutcome) {
	def := m.defender()
	if m.tagOutOnDefense(def, outcome) {
		return
	}

	switch outcome.Type {
	case DefDazed, DefHurt:
		m.emit(defenseEvent(def.Card.Name, outcome.Type))
		m.offLevel = levelIndex(outcome.Power)
	case DefDown:
		m.resolveDown(def, outcome)
	case DefReversal:
		m.emit(reversalEvent(def.Card.Name))
		m.switchOffense()
		m.offLevel = levelIndex(outcome.Power)
	case DefPIN:
		m.resolvePinPredicament(def)
	}
}

func (m *Match) resolveDown(def *WrestlerState, outcome DefenseOutcome) {
	m.emit(defenseEvent(def.Card.Name, DefDown))

	defIdx := m.sideOf(def)
	if outcome.Power == 3 && m.shouldUseInterference(defIdx) {
		m.emit(newEvent(EventInterference, "%s calls for outside interference!", def.Card.Name))
		m.resolveInterference(defIdx, false)
		return
	}

	m.offLevel = levelIndex(outcome.Power)
	if outcome.Power == 3 && outcome.HasTag(TagLeave) {
		m.offerToLeaveRing(def)
	}
}

// offerToLeaveRing handles the (lv) option, which does not exist in a cage.
// The simulator leaves when the wrestler's Ring rating is A or B, and he
// rolls the chart on his own rating.
func (m *Match) offerToLeaveRing(def *WrestlerState) {
	if m.Type == MatchCage {
		return
	}
	m.emit(newEvent(EventChart, "%s has the option to leave the ring!", def.Card.Name))
	if def.Card.Ring > RatingB {
		return
	}
	m.emit(newEvent(EventChart, "%s rolls out of the ring!", def.Card.Name))
	m.resolveChart(m.attacker(), def, "ring")
}

func (m *Match) resolvePinPredicament(def *WrestlerState) {
	att := m.attacker()
	defIdx := m.sideOf(def)

	if m.shouldUseInterference(defIdx) {
		m.emit(newEvent(EventInterference, "%s calls for outside interference!", def.Card.Name))
		m.resolveInterference(defIdx, true)
		return
	}

	m.emit(newEvent(EventPin, "%s is in a pinning predicament!", def.Card.Name))
	if m.tryDistraction(defIdx) {
		return
	}
	m.resolvePIN(att, def)
}

func (m *Match) switchOffense() {
	m.onOffense = 1 - m.onOffense
}

// ─── CHART RESOLUTION ───────────────────────────────────────────────────────

// chartFor returns a chart and the rating the wrestler rolling on it uses.
func chartFor(chartType string, roller *WrestlerCard) (ChartTable, Rating, bool) {
	switch chartType {
	case "ropes":
		return RopesChart, roller.Ropes, true
	case "turnbuckle":
		return TurnbuckleChart, roller.Turnbuckle, true
	case "ring":
		return OutOfRingChart, roller.Ring, true
	case "deathjump":
		return DeathjumpChart, roller.Deathjump, true
	default:
		return nil, RatingC, false
	}
}

// resolveChart has def roll on a chart that att sent him to.
func (m *Match) resolveChart(att, def *WrestlerState, chartType string) {
	chart, rating, known := chartFor(chartType, def.Card)
	if !known {
		m.emit(newEvent(EventChart, "Unknown chart type %q, continuing normally.", chartType))
		m.offLevel = 2
		return
	}

	roll := m.rollTwo().total()
	if chartType == "ring" && m.ringsideAllyInterferes(att, def, roll) {
		return
	}

	outcome := chart.Lookup(rating, roll)
	if outcome == nil {
		m.emit(newEvent(EventChart, "Chart lookup failed for %s rating %s roll %d.", chartType, rating, roll))
		return
	}

	m.emit(Event{
		Type: EventChart,
		Text: fmt.Sprintf("[%s Chart, Rating %s, roll %d] %s", chartType, rating, roll, outcome.Text),
		Roll: roll,
	})

	m.resolveChartOutcome(att, def, outcome, chartType)
}

func (m *Match) getRating(ws *WrestlerState, ratingType string) Rating {
	switch ratingType {
	case "ropes":
		return ws.Card.Ropes
	case "turnbuckle":
		return ws.Card.Turnbuckle
	case "ring":
		return ws.Card.Ring
	case "deathjump":
		return ws.Card.Deathjump
	default:
		return RatingC
	}
}
