package engine

// ChartOutcomeType describes what a chart result tells the engine to do.
type ChartOutcomeType int

const (
	ChartRollOnOffense    ChartOutcomeType = iota // "ROLL ON LEVEL X OFFENSE"
	ChartOppRollOnOffense                         // "OPPONENT ROLLS ON LEVEL X OFFENSE"
	ChartRollOnDefense                            // "OPPONENT ROLLS ON LEVEL X DEFENSE"
	ChartRollPIN                                  // "OPPONENT ROLLS PIN" (opponent of the chart-roller is pinned)
	ChartRollYourPIN                              // "ROLL YOUR PIN" (the chart-roller is pinned)
	ChartRollAgain                                // "ROLL AGAIN ON THIS CHART"
	ChartRollDQ                                   // "ROLL DISQUALIFICATION"
	ChartBothRollDQ                               // "BOTH WRESTLERS ROLL DISQUALIFICATION"
	ChartOppRollDQ                                // "OPPONENT ROLLS DISQUALIFICATION"
	ChartRollCountOut                             // "ROLL COUNT OUT"
	ChartPowerCheck                               // Outcome depends on power comparison
	ChartBetterRating                             // Outcome depends on who has better rating for this chart
	ChartAgilityCheck                             // Outcome depends on agility comparison
	ChartRefDown                                  // Referee gets knocked out
	ChartOppRollOnChart                           // "OPPONENT ROLLS ON [some other] CHART"
)

// ChartOutcome is a single result row from a chart.
type ChartOutcome struct {
	MinRoll int
	MaxRoll int
	Type    ChartOutcomeType
	Level   int    // Offense/defense level (1-3) if applicable
	Text    string // Commentary flavor text

	// For compound outcomes (e.g. "ROLL DQ, then if no DQ, ROLL ON LEVEL 3")
	DQThreshold  int              // If > 0, roll DQ against this threshold first
	ThenType     ChartOutcomeType // What happens after DQ passes
	ThenLevel    int
	AddFatigue   bool   // Add 1 to PIN rating
	ChartRef     string // For ChartOppRollOnChart, which chart
	RatingType   string // "ropes", "turnbuckle", "ring" — for ChartBetterRating
}

// ChartTable maps a Rating to a list of outcomes.
type ChartTable map[Rating][]ChartOutcome

// Lookup finds the outcome for a given rating and roll.
func (ct ChartTable) Lookup(rating Rating, roll int) *ChartOutcome {
	outcomes := ct[rating]
	for i := range outcomes {
		if roll >= outcomes[i].MinRoll && roll <= outcomes[i].MaxRoll {
			return &outcomes[i]
		}
	}
	return nil
}

// ─── INTO THE ROPES ─────────────────────────────────────────────────────────

var RopesChart = ChartTable{
	RatingA: {
		{MinRoll: 2, MaxRoll: 3, Type: ChartRollPIN,
			Text: "The opponent tries a sunset flip but you grab his legs and roll him into a pinning combination!"},
		{MinRoll: 4, MaxRoll: 5, Type: ChartRollOnOffense, Level: 3,
			Text: "The opponent gets into position for a back body drop but you grab him and hit him with an awesome piledriver!"},
		{MinRoll: 6, MaxRoll: 6, Type: ChartRollOnOffense, Level: 2,
			Text: "You come off the ropes with a powerful flying shoulder tackle!"},
		{MinRoll: 7, MaxRoll: 7, Type: ChartRollAgain,
			Text: "The opponent drops down, you go into the ropes again."},
		{MinRoll: 8, MaxRoll: 9, Type: ChartOppRollOnOffense, Level: 2,
			Text: "You come off the ropes and the opponent puts you down hard with a running back elbow!"},
		{MinRoll: 10, MaxRoll: 10, Type: ChartPowerCheck,
			Text: "The opponent tries a big shoulder tackle, but whether it works depends on which wrestler is more powerful!"},
		{MinRoll: 11, MaxRoll: 11, Type: ChartBetterRating, RatingType: "ropes",
			Text: "The opponent tries a running clothesline but so do you! Double clothesline and both go down!"},
		{MinRoll: 12, MaxRoll: 12, Type: ChartOppRollOnOffense, Level: 3,
			Text: "The opponent uses one of his specialty moves on you and goes in for the kill!"},
	},
	RatingB: {
		{MinRoll: 2, MaxRoll: 2, Type: ChartRollPIN,
			Text: "The opponent tries a sunset flip but you grab his legs and roll him into a pinning combination!"},
		{MinRoll: 3, MaxRoll: 4, Type: ChartRollOnOffense, Level: 3,
			Text: "The opponent gets into position for a back body drop but you grab him and hit him with an awesome piledriver!"},
		{MinRoll: 5, MaxRoll: 5, Type: ChartRollOnOffense, Level: 2,
			Text: "You come off the ropes with a powerful flying shoulder tackle!"},
		{MinRoll: 6, MaxRoll: 6, Type: ChartRollAgain,
			Text: "The opponent drops down, you go into the ropes again."},
		{MinRoll: 7, MaxRoll: 8, Type: ChartOppRollOnOffense, Level: 2,
			Text: "You come off the ropes and the opponent puts you down hard with a running back elbow!"},
		{MinRoll: 9, MaxRoll: 9, Type: ChartPowerCheck,
			Text: "The opponent tries a big shoulder tackle, but whether it works depends on which wrestler is more powerful!"},
		{MinRoll: 10, MaxRoll: 10, Type: ChartBetterRating, RatingType: "ropes",
			Text: "The opponent tries a running clothesline but so do you! Double clothesline and both go down!"},
		{MinRoll: 11, MaxRoll: 12, Type: ChartOppRollOnOffense, Level: 3,
			Text: "The opponent uses one of his specialty moves on you and goes in for the kill!"},
	},
	RatingC: {
		{MinRoll: 2, MaxRoll: 3, Type: ChartRollOnOffense, Level: 3,
			Text: "The opponent gets into position for a back body drop but you grab him and hit him with an awesome piledriver!"},
		{MinRoll: 4, MaxRoll: 4, Type: ChartRollOnOffense, Level: 2,
			Text: "You come off the ropes with a powerful flying shoulder tackle!"},
		{MinRoll: 5, MaxRoll: 5, Type: ChartRollAgain,
			Text: "The opponent drops down, you go into the ropes again."},
		{MinRoll: 6, MaxRoll: 7, Type: ChartOppRollOnOffense, Level: 2,
			Text: "You come off the ropes and the opponent puts you down hard with a running back elbow!"},
		{MinRoll: 8, MaxRoll: 8, Type: ChartPowerCheck,
			Text: "The opponent tries a big shoulder tackle, but whether it works depends on which wrestler is more powerful!"},
		{MinRoll: 9, MaxRoll: 9, Type: ChartBetterRating, RatingType: "ropes",
			Text: "The opponent tries a running clothesline but so do you! Double clothesline and both go down!"},
		{MinRoll: 10, MaxRoll: 12, Type: ChartOppRollOnOffense, Level: 3,
			Text: "The opponent uses one of his specialty moves on you and goes in for the kill!"},
	},
}

// ─── INTO THE TURNBUCKLE ────────────────────────────────────────────────────

var TurnbuckleChart = ChartTable{
	RatingA: {
		{MinRoll: 2, MaxRoll: 3, Type: ChartRollPIN,
			Text: "The opponent tries a running clothesline, but you move and he crashes into the turnbuckle! You cover him for the pin!"},
		{MinRoll: 4, MaxRoll: 4, Type: ChartOppRollOnChart, ChartRef: "ring",
			Text: "The opponent charges you with a running shoulder dive but you move and he goes crashing outside of the ring!"},
		{MinRoll: 5, MaxRoll: 5, Type: ChartRollOnDefense, Level: 3,
			Text: "You bounce forward off the turnbuckle and catch the charging opponent with a skull-splitting running lariat! He goes down hard!"},
		{MinRoll: 6, MaxRoll: 6, Type: ChartRollOnOffense, Level: 2,
			Text: "You lift a knee to the oncoming opponent's head! He is hurt!"},
		{MinRoll: 7, MaxRoll: 7, Type: ChartOppRollOnChart, ChartRef: "turnbuckle",
			Text: "REVERSAL! You reverse the move and throw the opponent into the turnbuckle!"},
		{MinRoll: 8, MaxRoll: 10, Type: ChartOppRollOnOffense, Level: 2,
			Text: "You are crushed by a big kick by the opponent!"},
		{MinRoll: 11, MaxRoll: 11, Type: ChartBetterRating, RatingType: "turnbuckle",
			Text: "You come off the turnbuckle with a big specialty move, but the opponent tries a specialty move of his own! Both wrestlers go down!"},
		{MinRoll: 12, MaxRoll: 12, Type: ChartOppRollOnOffense, Level: 3,
			Text: "You bounce forward off the turnbuckle and into an incredible neck-breaking clothesline!"},
	},
	RatingB: {
		{MinRoll: 2, MaxRoll: 2, Type: ChartRollPIN,
			Text: "The opponent tries a running clothesline, but you move and he crashes into the turnbuckle! You cover him for the pin!"},
		{MinRoll: 3, MaxRoll: 3, Type: ChartOppRollOnChart, ChartRef: "ring",
			Text: "The opponent charges you with a running shoulder dive but you move and he goes crashing outside of the ring!"},
		{MinRoll: 4, MaxRoll: 4, Type: ChartRollOnDefense, Level: 3,
			Text: "You bounce forward off the turnbuckle and catch the charging opponent with a skull-splitting running lariat! He goes down hard!"},
		{MinRoll: 5, MaxRoll: 5, Type: ChartRollOnOffense, Level: 2,
			Text: "You lift a knee to the oncoming opponent's head! He is hurt!"},
		{MinRoll: 6, MaxRoll: 6, Type: ChartOppRollOnChart, ChartRef: "turnbuckle",
			Text: "REVERSAL! You reverse the move and throw the opponent into the turnbuckle!"},
		{MinRoll: 7, MaxRoll: 9, Type: ChartOppRollOnOffense, Level: 2,
			Text: "You are crushed by a big kick by the opponent!"},
		{MinRoll: 10, MaxRoll: 10, Type: ChartBetterRating, RatingType: "turnbuckle",
			Text: "You come off the turnbuckle with a big specialty move, but the opponent tries a specialty move of his own! Both wrestlers go down!"},
		{MinRoll: 11, MaxRoll: 12, Type: ChartOppRollOnOffense, Level: 3,
			Text: "You bounce forward off the turnbuckle and into an incredible neck-breaking clothesline!"},
	},
	RatingC: {
		{MinRoll: 2, MaxRoll: 2, Type: ChartOppRollOnChart, ChartRef: "ring",
			Text: "The opponent charges you with a running shoulder dive but you move and he goes crashing outside of the ring!"},
		{MinRoll: 3, MaxRoll: 3, Type: ChartRollOnDefense, Level: 3,
			Text: "You bounce forward off the turnbuckle and catch the charging opponent with a skull-splitting running lariat! He goes down hard!"},
		{MinRoll: 4, MaxRoll: 4, Type: ChartRollOnOffense, Level: 2,
			Text: "You lift a knee to the oncoming opponent's head! He is hurt!"},
		{MinRoll: 5, MaxRoll: 5, Type: ChartOppRollOnChart, ChartRef: "turnbuckle",
			Text: "REVERSAL! You reverse the move and throw the opponent into the turnbuckle!"},
		{MinRoll: 6, MaxRoll: 8, Type: ChartOppRollOnOffense, Level: 2,
			Text: "You are crushed by a big kick by the opponent!"},
		{MinRoll: 9, MaxRoll: 9, Type: ChartBetterRating, RatingType: "turnbuckle",
			Text: "You come off the turnbuckle with a big specialty move, but the opponent tries a specialty move of his own! Both wrestlers go down!"},
		{MinRoll: 10, MaxRoll: 12, Type: ChartOppRollOnOffense, Level: 3,
			Text: "You bounce forward off the turnbuckle and into an incredible neck-breaking clothesline!"},
	},
}

// ─── OUT OF THE RING ────────────────────────────────────────────────────────

var OutOfRingChart = ChartTable{
	RatingA: {
		{MinRoll: 2, MaxRoll: 4, Type: ChartRollOnOffense, Level: 3,
			Text: "You grab the opponent by the leg, drag him out of the ring, and smash him into the turnbuckle post!"},
		{MinRoll: 5, MaxRoll: 5, Type: ChartBothRollDQ, DQThreshold: 0,
			Text: "The opponent comes out of the ring and a wild brawl erupts!"},
		{MinRoll: 6, MaxRoll: 6, Type: ChartRollDQ,
			Text: "The opponent comes out of the ring but you grab him and smash him onto the announcer's table!",
			ThenType: ChartRollOnOffense, ThenLevel: 3},
		{MinRoll: 7, MaxRoll: 7, Type: ChartBetterRating, RatingType: "ring",
			Text: "The opponent comes out of the ring and a wild brawl erupts."},
		{MinRoll: 8, MaxRoll: 9, Type: ChartOppRollOnOffense, Level: 3,
			Text: "In order to meet the referee's count you crawl helplessly back into the ring."},
		{MinRoll: 10, MaxRoll: 11, Type: ChartOppRollDQ,
			Text: "The opponent comes out and tries to hit you with a steel chair! The referee warns him!",
			ThenType: ChartOppRollOnOffense, ThenLevel: 3},
		{MinRoll: 12, MaxRoll: 12, Type: ChartRollCountOut, AddFatigue: true,
			Text: "The opponent crushes you with a spectacular move outside the ring!"},
	},
	RatingB: {
		{MinRoll: 2, MaxRoll: 3, Type: ChartRollOnOffense, Level: 3,
			Text: "You grab the opponent by the leg, drag him out of the ring, and smash him into the turnbuckle post!"},
		{MinRoll: 4, MaxRoll: 4, Type: ChartBothRollDQ, DQThreshold: 0,
			Text: "The opponent comes out of the ring and a wild brawl erupts!"},
		{MinRoll: 5, MaxRoll: 5, Type: ChartRollDQ,
			Text: "The opponent comes out of the ring but you grab him and smash him onto the announcer's table!",
			ThenType: ChartRollOnOffense, ThenLevel: 3},
		{MinRoll: 6, MaxRoll: 6, Type: ChartBetterRating, RatingType: "ring",
			Text: "The opponent comes out of the ring and a wild brawl erupts."},
		{MinRoll: 7, MaxRoll: 9, Type: ChartOppRollOnOffense, Level: 3,
			Text: "In order to meet the referee's count you crawl helplessly back into the ring."},
		{MinRoll: 10, MaxRoll: 10, Type: ChartOppRollDQ,
			Text: "The opponent comes out and tries to hit you with a steel chair! The referee warns him!",
			ThenType: ChartOppRollOnOffense, ThenLevel: 3},
		{MinRoll: 11, MaxRoll: 12, Type: ChartRollCountOut, AddFatigue: true,
			Text: "The opponent crushes you with a spectacular move outside the ring!"},
	},
	RatingC: {
		{MinRoll: 2, MaxRoll: 2, Type: ChartRollOnOffense, Level: 3,
			Text: "You grab the opponent by the leg, drag him out of the ring, and smash him into the turnbuckle post!"},
		{MinRoll: 3, MaxRoll: 3, Type: ChartBothRollDQ, DQThreshold: 0,
			Text: "The opponent comes out of the ring and a wild brawl erupts!"},
		{MinRoll: 4, MaxRoll: 4, Type: ChartRollDQ,
			Text: "The opponent comes out of the ring but you grab him and smash him onto the announcer's table!",
			ThenType: ChartRollOnOffense, ThenLevel: 3},
		{MinRoll: 5, MaxRoll: 5, Type: ChartBetterRating, RatingType: "ring",
			Text: "The opponent comes out of the ring and a wild brawl erupts."},
		{MinRoll: 6, MaxRoll: 9, Type: ChartOppRollOnOffense, Level: 3,
			Text: "In order to meet the referee's count you crawl helplessly back into the ring."},
		{MinRoll: 10, MaxRoll: 10, Type: ChartOppRollDQ,
			Text: "The opponent comes out and tries to hit you with a steel chair! The referee warns him!",
			ThenType: ChartOppRollOnOffense, ThenLevel: 3},
		{MinRoll: 11, MaxRoll: 12, Type: ChartRollCountOut, AddFatigue: true,
			Text: "The opponent crushes you with a spectacular move outside the ring!"},
	},
}

// ─── DEATHJUMP ──────────────────────────────────────────────────────────────

var DeathjumpChart = ChartTable{
	RatingA: {
		{MinRoll: 2, MaxRoll: 2, Type: ChartRefDown,
			Text: "The opponent tries a spectacular move but accidentally smashes into the referee! The referee is down!"},
		{MinRoll: 3, MaxRoll: 4, Type: ChartRollPIN,
			Text: "The opponent comes off the top rope with an awesome cross body block but you counter with a specialty move!"},
		{MinRoll: 5, MaxRoll: 6, Type: ChartRollOnOffense, Level: 3,
			Text: "The opponent climbs to the top but you recover and throw him off the turnbuckle! He goes down hard!"},
		{MinRoll: 7, MaxRoll: 9, Type: ChartOppRollOnOffense, Level: 3,
			Text: "The opponent blasts you with a flying clothesline from the top ropes! You are in trouble!"},
		{MinRoll: 10, MaxRoll: 11, Type: ChartAgilityCheck,
			Text: "The opponent climbs to the top but you recover and climb up and a struggle takes place!"},
		{MinRoll: 12, MaxRoll: 12, Type: ChartRollYourPIN,
			Text: "You stumble to your feet and the opponent comes off the top turnbuckle with a perfect cross body block and covers you!"},
	},
	RatingB: {
		{MinRoll: 2, MaxRoll: 2, Type: ChartRefDown,
			Text: "The opponent tries a spectacular move but accidentally smashes into the referee! The referee is down!"},
		{MinRoll: 3, MaxRoll: 3, Type: ChartRollPIN,
			Text: "The opponent comes off the top rope with an awesome cross body block but you counter with a specialty move!"},
		{MinRoll: 4, MaxRoll: 5, Type: ChartRollOnOffense, Level: 3,
			Text: "The opponent climbs to the top but you recover and throw him off the turnbuckle! He goes down hard!"},
		{MinRoll: 6, MaxRoll: 9, Type: ChartOppRollOnOffense, Level: 3,
			Text: "The opponent blasts you with a flying clothesline from the top ropes! You are in trouble!"},
		{MinRoll: 10, MaxRoll: 10, Type: ChartAgilityCheck,
			Text: "The opponent climbs to the top but you recover and climb up and a struggle takes place!"},
		{MinRoll: 11, MaxRoll: 12, Type: ChartRollYourPIN,
			Text: "You stumble to your feet and the opponent comes off the top turnbuckle with a perfect cross body block and covers you!"},
	},
	RatingC: {
		{MinRoll: 2, MaxRoll: 2, Type: ChartRollPIN,
			Text: "The opponent comes off the top rope with an awesome cross body block but you counter with a specialty move!"},
		{MinRoll: 3, MaxRoll: 4, Type: ChartRollOnOffense, Level: 3,
			Text: "The opponent climbs to the top but you recover and throw him off the turnbuckle! He goes down hard!"},
		{MinRoll: 5, MaxRoll: 8, Type: ChartOppRollOnOffense, Level: 3,
			Text: "The opponent blasts you with a flying clothesline from the top ropes! You are in trouble!"},
		{MinRoll: 9, MaxRoll: 9, Type: ChartAgilityCheck,
			Text: "The opponent climbs to the top but you recover and climb up and a struggle takes place!"},
		{MinRoll: 10, MaxRoll: 12, Type: ChartRollYourPIN,
			Text: "You stumble to your feet and the opponent comes off the top turnbuckle with a perfect cross body block and covers you!"},
	},
}

// ─── CHOICE SITUATIONS ──────────────────────────────────────────────────────

// ChoiceOption represents one of the two moves available in a choice situation.
type ChoiceOption struct {
	Name      string           // Move name
	Power     int              // Move power (1-3)
	Threshold int              // Works on rolls of X or lower
	StatType  string           // "ag" or "pw" — which stat modifies the threshold
	IsChart   bool             // If true, triggers a chart instead of a roll check
	ChartRef  string           // "ropes", "turnbuckle", "deathjump"
}

// ChoiceSituation holds the two options for a choice letter.
type ChoiceSituation struct {
	Option1 ChoiceOption
	Option2 ChoiceOption
}

var ChoiceSituations = map[string]ChoiceSituation{
	"A": {
		Option1: ChoiceOption{Name: "Into the Ropes", IsChart: true, ChartRef: "ropes"},
		Option2: ChoiceOption{Name: "Belly to Belly Suplex", Power: 2, Threshold: 8, StatType: "pw"},
	},
	"B": {
		Option1: ChoiceOption{Name: "Standing Dropkick", Power: 2, Threshold: 8, StatType: "ag"},
		Option2: ChoiceOption{Name: "Into the Turnbuckle", IsChart: true, ChartRef: "turnbuckle"},
	},
	"C": {
		Option1: ChoiceOption{Name: "Moonsault", Power: 3, Threshold: 7, StatType: "ag"},
		Option2: ChoiceOption{Name: "Kick to Knee", Power: 2, Threshold: 7, StatType: "pw"},
	},
	"D": {
		Option1: ChoiceOption{Name: "Kick to Face", Power: 2, Threshold: 9, StatType: "ag"},
		Option2: ChoiceOption{Name: "Cobra Clutch Suplex", Power: 3, Threshold: 9, StatType: "pw"},
	},
	"E": {
		Option1: ChoiceOption{Name: "Scorpion Death Lock", Power: 3, Threshold: 9, StatType: "ag"},
		Option2: ChoiceOption{Name: "Power Slam", Power: 2, Threshold: 9, StatType: "pw"},
	},
	"F": {
		Option1: ChoiceOption{Name: "Leg Drop", Power: 2, Threshold: 7, StatType: "ag"},
		Option2: ChoiceOption{Name: "Running Lariat", Power: 3, Threshold: 7, StatType: "pw"},
	},
	"G": {
		Option1: ChoiceOption{Name: "Deathjump", IsChart: true, ChartRef: "deathjump"},
		Option2: ChoiceOption{Name: "Tombstone Piledriver", Power: 3, Threshold: 8, StatType: "pw"},
	},
	"H": {
		Option1: ChoiceOption{Name: "Flying Elbow Drop", Power: 3, Threshold: 8, StatType: "ag"},
		Option2: ChoiceOption{Name: "Deathjump", IsChart: true, ChartRef: "deathjump"},
	},
}

// ─── PIN SAVES (TAG MATCHES) ────────────────────────────────────────────────

type PinSaveOutcomeType int

const (
	PinSaveInterference PinSaveOutcomeType = iota // Roll on interference chart
	PinSaveSaved                                   // Partner saves, opponent rolls L3 offense
	PinSaveFailed                                  // Partner stopped, roll your PIN
	PinSaveBrawl                                   // Wild brawl, possible double DQ
	PinSaveReversed                                // Reversed pinning combination, opponent rolls PIN
)

type PinSaveOutcome struct {
	MinRoll int
	MaxRoll int
	Type    PinSaveOutcomeType
	Text    string
}

var PinSavesChart = []PinSaveOutcome{
	{MinRoll: 2, MaxRoll: 3, Type: PinSaveInterference,
		Text: "Your tag partner goes crazy and interferes in a big way!"},
	{MinRoll: 4, MaxRoll: 6, Type: PinSaveSaved,
		Text: "Your tag partner saves you and breaks the referee's count!"},
	{MinRoll: 7, MaxRoll: 10, Type: PinSaveFailed,
		Text: "Your tag partner tries to help, but is stopped by the opponent's tag partner!"},
	{MinRoll: 11, MaxRoll: 11, Type: PinSaveBrawl,
		Text: "Your tag partner runs in and so does the opponent's tag partner. A wild brawl erupts!"},
	{MinRoll: 12, MaxRoll: 12, Type: PinSaveReversed,
		Text: "Your tag partner runs in and reverses the pinning combination to put you on top!"},
}

func LookupPinSave(roll int) *PinSaveOutcome {
	for i := range PinSavesChart {
		if roll >= PinSavesChart[i].MinRoll && roll <= PinSavesChart[i].MaxRoll {
			return &PinSavesChart[i]
		}
	}
	return nil
}

// ─── OUTSIDE INTERFERENCE ───────────────────────────────────────────────────

type InterferenceOutcomeType int

const (
	InterfDoubleTeam    InterferenceOutcomeType = iota // DQ 8, then opponent rolls PIN
	InterfFinisher                                      // DQ 7, then finisher + PIN
	InterfAttackAndPin                                  // DQ 6, then opponent rolls PIN
	InterfAttackAndL3                                   // DQ 5, then roll L3 offense
	InterfBrawl                                         // DQ 4, then coin flip for L3
	InterfDistract                                      // Ally distracts ref, opp L3 offense
	InterfBackfire                                      // Opponent wins brawl, roll your PIN
	InterfBackfireFinish                                // Opponent wins, roll PIN + finisher
)

type InterferenceOutcome struct {
	MinRoll     int
	MaxRoll     int
	Type        InterferenceOutcomeType
	DQThreshold int
	Text        string
}

var InterferenceChart = []InterferenceOutcome{
	{MinRoll: 2, MaxRoll: 3, Type: InterfDoubleTeam, DQThreshold: 8,
		Text: "Your ally attacks the opponent from behind! Double-team with deadly specialty moves!"},
	{MinRoll: 4, MaxRoll: 4, Type: InterfFinisher, DQThreshold: 7,
		Text: "Your ally attacks the opponent with a deadly specialty move!"},
	{MinRoll: 5, MaxRoll: 5, Type: InterfAttackAndPin, DQThreshold: 6,
		Text: "Your ally attacks the opponent with a deadly specialty move!"},
	{MinRoll: 6, MaxRoll: 6, Type: InterfAttackAndL3, DQThreshold: 5,
		Text: "Your ally attacks the opponent with a deadly specialty move!"},
	{MinRoll: 7, MaxRoll: 7, Type: InterfBrawl, DQThreshold: 4,
		Text: "Your ally attacks the opponent and a brawl results!"},
	{MinRoll: 8, MaxRoll: 9, Type: InterfDistract,
		Text: "Your ally distracts the referee breaking the pin count. The referee orders him to leave."},
	{MinRoll: 10, MaxRoll: 10, Type: InterfBackfire,
		Text: "Your ally storms the ring but the opponent wins the ensuing brawl and throws him out!"},
	{MinRoll: 11, MaxRoll: 12, Type: InterfBackfireFinish,
		Text: "Your ally storms the ring but the opponent wins the brawl! He motions to the crowd — finisher time!"},
}

func LookupInterference(roll int) *InterferenceOutcome {
	for i := range InterferenceChart {
		if roll >= InterferenceChart[i].MinRoll && roll <= InterferenceChart[i].MaxRoll {
			return &InterferenceChart[i]
		}
	}
	return nil
}

// ─── FEUD TABLE ─────────────────────────────────────────────────────────────

type FeudOutcomeType int

const (
	FeudAttackedByLoser   FeudOutcomeType = iota // Opponent attacks winner, injury 2 cards
	FeudAllyDoubleTeam                            // Your ally storms ring, double-team loser
	FeudPostMatchAttack                           // You attack after bell, opponent injured 1 card
	FeudFourManBrawl                              // Wild brawl, leads to tag match
	FeudOpponentAlly                              // Attacked by opponent's ally, injury 2 cards
	FeudGangAttack                                // Call allies, total destruction
)

type FeudOutcome struct {
	MinRoll    int
	MaxRoll    int
	Type       FeudOutcomeType
	InjuryDays int // Fight cards of injury
	Text       string
}

var FeudTable = []FeudOutcome{
	{MinRoll: 2, MaxRoll: 4, Type: FeudAttackedByLoser, InjuryDays: 2,
		Text: "You celebrate your victory! The opponent recovers and attacks you from behind with his finisher! YOU ARE INJURED FOR TWO FIGHT CARDS."},
	{MinRoll: 5, MaxRoll: 6, Type: FeudAllyDoubleTeam,
		Text: "One of your allies storms the ring and you double-team the opponent! He challenges your ally to a match!"},
	{MinRoll: 7, MaxRoll: 7, Type: FeudPostMatchAttack, InjuryDays: 1,
		Text: "You continue your attack after the bell! OPPONENT IS INJURED FOR ONE FIGHT CARD."},
	{MinRoll: 8, MaxRoll: 9, Type: FeudFourManBrawl,
		Text: "You are attacked by an ally of the opponent. Your ally rushes to the ring — a wild four-man brawl erupts!"},
	{MinRoll: 10, MaxRoll: 10, Type: FeudOpponentAlly, InjuryDays: 2,
		Text: "You are attacked by the opponent's ally! They double team you! YOU ARE INJURED FOR TWO FIGHT CARDS."},
	{MinRoll: 11, MaxRoll: 12, Type: FeudGangAttack,
		Text: "You call your allies to ringside. Total destruction ensues! ROLL ONE DIE FOR INJURY AND SUSPENSION!"},
}

func LookupFeud(roll int) *FeudOutcome {
	for i := range FeudTable {
		if roll >= FeudTable[i].MinRoll && roll <= FeudTable[i].MaxRoll {
			return &FeudTable[i]
		}
	}
	return nil
}
