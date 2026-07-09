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
	CurrentPIN int  // Starts at card's PINAdv, increases with fatigue
	Injured    bool
	InjuryLeft int  // Fight cards remaining on injury
}

// Side represents one corner of the match (one or more wrestlers in tag).
type Side struct {
	Wrestlers    []*WrestlerState
	ActiveIndex  int // Which wrestler is currently in the ring
	PinSavesUsed int // Max 2 per match in tag matches
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
}

// NewMatch creates a match between two wrestlers (singles).
func NewMatch(card1, card2 *WrestlerCard) *Match {
	m := &Match{
		Type: MatchSingles,
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
func (m *Match) Result() *MatchResult      { return m.result }

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
	r1 := Roll1d6()
	r2 := Roll1d6()
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

func (m *Match) executeTurn() {
	// Track ref recovery
	if m.refDown {
		m.refDownTurns--
		if m.refDownTurns <= 0 {
			m.refDown = false
			m.emit(newEvent(EventRefRecover, "The referee recovers and is back on his feet!"))
		}
	}

	// Tag match: attacker may tag out to partner on offense
	if m.Type == MatchTag {
		m.maybeTagOnOffense()
	}

	att := m.attacker()
	def := m.defender()

	// Step 1: Attacker rolls 1d6 on current offense level
	offRoll := Roll1d6()
	move := att.Card.Offense[m.offLevel][offRoll-1]

	m.emit(Event{
		Type:     EventMove,
		Text:     fmt.Sprintf("%s (Level %d, roll %d): %s!", att.Card.Name, m.offLevel+1, offRoll, move.Name),
		Attacker: att.Card.Name,
		Defender: def.Card.Name,
		Roll:     offRoll,
		Level:    m.offLevel + 1,
	})

	// Handle special move tags before normal resolution
	m.resolveMove(att, def, move)
}

func (m *Match) resolveMove(att, def *WrestlerState, move Move) {
	// Check agility/power requirements
	if move.HasTag(TagAgility) {
		if att.Card.Agility < def.Card.Agility {
			m.emit(newEvent(EventDefense, "%s's agility isn't good enough — %s counters!", att.Card.Name, def.Card.Name))
			m.switchOffense()
			m.offLevel = 1 // Level 2 defense -> offense
			return
		}
		m.emit(newEvent(EventMove, "%s has the agility advantage — the move connects!", att.Card.Name))
	}

	if move.HasTag(TagPower) {
		if att.Card.Power < def.Card.Power {
			m.emit(newEvent(EventDefense, "%s isn't powerful enough — %s overpowers and counters!", att.Card.Name, def.Card.Name))
			m.switchOffense()
			m.offLevel = 1
			return
		}
		m.emit(newEvent(EventMove, "%s has the power advantage — the move connects!", att.Card.Name))
	}

	// Check for DQ move
	if move.HasTag(TagDQ) {
		m.emit(newEvent(EventDQ, "%s goes for a dirty move — the referee is watching!", att.Card.Name))
		if m.rollDQ(att) {
			return // Match ended by DQ
		}
	}

	// Check for Add1 (automatically adds fatigue to opponent)
	if move.HasTag(TagAdd1) {
		def.CurrentPIN++
		m.emit(newEvent(EventFatigue, "Devastating move! %s's PIN rating increases to %d!", def.Card.Name, def.CurrentPIN))
	}

	// Check for chart moves
	if move.HasTag(TagChart) {
		chartType := move.ChartType
		// Cage match: "out of the ring" becomes "face into cage"
		if m.Type == MatchCage && chartType == "ring" {
			m.emit(newEvent(EventMove, "%s smashes %s face-first into the cage! - 3", att.Card.Name, def.Card.Name))
			m.offLevel = 2
			return
		}
		m.resolveChart(att, def, chartType)
		return
	}

	// Check for choice situation
	if move.HasTag(TagChoice) {
		m.resolveChoice(att, def, move.ChoiceKey)
		return
	}

	// Check if the move is a finisher (ALL CAPS name)
	if move.IsFinisher() {
		m.resolveFinisher(att, def)
		return
	}

	// Normal move — defender rolls on defense
	m.resolveNormalDefense(att, def, move)
}

func (m *Match) resolveNormalDefense(att, def *WrestlerState, move Move) {
	defLevel := move.DefLevel - 1
	if defLevel < 0 {
		defLevel = 0
	}
	if defLevel > 2 {
		defLevel = 2
	}

	defRoll := Roll1d6()
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

func (m *Match) resolveDefense(outcome DefenseOutcome) {
	def := m.defender()

	switch outcome.Type {
	case DefDazed:
		m.emit(defenseEvent(def.Card.Name, DefDazed))
		m.advanceOffLevel(outcome.Power)

	case DefHurt:
		m.emit(defenseEvent(def.Card.Name, DefHurt))
		m.advanceOffLevel(outcome.Power)

	case DefDown:
		m.emit(defenseEvent(def.Card.Name, DefDown))

		// Check for interference on down-3
		if outcome.Power == 3 {
			defIdx := 1 - m.onOffense
			if m.shouldUseInterference(defIdx) {
				m.emit(newEvent(EventInterference, "%s calls for outside interference!", def.Card.Name))
				m.resolveInterference(defIdx)
				return
			}
		}

		m.advanceOffLevel(outcome.Power)

		// Check for leaving the ring option
		if outcome.HasTag(TagLeave) && outcome.Power == 3 {
			m.emit(newEvent(EventChart, "%s has the option to leave the ring!", def.Card.Name))
			// AI decision: leave if ring rating is A or B
			if def.Card.Ring <= RatingB {
				m.emit(newEvent(EventChart, "%s rolls out of the ring!", def.Card.Name))
				m.resolveChart(def, m.attacker(), "ring")
				return
			}
		}

	case DefReversal:
		m.emit(reversalEvent(def.Card.Name))
		m.switchOffense()
		if outcome.Power >= 1 && outcome.Power <= 3 {
			m.offLevel = outcome.Power - 1
		} else {
			m.offLevel = 0
		}

	case DefPIN:
		att := m.attacker()
		defIdx := 1 - m.onOffense

		// Check for outside interference (higher priority, used when very desperate)
		if m.shouldUseInterference(defIdx) {
			m.emit(newEvent(EventInterference, "%s calls for outside interference!", def.Card.Name))
			m.resolveInterference(defIdx)
			return
		}

		m.emit(newEvent(EventPin, "%s is in a pinning predicament!", def.Card.Name))

		// Check for distraction before PIN roll
		if m.tryDistraction(defIdx) {
			return
		}

		m.resolvePIN(att, def)
	}
}

// advanceOffLevel moves the offense level up based on move power.
func (m *Match) advanceOffLevel(power int) {
	switch {
	case power >= 3:
		m.offLevel = 2 // Level 3
	case power == 2:
		if m.offLevel < 1 {
			m.offLevel = 1 // At least Level 2
		}
	default:
		if m.offLevel < 1 {
			m.offLevel = 1
		}
	}
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

	roll := Roll2d6()

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

func (m *Match) resolveChartOutcome(att, def *WrestlerState, outcome *ChartOutcome, chartType string) {
	switch outcome.Type {
	case ChartRollOnOffense:
		// The chart-roller (defender) gets offense at the specified level
		m.onOffense = m.sideOf(def)
		m.offLevel = outcome.Level - 1

	case ChartOppRollOnOffense:
		// The attacker (who threw them into chart) gets offense
		m.onOffense = m.sideOf(att)
		m.offLevel = outcome.Level - 1

	case ChartRollOnDefense:
		// Opponent rolls on defense at specified level
		defRoll := Roll1d6()
		defOutcome := att.Card.Defense[outcome.Level-1][defRoll-1]
		m.emit(Event{
			Type:     EventDefense,
			Text:     fmt.Sprintf("%s (Defense Level %d, roll %d): %s", att.Card.Name, outcome.Level, defRoll, defOutcome.Type),
			Defender: att.Card.Name,
			Roll:     defRoll,
			Level:    outcome.Level,
		})
		// Temporarily swap perspective for defense resolution
		origOff := m.onOffense
		m.onOffense = m.sideOf(def)
		m.resolveDefense(defOutcome)
		if !m.over && m.onOffense == m.sideOf(def) {
			// If defense didn't cause a reversal, restore
			_ = origOff
		}

	case ChartRollPIN:
		// Opponent of chart-roller is pinned (att threw def into chart, def countered and pins att)
		m.emit(newEvent(EventPin, "%s is in a pinning predicament!", att.Card.Name))
		m.resolvePIN(def, att)

	case ChartRollYourPIN:
		// Chart-roller (defender) is pinned
		m.emit(newEvent(EventPin, "%s is in a pinning predicament!", def.Card.Name))
		m.resolvePIN(att, def)

	case ChartRollAgain:
		m.resolveChart(att, def, chartType)

	case ChartRollDQ:
		// Current chart roller might get DQ'd
		if m.rollDQ(def) {
			return
		}
		// If no DQ, follow up
		if outcome.ThenType == ChartRollOnOffense {
			m.onOffense = m.sideOf(def)
			m.offLevel = outcome.ThenLevel - 1
		}

	case ChartOppRollDQ:
		// Attacker (the one who threw them) might get DQ'd
		if m.rollDQ(att) {
			return
		}
		if outcome.ThenType == ChartOppRollOnOffense {
			m.onOffense = m.sideOf(att)
			m.offLevel = outcome.ThenLevel - 1
		}

	case ChartBothRollDQ:
		// Both wrestlers roll DQ
		m.emit(newEvent(EventDQ, "Both wrestlers may be disqualified!"))
		if m.rollDQ(att) {
			return
		}
		if m.rollDQ(def) {
			return
		}
		// Neither DQ'd — roll 1d6: even = def wins brawl, odd = att wins
		brawlRoll := Roll1d6()
		if brawlRoll%2 == 0 {
			m.emit(newEvent(EventChart, "%s wins the brawl! (roll %d)", def.Card.Name, brawlRoll))
			m.onOffense = m.sideOf(def)
		} else {
			m.emit(newEvent(EventChart, "%s wins the brawl! (roll %d)", att.Card.Name, brawlRoll))
			m.onOffense = m.sideOf(att)
		}
		m.offLevel = 2 // Level 3

	case ChartPowerCheck:
		if def.Card.Power > att.Card.Power {
			m.emit(newEvent(EventChart, "%s is more powerful and knocks the opponent down with a shoulder tackle!", def.Card.Name))
			m.onOffense = m.sideOf(def)
			m.offLevel = 1 // Level 2
		} else {
			m.emit(newEvent(EventChart, "%s is overpowered! The opponent knocks him down with a shoulder tackle!", def.Card.Name))
			m.onOffense = m.sideOf(att)
			m.offLevel = 1
		}

	case ChartAgilityCheck:
		// Deathjump: if defender has better agility, they win the struggle
		if def.Card.Agility > att.Card.Agility {
			m.emit(newEvent(EventChart, "%s wins the struggle on the top rope with superior agility!", def.Card.Name))
			m.onOffense = m.sideOf(def)
		} else {
			m.emit(newEvent(EventChart, "%s pushes %s off the top rope!", att.Card.Name, def.Card.Name))
			m.onOffense = m.sideOf(att)
		}
		m.offLevel = 2 // Level 3

	case ChartBetterRating:
		defRating := m.getRating(def, outcome.RatingType)
		attRating := m.getRating(att, outcome.RatingType)
		if defRating < attRating { // Lower Rating value = better (A=0, B=1, C=2)
			m.emit(newEvent(EventChart, "%s has the better %s rating and recovers first!", def.Card.Name, outcome.RatingType))
			m.onOffense = m.sideOf(def)
		} else {
			m.emit(newEvent(EventChart, "%s recovers first!", att.Card.Name))
			m.onOffense = m.sideOf(att)
		}
		m.offLevel = 2

	case ChartRefDown:
		refTurns := Roll2d6()
		m.refDown = true
		m.refDownTurns = refTurns
		m.emit(newEvent(EventRefDown, "THE REFEREE IS DOWN! He'll be out for %d moves!", refTurns))
		// Roll 1d6: even = defender recovers, odd = attacker continues
		whoRoll := Roll1d6()
		if whoRoll%2 == 0 {
			m.emit(newEvent(EventChart, "%s takes advantage of the chaos! (roll %d)", def.Card.Name, whoRoll))
			m.onOffense = m.sideOf(def)
		} else {
			m.emit(newEvent(EventChart, "%s is still down — %s goes for the kill! (roll %d)", def.Card.Name, att.Card.Name, whoRoll))
			m.onOffense = m.sideOf(att)
		}
		m.offLevel = 2

	case ChartOppRollOnChart:
		// Redirect to another chart (e.g., turnbuckle reversal -> opponent on turnbuckle chart)
		m.resolveChart(def, att, outcome.ChartRef)

	case ChartRollCountOut:
		m.emit(newEvent(EventCountOut, "%s may be counted out!", def.Card.Name))
		m.resolveCountOut(att, def)
		if outcome.AddFatigue && !m.over {
			def.CurrentPIN++
			m.emit(newEvent(EventFatigue, "%s's PIN rating increases to %d!", def.Card.Name, def.CurrentPIN))
		}
		if !m.over {
			m.onOffense = m.sideOf(att)
			m.offLevel = 2
		}
	}
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

// ─── CHOICE SITUATIONS ──────────────────────────────────────────────────────

func (m *Match) resolveChoice(att, def *WrestlerState, choiceKey string) {
	choice, ok := ChoiceSituations[choiceKey]
	if !ok {
		m.emit(newEvent(EventChart, "Unknown choice situation: %s — continuing normally.", choiceKey))
		return
	}

	m.emit(newEvent(EventChart, "CHOICE SITUATION %s! %s must decide between %s or %s!",
		choiceKey, att.Card.Name, choice.Option1.Name, choice.Option2.Name))

	// AI picks the option with the better chance of success
	opt := m.pickChoiceOption(att, def, choice)

	if opt.IsChart {
		m.emit(newEvent(EventChart, "%s chooses: %s!", att.Card.Name, opt.Name))
		m.resolveChart(att, def, opt.ChartRef)
		return
	}

	// Roll-based choice move
	roll := Roll2d6()
	statMod := 0
	if opt.StatType == "ag" {
		statMod = def.Card.Agility // Plus or minus opponent's agility
	} else if opt.StatType == "pw" {
		statMod = def.Card.Power
	}

	adjustedThreshold := opt.Threshold - statMod // Opponent's stat reduces the threshold
	m.emit(newEvent(EventChart, "%s tries a %s! (roll %d, needs %d or lower, adjusted for opponent's %s)",
		att.Card.Name, opt.Name, roll, adjustedThreshold, opt.StatType))

	if roll <= adjustedThreshold {
		m.emit(newEvent(EventMove, "%s hits the %s!", att.Card.Name, opt.Name))
		m.advanceOffLevel(opt.Power)
	} else {
		m.emit(newEvent(EventDefense, "The %s fails! %s takes over!", opt.Name, def.Card.Name))
		m.switchOffense()
		m.offLevel = 1 // Opponent rolls Level 2 offense on failure
	}
}

func (m *Match) pickChoiceOption(att, def *WrestlerState, choice ChoiceSituation) ChoiceOption {
	// Chart options are always viable
	if choice.Option1.IsChart && !choice.Option2.IsChart {
		// Compare: chart (unpredictable) vs roll check
		// Pick whichever is more likely to succeed; for simplicity, prefer the roll if threshold is high
		score2 := choice.Option2.Threshold
		if choice.Option2.StatType == "ag" {
			score2 -= def.Card.Agility
		} else {
			score2 -= def.Card.Power
		}
		if score2 >= 8 {
			return choice.Option2
		}
		return choice.Option1
	}
	if choice.Option2.IsChart && !choice.Option1.IsChart {
		score1 := choice.Option1.Threshold
		if choice.Option1.StatType == "ag" {
			score1 -= def.Card.Agility
		} else {
			score1 -= def.Card.Power
		}
		if score1 >= 8 {
			return choice.Option1
		}
		return choice.Option2
	}

	// Both are roll checks — pick the one with a better effective threshold
	score1 := choice.Option1.Threshold
	if choice.Option1.StatType == "ag" {
		score1 -= def.Card.Agility
	} else {
		score1 -= def.Card.Power
	}
	score2 := choice.Option2.Threshold
	if choice.Option2.StatType == "ag" {
		score2 -= def.Card.Agility
	} else {
		score2 -= def.Card.Power
	}

	// Prefer the higher-power move if thresholds are close
	if score1 >= score2 {
		return choice.Option1
	}
	return choice.Option2
}

// ─── PIN / FINISHER / DQ / COUNTOUT ─────────────────────────────────────────

// resolvePIN handles a pin attempt. Defender rolls 2d6 vs their current PIN rating.
func (m *Match) resolvePIN(pinner, pinned *WrestlerState) {
	// If ref is down, PIN can't be counted
	if m.refDown {
		m.emit(newEvent(EventPin, "PIN ATTEMPT — but the referee is still down! No count!"))
		pinned.CurrentPIN++
		m.emit(newEvent(EventFatigue, "%s's PIN rating increases to %d from fatigue.", pinned.Card.Name, pinned.CurrentPIN))
		m.onOffense = m.sideOf(pinner)
		m.offLevel = 2
		return
	}

	roll := Roll2d6()
	threshold := pinned.CurrentPIN

	if roll <= threshold {
		m.emit(pinEvent(pinned.Card.Name, roll, threshold, true))
		// In tag matches, partner can try a pin save
		if m.Type == MatchTag {
			pinnedSide := m.sideOf(pinned)
			if m.tryPinSave(pinnedSide) {
				return // Save was successful (or ended the match via double DQ)
			}
		}
		if !m.over {
			m.endMatch(pinner, pinned, "pinfall")
		}
	} else {
		m.emit(pinEvent(pinned.Card.Name, roll, threshold, false))
		pinned.CurrentPIN++
		m.emit(Event{
			Type: EventFatigue,
			Text: fmt.Sprintf("%s's PIN rating increases to %d from fatigue.", pinned.Card.Name, pinned.CurrentPIN),
		})
		// After kick-out, pinned wrestler gets offense at Level 3
		m.onOffense = m.sideOf(pinned)
		m.offLevel = 2
	}
}

// resolveFinisher handles when a finisher move is rolled.
func (m *Match) resolveFinisher(att, def *WrestlerState) {
	finisher := att.Card.Finisher
	m.emit(finisherEvent(att.Card.Name, finisher.Name))

	// Handle roll finishers
	if finisher.IsRoll {
		fRoll := Roll1d6()
		m.emit(newEvent(EventFinisher, "%s rolls for the finisher: %d! (needs %d-%d)",
			att.Card.Name, fRoll, finisher.RollMin, finisher.RollMax))
		if fRoll < finisher.RollMin || fRoll > finisher.RollMax {
			m.emit(newEvent(EventFinisher, "The %s misses! %s dodges and takes over!", finisher.Name, def.Card.Name))
			m.switchOffense()
			m.offLevel = 1
			return
		}
		m.emit(newEvent(EventFinisher, "The %s connects!", finisher.Name))
	}

	// Defender rolls 2d6 against their PIN + finisher rating
	roll := Roll2d6()
	threshold := def.CurrentPIN + finisher.Rating

	if m.refDown {
		m.emit(newEvent(EventPin, "%s hits the %s — but the referee is still down! No count!", att.Card.Name, finisher.Name))
		def.CurrentPIN++
		m.emit(newEvent(EventFatigue, "%s's PIN rating increases to %d from fatigue.", def.Card.Name, def.CurrentPIN))
		m.onOffense = m.sideOf(att)
		m.offLevel = 2
		return
	}

	if roll <= threshold {
		m.emit(pinEvent(def.Card.Name, roll, threshold, true))
		m.endMatch(att, def, "pinfall")
	} else {
		m.emit(pinEvent(def.Card.Name, roll, threshold, false))
		def.CurrentPIN++
		m.emit(Event{
			Type: EventFatigue,
			Text: fmt.Sprintf("%s's PIN rating increases to %d from fatigue.", def.Card.Name, def.CurrentPIN),
		})
		m.onOffense = m.sideOf(def)
		m.offLevel = 2
	}
}

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

	roll := Roll2d6()
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
	roll := Roll2d6()
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

	roll := Roll2d6()
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

	roll := Roll2d6()
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
	roll := Roll2d6()
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
		pinRoll := Roll2d6()
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
		dqRoll := Roll2d6()
		if dqRoll <= 4 {
			m.emit(newEvent(EventDQ, "DOUBLE DISQUALIFICATION! Both teams are thrown out!"))
			m.over = true
			m.result = &MatchResult{Method: "double dq"}
			m.emit(newEvent(EventMatchEnd, "The match ends in a DOUBLE DISQUALIFICATION!"))
			return true
		}
		brawlRoll := Roll1d6()
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

	roll := Roll2d6()
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
		pinRoll := Roll2d6()
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
		brawlRoll := Roll1d6()
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
		pinRoll := Roll2d6()
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

	roll := Roll2d6()
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
	total, doubles := RollIsDoubles()
	if !doubles {
		m.emit(newEvent(EventMatchEnd, "Post-match: no doubles rolled (%d) — the feud simmers down... for now.", total))
		return
	}

	m.emit(newEvent(EventMatchEnd, ""))
	m.emit(newEvent(EventMatchEnd, "DOUBLES ROLLED (%d)! THE FEUD CONTINUES AFTER THE BELL!", total))

	feudRoll := Roll2d6()
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
		injuryRoll := Roll1d6()
		suspensionRoll := Roll1d6()
		m.emit(newEvent(EventMatchEnd, "Injury roll: %d fight cards! Suspension roll: %d fight cards!", injuryRoll, suspensionRoll))
		m.result.InjuredWrestler = winner.Card.Name
		m.result.InjuryCards = injuryRoll
	}

	if m.result.InjuredWrestler != "" {
		m.emit(newEvent(EventMatchEnd, "%s IS INJURED FOR %d FIGHT CARD(S)!", m.result.InjuredWrestler, m.result.InjuryCards))
	}
}
