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
}

// Active returns the wrestler currently in the ring for this side.
func (s *Side) Active() *WrestlerState {
	return s.Wrestlers[s.ActiveIndex]
}

// MatchResult describes how a match ended.
type MatchResult struct {
	WinningSide     int    // 0 or 1
	Winner          string // Wrestler name
	Loser           string // Wrestler name
	Method          string // "pinfall", "dq", "countout"
	FeudText        string // Post-match feud narration (if any)
	InjuredWrestler string // Name of wrestler injured in feud
	InjuryCards     int    // Fight cards of injury
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
			m.emit(newEvent(EventMatchEnd, "Match ends in a draw — time limit exceeded!"))
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

// resolveChartMove sends the defender to a chart, or into the cage wall when
// an out of the ring move is rolled in a cage match.
func (m *Match) resolveChartMove(att, def *WrestlerState, chartType string) {
	if m.Type == MatchCage && chartType == "ring" {
		m.emit(newEvent(EventMove, "%s smashes %s face-first into the cage! - 3", att.Card.Name, def.Card.Name))
		m.offLevel = 2
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
		m.resolveInterference(defIdx)
		return
	}

	m.offLevel = levelIndex(outcome.Power)
	if outcome.Power == 3 && outcome.HasTag(TagLeave) {
		m.offerToLeaveRing(def)
	}
}

// offerToLeaveRing handles the (lv) option. The simulator leaves when the
// wrestler's Ring rating is A or B, and he rolls the chart on his own rating.
func (m *Match) offerToLeaveRing(def *WrestlerState) {
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
		m.resolveInterference(defIdx)
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

func (m *Match) resolveChart(att, def *WrestlerState, chartType string) {
	var chart ChartTable
	var rating Rating

	switch chartType {
	case "ropes":
		chart = RopesChart
		rating = def.Card.Ropes
	case "turnbuckle":
		chart = TurnbuckleChart
		rating = def.Card.Turnbuckle
	case "ring":
		chart = OutOfRingChart
		rating = def.Card.Ring
	case "deathjump":
		chart = DeathjumpChart
		rating = def.Card.Deathjump
	default:
		m.emit(newEvent(EventChart, "Unknown chart type: %s — continuing normally.", chartType))
		m.offLevel = 2
		return
	}

	roll := m.rollTwo().total()

	// Ringside ally interaction: when defender is thrown out of ring
	// and attacker has an ally, ally can attack on rolls 6 or lower
	if chartType == "ring" {
		attSide := m.sideOf(att)
		if m.Sides[attSide].Ally != nil && roll <= 6 {
			m.resolveRingsideAllyAttack(att, def)
			return
		}
		if m.Sides[attSide].Ally != nil && roll > 6 {
			m.emit(newEvent(EventInterference, "The referee prevents the ringside ally from interfering!"))
		}
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

// ─── PIN / FINISHER / DQ / COUNTOUT ─────────────────────────────────────────

// rollDQ rolls disqualification for a wrestler. Returns true if DQ'd (match over).
func (m *Match) rollDQ(ws *WrestlerState) bool {
	if m.Type == MatchNoDQ {
		m.emit(newEvent(EventDQ, "No disqualification in this match!"))
		return false
	}
	if m.refDown {
		m.emit(newEvent(EventDQ, "The referee is down — no disqualification possible!"))
		return false
	}

	roll := m.rollTwo().total()
	threshold := ws.Card.DQ
	m.emit(newEvent(EventDQ, "%s rolls %d for disqualification (DQ rating: %d).", ws.Card.Name, roll, threshold))

	if roll <= threshold {
		m.emit(newEvent(EventDQ, "%s HAS BEEN DISQUALIFIED!", ws.Card.Name))
		// The other wrestler wins
		other := m.otherWrestler(ws)
		m.endMatch(other, ws, "dq")
		return true
	}
	m.emit(newEvent(EventDQ, "%s avoids disqualification!", ws.Card.Name))
	return false
}

// rollDQWithThreshold rolls DQ against a specific threshold (used by charts like interference).
func (m *Match) rollDQWithThreshold(ws *WrestlerState, threshold int) bool {
	if m.Type == MatchNoDQ || m.refDown {
		return false
	}
	roll := m.rollTwo().total()
	m.emit(newEvent(EventDQ, "%s rolls %d for disqualification (threshold: %d).", ws.Card.Name, roll, threshold))
	if roll <= threshold {
		m.emit(newEvent(EventDQ, "%s HAS BEEN DISQUALIFIED!", ws.Card.Name))
		other := m.otherWrestler(ws)
		m.endMatch(other, ws, "dq")
		return true
	}
	m.emit(newEvent(EventDQ, "%s avoids disqualification!", ws.Card.Name))
	return false
}

// resolveCountOut checks if a wrestler is counted out (uses PIN rating as threshold).
func (m *Match) resolveCountOut(att, def *WrestlerState) {
	if m.Type == MatchNoDQ || m.Type == MatchCage {
		m.emit(newEvent(EventCountOut, "No count-outs in this match type!"))
		return
	}
	if m.refDown {
		m.emit(newEvent(EventCountOut, "The referee is down — no count-out possible!"))
		return
	}

	roll := m.rollTwo().total()
	threshold := def.CurrentPIN
	m.emit(newEvent(EventCountOut, "%s rolls %d for count-out (PIN rating: %d).", def.Card.Name, roll, threshold))

	if roll <= threshold {
		m.emit(newEvent(EventCountOut, "%s HAS BEEN COUNTED OUT!", def.Card.Name))
		m.endMatch(att, def, "countout")
	} else {
		m.emit(newEvent(EventCountOut, "%s beats the count!", def.Card.Name))
	}
}

// ─── HELPERS ────────────────────────────────────────────────────────────────

func (m *Match) endMatch(winner, loser *WrestlerState, method string) {
	m.over = true
	m.result = &MatchResult{
		WinningSide: m.sideOf(winner),
		Winner:      winner.Card.Name,
		Loser:       loser.Card.Name,
		Method:      method,
	}
	m.emit(matchEndEvent(winner.Card.Name, loser.Card.Name, method))

	// Feud table check
	if m.IsFeud {
		m.resolveFeudTable(winner, loser)
	}
}

func (m *Match) sideOf(ws *WrestlerState) int {
	for i, side := range m.Sides {
		for _, w := range side.Wrestlers {
			if w == ws {
				return i
			}
		}
	}
	return 0
}

func (m *Match) otherWrestler(ws *WrestlerState) *WrestlerState {
	side := m.sideOf(ws)
	return m.Sides[1-side].Active()
}

// ─── TAG TEAM ───────────────────────────────────────────────────────────────

// maybeTagOnOffense gives the attacking side a chance to tag in their partner.
// AI logic: tag if current wrestler's PIN rating is getting high (fatigued).
func (m *Match) maybeTagOnOffense() {
	side := m.Sides[m.onOffense]
	if len(side.Wrestlers) < 2 {
		return
	}
	active := side.Active()
	// Tag if fatigued (PIN has increased by 3+ from base)
	if active.CurrentPIN >= active.Card.PINAdv+3 {
		oldName := active.Card.Name
		side.ActiveIndex = 1 - side.ActiveIndex
		newName := side.Active().Card.Name
		m.emit(newEvent(EventTagIn, "%s tags out! %s enters the ring!", oldName, newName))
		// Stay at same offense level
	}
}

// tryTagOnDefense attempts to tag out on defense. Roll 2d6, 4 or less = success.
func (m *Match) tryTagOnDefense() bool {
	defSideIdx := 1 - m.onOffense
	side := m.Sides[defSideIdx]
	if len(side.Wrestlers) < 2 {
		return false
	}

	roll := m.rollTwo().total()
	oldName := side.Active().Card.Name
	m.emit(newEvent(EventTagAttempt, "%s reaches for a tag! Rolls %d (needs 4 or less)...", oldName, roll))

	if roll <= 4 {
		side.ActiveIndex = 1 - side.ActiveIndex
		newName := side.Active().Card.Name
		m.emit(newEvent(EventTagIn, "TAG MADE! %s enters the ring fresh!", newName))
		// Successful tag = partner enters on Level 1 offense
		m.onOffense = defSideIdx
		m.offLevel = 0
		return true
	}
	m.emit(newEvent(EventTagAttempt, "%s can't reach the tag!", oldName))
	return false
}

// tryPinSave attempts a pin save in a tag match. Returns true if save was successful.
func (m *Match) tryPinSave(pinnedSide int) bool {
	side := m.Sides[pinnedSide]
	if len(side.Wrestlers) < 2 {
		return false
	}
	if side.PinSavesUsed >= 2 {
		m.emit(newEvent(EventPinSave, "No more pin saves available — both already used!"))
		return false
	}

	side.PinSavesUsed++
	roll := m.rollTwo().total()
	outcome := LookupPinSave(roll)
	if outcome == nil {
		return false
	}

	m.emit(newEvent(EventPinSave, "[Pin Save, roll %d] %s", roll, outcome.Text))

	switch outcome.Type {
	case PinSaveSaved:
		// Partner saves! Opponent rolls L3 offense
		m.onOffense = 1 - pinnedSide
		m.offLevel = 2
		return true

	case PinSaveReversed:
		// Reversed! Opponent rolls PIN instead
		opp := m.Sides[1-pinnedSide].Active()
		m.emit(newEvent(EventPin, "%s is now in a pinning predicament!", opp.Card.Name))
		// Don't recurse with pin saves — just do a straight PIN
		pinRoll := m.rollTwo().total()
		if pinRoll <= opp.CurrentPIN {
			m.emit(pinEvent(opp.Card.Name, pinRoll, opp.CurrentPIN, true))
			m.endMatch(side.Active(), opp, "pinfall")
		} else {
			m.emit(pinEvent(opp.Card.Name, pinRoll, opp.CurrentPIN, false))
			opp.CurrentPIN++
			m.onOffense = 1 - pinnedSide
			m.offLevel = 2
		}
		return true

	case PinSaveFailed:
		// Partner stopped — PIN proceeds normally
		return false

	case PinSaveBrawl:
		// Wild brawl, possible double DQ
		m.emit(newEvent(EventDQ, "A wild brawl erupts with all wrestlers!"))
		dqRoll := m.rollTwo().total()
		if dqRoll <= 4 {
			m.emit(newEvent(EventDQ, "DOUBLE DISQUALIFICATION! Both teams are thrown out!"))
			m.over = true
			m.result = &MatchResult{Method: "double dq"}
			m.emit(newEvent(EventMatchEnd, "The match ends in a DOUBLE DISQUALIFICATION!"))
			return true
		}
		brawlRoll := m.rollOne()
		if brawlRoll%2 == 0 {
			m.emit(newEvent(EventChart, "Team %s wins the brawl!", side.Active().Card.Name))
			m.onOffense = pinnedSide
		} else {
			m.emit(newEvent(EventChart, "The opponents win the brawl!"))
			m.onOffense = 1 - pinnedSide
		}
		m.offLevel = 2
		return true

	case PinSaveInterference:
		// Tag partner goes crazy — roll on the Interference Chart!
		m.resolveInterference(pinnedSide)
		return true
	}
	return false
}

// ─── OUTSIDE INTERFERENCE ────────────────────────────────────────────────────

// shouldUseInterference decides if the AI should call for interference.
func (m *Match) shouldUseInterference(sideIdx int) bool {
	side := m.Sides[sideIdx]
	if side.Ally == nil || m.interferenceUsed[sideIdx] {
		return false
	}
	ws := side.Active()
	// Use interference when PIN is very dangerous (>= 6) or fatigue is high
	return ws.CurrentPIN >= 6
}

// resolveInterference handles outside interference for a side.
// defIdx is the side index whose ally is interfering on their behalf.
func (m *Match) resolveInterference(defIdx int) {
	m.interferenceUsed[defIdx] = true
	def := m.Sides[defIdx].Active()
	att := m.Sides[1-defIdx].Active()
	ally := m.Sides[defIdx].Ally

	allyName := "An ally"
	if ally != nil {
		allyName = ally.Name
	}

	roll := m.rollTwo().total()
	outcome := LookupInterference(roll)
	if outcome == nil {
		m.emit(newEvent(EventInterference, "Interference fails — nothing happens."))
		return
	}

	m.emit(newEvent(EventInterference, "%s storms the ring!", allyName))
	m.emit(newEvent(EventInterference, "[Interference Chart, roll %d] %s", roll, outcome.Text))

	switch outcome.Type {
	case InterfDoubleTeam:
		att.CurrentPIN++
		m.emit(newEvent(EventFatigue, "%s's PIN rating increases to %d!", att.Card.Name, att.CurrentPIN))
		if m.rollDQWithThreshold(def, outcome.DQThreshold) {
			return
		}
		m.emit(newEvent(EventPin, "%s covers %s for the pin!", def.Card.Name, att.Card.Name))
		m.resolvePIN(def, att)

	case InterfFinisher:
		if m.rollDQWithThreshold(def, outcome.DQThreshold) {
			return
		}
		m.emit(newEvent(EventFinisher, "%s hits the %s on %s!", def.Card.Name, def.Card.Finisher.Name, att.Card.Name))
		pinRoll := m.rollTwo().total()
		threshold := att.CurrentPIN + def.Card.Finisher.Rating
		if pinRoll <= threshold {
			m.emit(pinEvent(att.Card.Name, pinRoll, threshold, true))
			m.endMatch(def, att, "pinfall")
		} else {
			m.emit(pinEvent(att.Card.Name, pinRoll, threshold, false))
			att.CurrentPIN++
			m.emit(newEvent(EventFatigue, "%s's PIN rating increases to %d!", att.Card.Name, att.CurrentPIN))
			m.onOffense = m.sideOf(att)
			m.offLevel = 2
		}

	case InterfAttackAndPin:
		if m.rollDQWithThreshold(def, outcome.DQThreshold) {
			return
		}
		m.emit(newEvent(EventPin, "%s covers %s for the pin!", def.Card.Name, att.Card.Name))
		m.resolvePIN(def, att)

	case InterfAttackAndL3:
		if m.rollDQWithThreshold(def, outcome.DQThreshold) {
			return
		}
		m.emit(newEvent(EventInterference, "%s recovers and attacks %s!", def.Card.Name, att.Card.Name))
		m.onOffense = defIdx
		m.offLevel = 2

	case InterfBrawl:
		if m.rollDQWithThreshold(def, outcome.DQThreshold) {
			return
		}
		brawlRoll := m.rollOne()
		if brawlRoll%2 == 0 {
			m.emit(newEvent(EventInterference, "%s flattens %s! %s takes over! (roll %d)", allyName, att.Card.Name, def.Card.Name, brawlRoll))
			m.onOffense = defIdx
		} else {
			m.emit(newEvent(EventInterference, "%s smashes %s and takes over! (roll %d)", att.Card.Name, allyName, brawlRoll))
			m.onOffense = 1 - defIdx
		}
		m.offLevel = 2

	case InterfDistract:
		m.emit(newEvent(EventDistraction, "%s distracts the referee, breaking the pin count! The referee orders him to leave!", allyName))
		m.onOffense = 1 - defIdx
		m.offLevel = 2

	case InterfBackfire:
		m.emit(newEvent(EventInterference, "%s storms the ring but %s wins the brawl and throws him out!", allyName, att.Card.Name))
		m.emit(newEvent(EventPin, "%s performs a big move and pins %s!", att.Card.Name, def.Card.Name))
		m.resolvePIN(att, def)

	case InterfBackfireFinish:
		m.emit(newEvent(EventInterference, "%s storms the ring but %s wins the brawl and throws him out!", allyName, att.Card.Name))
		m.emit(newEvent(EventFinisher, "%s motions to the crowd — %s time!", att.Card.Name, att.Card.Finisher.Name))
		pinRoll := m.rollTwo().total()
		threshold := def.CurrentPIN + att.Card.Finisher.Rating
		if pinRoll <= threshold {
			m.emit(pinEvent(def.Card.Name, pinRoll, threshold, true))
			m.endMatch(att, def, "pinfall")
		} else {
			m.emit(pinEvent(def.Card.Name, pinRoll, threshold, false))
			def.CurrentPIN++
			m.emit(newEvent(EventFatigue, "%s's PIN rating increases to %d!", def.Card.Name, def.CurrentPIN))
			m.onOffense = defIdx
			m.offLevel = 2
		}
	}
}

// ─── DISTRACTION ─────────────────────────────────────────────────────────────

// tryDistraction attempts to distract the referee before a PIN roll.
// Returns true if distraction was successful (PIN is avoided).
func (m *Match) tryDistraction(pinnedIdx int) bool {
	side := m.Sides[pinnedIdx]
	pinned := side.Active()

	if m.distractionUsed[pinnedIdx] || side.Ally == nil {
		return false
	}

	// AI decision: use distraction if PIN is moderately dangerous
	if pinned.CurrentPIN < 4 {
		return false
	}

	// Don't use distraction if interference is still available and PIN is very high
	// (save distraction for moderate danger, interference for high danger)
	if pinned.CurrentPIN >= 6 && !m.interferenceUsed[pinnedIdx] {
		return false
	}

	m.distractionUsed[pinnedIdx] = true

	allyName := side.Ally.Name
	distRating := pinned.Card.Distractor

	roll := m.rollTwo().total()
	m.emit(newEvent(EventDistraction, "%s tries to distract the referee! (roll %d, needs %d or lower)", allyName, roll, distRating))

	if roll <= distRating {
		m.emit(newEvent(EventDistraction, "The distraction works! The referee is distracted and the pin count is broken!"))
		pinned.CurrentPIN++
		m.emit(newEvent(EventFatigue, "%s's PIN rating increases to %d from fatigue.", pinned.Card.Name, pinned.CurrentPIN))
		m.onOffense = 1 - pinnedIdx
		m.offLevel = 2
		return true
	}

	m.emit(newEvent(EventDistraction, "The distraction fails! The referee orders %s to leave!", allyName))
	return false
}

// ─── RINGSIDE ALLY ───────────────────────────────────────────────────────────

// resolveRingsideAllyAttack handles when an attacker's ringside ally attacks
// the defender who has been thrown out of the ring.
func (m *Match) resolveRingsideAllyAttack(att, def *WrestlerState) {
	attSide := m.sideOf(att)
	allyName := m.Sides[attSide].Ally.Name

	m.emit(newEvent(EventInterference, "%s is attacked outside the ring by %s!", def.Card.Name, allyName))
	m.emit(newEvent(EventInterference, "%s smashes %s into the steel post!", allyName, def.Card.Name))
	m.emit(newEvent(EventDQ, "%s and %s may be disqualified!", att.Card.Name, allyName))

	// DQ threshold is always 6 regardless of wrestler's DQ rating
	if m.rollDQWithThreshold(att, 6) {
		return
	}

	m.emit(newEvent(EventInterference, "%s tosses %s back into the ring to the waiting hands of %s!", allyName, def.Card.Name, att.Card.Name))
	m.onOffense = m.sideOf(att)
	m.offLevel = 2 // Level 3
}

// ─── FEUD TABLE ──────────────────────────────────────────────────────────────

// resolveFeudTable is called after a feud match ends. Rolls for doubles,
// and if doubles come up, resolves the feud table outcome.
func (m *Match) resolveFeudTable(winner, loser *WrestlerState) {
	postMatch := m.rollTwo()
	if !postMatch.doubles() {
		m.emit(newEvent(EventMatchEnd, "Post-match: no doubles rolled (%d) — the feud simmers down... for now.", postMatch.total()))
		return
	}

	m.emit(newEvent(EventMatchEnd, ""))
	m.emit(newEvent(EventMatchEnd, "DOUBLES ROLLED (%d)! THE FEUD CONTINUES AFTER THE BELL!", postMatch.total()))

	feudRoll := m.rollTwo().total()
	outcome := LookupFeud(feudRoll)
	if outcome == nil {
		return
	}

	m.emit(newEvent(EventMatchEnd, "[Feud Table, roll %d] %s", feudRoll, outcome.Text))
	m.result.FeudText = outcome.Text

	switch outcome.Type {
	case FeudAttackedByLoser:
		// Winner is injured by the loser's post-match attack
		m.result.InjuredWrestler = winner.Card.Name
		m.result.InjuryCards = outcome.InjuryDays
	case FeudAllyDoubleTeam:
		// No injury — ally challenges for next match
		m.emit(newEvent(EventMatchEnd, "A new rivalry is born!"))
	case FeudPostMatchAttack:
		// Loser is injured
		m.result.InjuredWrestler = loser.Card.Name
		m.result.InjuryCards = outcome.InjuryDays
	case FeudFourManBrawl:
		// Wild brawl — no direct injury, leads to tag match booking
		m.emit(newEvent(EventMatchEnd, "The commissioner books a tag team super match!"))
	case FeudOpponentAlly:
		// Winner is injured by opponent's ally
		m.result.InjuredWrestler = winner.Card.Name
		m.result.InjuryCards = outcome.InjuryDays
	case FeudGangAttack:
		// Roll 1d6 for injury duration
		injuryRoll := m.rollOne()
		suspensionRoll := m.rollOne()
		m.emit(newEvent(EventMatchEnd, "Injury roll: %d fight cards! Suspension roll: %d fight cards!", injuryRoll, suspensionRoll))
		m.result.InjuredWrestler = winner.Card.Name
		m.result.InjuryCards = injuryRoll
	}

	if m.result.InjuredWrestler != "" {
		m.emit(newEvent(EventMatchEnd, "%s IS INJURED FOR %d FIGHT CARD(S)!", m.result.InjuredWrestler, m.result.InjuryCards))
	}
}
