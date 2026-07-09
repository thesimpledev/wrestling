package engine

// Rating represents a wrestler's chart rating (A, B, or C).
type Rating int

const (
	RatingA Rating = iota
	RatingB
	RatingC
)

func (r Rating) String() string {
	switch r {
	case RatingA:
		return "A"
	case RatingB:
		return "B"
	case RatingC:
		return "C"
	default:
		return "?"
	}
}

// MoveTag represents special instructions attached to a move.
type MoveTag string

const (
	TagChart   MoveTag = "chart"   // Triggers a chart lookup (ropes/turnbuckle/ring/deathjump)
	TagChoice  MoveTag = "ch"      // Choice situation (followed by letter A-H)
	TagAgility MoveTag = "ag"      // Requires agility check
	TagPower   MoveTag = "pw"      // Requires power check
	TagDQ      MoveTag = "dis"     // May cause disqualification
	TagAdd1    MoveTag = "add1"    // Adds 1 to opponent's PIN rating
	TagTagTeam MoveTag = "tag"     // Tag team only move
	TagSingles MoveTag = "singles" // Singles only move
	TagLeave   MoveTag = "lv"      // Option to leave the ring
	TagRoll    MoveTag = "roll"    // Roll finisher
)

// Move represents one entry in a wrestler's offense grid.
type Move struct {
	Name     string  // e.g. "Hammerlock", "STONE COLD STUNNER"
	Power    int     // 1, 2, or 3 - how powerful the move is
	DefLevel int     // Which defense level opponent checks (1, 2, or 3)
	Tags     []MoveTag
	// For chart moves: which chart to consult
	ChartType string // "ropes", "turnbuckle", "ring", "deathjump"
	// For choice moves: which choice letter (A-H)
	ChoiceKey string
}

// IsFinisher returns true if the move name is in ALL CAPS (the game's convention).
func (m Move) IsFinisher() bool {
	if len(m.Name) == 0 {
		return false
	}
	for _, r := range m.Name {
		if r >= 'a' && r <= 'z' {
			return false
		}
	}
	return true
}

func (m Move) HasTag(tag MoveTag) bool {
	for _, t := range m.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// DefenseType describes what happens when a defender rolls on their defense grid.
type DefenseType int

const (
	DefDazed    DefenseType = iota // Not seriously hurt
	DefHurt                        // More damage taken
	DefDown                        // At weakest
	DefReversal                    // Counters and takes offense
	DefPIN                         // PIN attempt triggered
)

func (d DefenseType) String() string {
	switch d {
	case DefDazed:
		return "Dazed"
	case DefHurt:
		return "Hurt"
	case DefDown:
		return "Down"
	case DefReversal:
		return "Reversal"
	case DefPIN:
		return "PIN"
	default:
		return "?"
	}
}

// DefenseOutcome represents one entry in a wrestler's defense grid.
type DefenseOutcome struct {
	Type  DefenseType
	Power int       // 1, 2, or 3 (for dazed/hurt/down outcomes)
	Tags  []MoveTag // e.g. "tag", "lv"
	// For PIN outcomes, the threshold is stored as PINThreshold
	// (roll 2d6, if <= threshold, pinned)
	PINThreshold int
}

func (d DefenseOutcome) HasTag(tag MoveTag) bool {
	for _, t := range d.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Finisher represents a wrestler's finishing move details.
type Finisher struct {
	Name    string // e.g. "STONE COLD STUNNER"
	Rating  int    // Bonus added to PIN rating for finisher attempts (+0 to +5+)
	IsRoll  bool   // True if this is a roll finisher (variable rating)
	RollMin int    // Min die roll for success (roll finishers only)
	RollMax int    // Max die roll for success (roll finishers only)
}

// WrestlerCard represents a complete wrestler playing card.
type WrestlerCard struct {
	Name string

	// Offense grid: [level 0-2][roll 0-5] (level 0 = Level 1, roll 0 = die roll 1)
	Offense [3][6]Move

	// Defense grid: [level 0-2][roll 0-5]
	Defense [3][6]DefenseOutcome

	// Chart ratings
	Ropes      Rating
	Turnbuckle Rating
	Ring       Rating
	Deathjump  Rating

	// Numeric ratings
	PIN        int // Base PIN rating (basic rules)
	PINAdv     int // PIN rating in parentheses (advanced/fatigue rules)
	Cage       int // Cage match PIN replacement
	DQ         int // Disqualification rating
	Agility    int // -5 to +5
	Power      int // -5 to +5
	Distractor int // Default 5

	Finisher Finisher
}
