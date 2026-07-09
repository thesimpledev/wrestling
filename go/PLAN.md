# Architecture & Data Models Plan

## Directory Structure

```
wrestling/
├── main.go                     # Entry point, Ebitengine game loop setup
├── go.mod
├── go.sum
│
├── data/                       # Wrestler card data files (user-created)
│   └── wrestlers/
│       └── example.yaml        # Example wrestler card
│
├── engine/                     # Core match simulation (no UI dependency)
│   ├── match.go                # Match state machine, turn loop
│   ├── wrestler.go             # Wrestler card data model
│   ├── dice.go                 # Dice rolling (1d6, 2d6)
│   ├── charts.go               # All game charts (ropes, turnbuckle, etc.)
│   ├── resolve.go              # Move resolution, pin/dq/countout logic
│   └── events.go               # Event types emitted by the engine
│
├── ui/                         # Ebitengine rendering & input
│   ├── game.go                 # Implements ebiten.Game interface
│   ├── screen_match.go         # Match screen (ring + commentary)
│   ├── screen_menu.go          # Main menu screen
│   ├── screen_cardeditor.go    # Card editor screen (later milestone)
│   ├── commentary.go           # Scrolling text log
│   ├── ring.go                 # Ring sprite rendering
│   └── input.go                # Input handling (step/auto-play/speed)
│
└── loader/                     # Card loading & validation
    └── loader.go               # YAML/JSON -> engine.WrestlerCard
```

## Data Models

### WrestlerCard

The core data structure - represents one physical game card.

```go
type WrestlerCard struct {
    Name     string    `yaml:"name"`

    // Offense: 3 levels, each with 6 moves (indexed by die roll 1-6)
    Offense [3][6]Move `yaml:"offense"`

    // Defense: 3 levels, each with 6 outcomes (indexed by die roll 1-6)
    Defense [3][6]DefenseOutcome `yaml:"defense"`

    // Ratings
    Ropes        Rating `yaml:"ropes"`         // A, B, or C
    Turnbuckle   Rating `yaml:"turnbuckle"`    // A, B, or C
    Ring         Rating `yaml:"ring"`          // A, B, or C
    Deathjump    Rating `yaml:"deathjump"`     // A, B, or C
    PIN          int    `yaml:"pin"`           // Base PIN number (basic rules)
    PINAdvanced  int    `yaml:"pin_advanced"`  // PIN in parentheses (advanced rules)
    Cage         int    `yaml:"cage"`          // Cage match PIN replacement
    DQ           int    `yaml:"dq"`            // Disqualification rating
    Agility      int    `yaml:"agility"`       // -5 to +5
    Power        int    `yaml:"power"`         // -5 to +5
    Distractor   int    `yaml:"distractor"`    // Default 5 if not set

    // Finisher
    Finisher     Finisher `yaml:"finisher"`
}

type Rating int // RatingA=0, RatingB=1, RatingC=2

type Move struct {
    Name       string   `yaml:"name"`
    Power      int      `yaml:"power"`      // 1, 2, or 3
    DefLevel   int      `yaml:"def_level"`  // Which defense level opponent checks (1-3)
    IsFinisher bool     `yaml:"is_finisher"`
    Tags       []string `yaml:"tags"`       // "ch:A", "ag", "pw", "dis", "add1", "tag", "singles", "lv", "roll"
}

type DefenseOutcome struct {
    Type       string `yaml:"type"`        // "dazed", "hurt", "down", "reversal", "pin", "chart"
    Power      int    `yaml:"power"`       // 1, 2, or 3 (for dazed/hurt/down)
    Tags       []string `yaml:"tags"`      // "tag", "lv", etc.
    ChartRef   string `yaml:"chart_ref"`   // "ropes", "turnbuckle", "ring", "deathjump" (if type=chart)
}

type Finisher struct {
    Name   string `yaml:"name"`
    Rating int    `yaml:"rating"`  // +0 to +5 (or higher), added to PIN
    IsRoll bool   `yaml:"is_roll"` // Roll finisher (variable rating from die roll)
    RollMin int   `yaml:"roll_min"` // Min roll for success (roll finishers)
    RollMax int   `yaml:"roll_max"` // Max roll for success (roll finishers)
}
```

### Match State

Everything the engine tracks during a match.

```go
type Match struct {
    Type        MatchType  // Singles, Tag, Cage, NoDQ
    IsFeud      bool

    Sides       [2]*Side   // Two sides (each side can have multiple wrestlers for tag)
    OnOffense   int        // 0 or 1 - which side is attacking
    OffLevel    int        // Current offense level (1, 2, or 3)

    RefDown     bool       // Referee knocked out (from deathjump chart)
    RefDownTurns int       // How many moves ref will miss

    Events      []Event    // Log of everything that happened (feeds commentary)
    TurnNumber  int
    Over        bool
    Result      *MatchResult
}

type Side struct {
    Wrestlers    []*WrestlerState  // 1 for singles, 2+ for tag
    ActiveIndex  int               // Which wrestler is in the ring
    PinSavesUsed int               // Max 2 per match (tag only)
}

type WrestlerState struct {
    Card              *WrestlerCard
    CurrentPIN        int   // Starts at card's PIN, increases with fatigue
    Injured           bool
    InjuryCardsLeft   int
    InterferenceUsed  bool  // Once per match
    DistractionUsed   bool  // Once per match
}

type MatchResult struct {
    Winner      *WrestlerCard
    Loser       *WrestlerCard
    Method      string // "pin", "submission", "dq", "countout", "cage_escape"
    Description string // Flavor text for the finish
}
```

### Event System

The engine emits events; the UI consumes them for commentary and animation.

```go
type EventType int

const (
    EventRoll           EventType = iota // A die/dice was rolled
    EventMove                            // An offensive move was performed
    EventDefense                         // Defense outcome resolved
    EventPin                             // PIN attempt
    EventFinisher                        // Finisher attempted
    EventChart                           // Chart consulted (ropes/turnbuckle/etc.)
    EventDQ                              // DQ check
    EventCountOut                        // Count-out check
    EventTagIn                           // Tag partner enters
    EventTagAttempt                      // Attempted tag on defense
    EventPinSave                         // Tag partner pin save attempt
    EventInterference                    // Outside interference
    EventDistraction                     // Manager/ally distraction
    EventRefDown                         // Referee knocked out
    EventRefRecover                      // Referee recovers
    EventControlSwitch                   // Offense control changes
    EventMatchEnd                        // Match is over
)

type Event struct {
    Type        EventType
    Description string            // Human-readable commentary text
    Data        map[string]any    // Structured data for UI (roll values, wrestler names, etc.)
}
```

## Match Engine State Machine

```
                    ┌──────────────┐
                    │  MATCH_START │
                    │ (roll for    │
                    │  initiative) │
                    └──────┬───────┘
                           │
                           v
               ┌───────────────────────┐
        ┌─────>│    ATTACKER_ROLL      │<────────────────┐
        │      │ (roll 1d6 on current  │                 │
        │      │  offense level)       │                 │
        │      └───────────┬───────────┘                 │
        │                  │                             │
        │                  v                             │
        │      ┌───────────────────────┐                 │
        │      │   RESOLVE_MOVE        │                 │
        │      │ (check tags: ch, ag,  │                 │
        │      │  pw, dis, chart, etc) │                 │
        │      └───────────┬───────────┘                 │
        │                  │                             │
        │         ┌────────┼────────┐                    │
        │         v        v        v                    │
        │    [CHART]   [CHOICE]  [NORMAL]                │
        │      │         │        │                      │
        │      v         v        v                      │
        │      ┌───────────────────────┐                 │
        │      │   DEFENDER_ROLL       │                 │
        │      │ (roll 1d6 on defense  │                 │
        │      │  level indicated by   │                 │
        │      │  move's power)        │                 │
        │      └───────────┬───────────┘                 │
        │                  │                             │
        │         ┌────────┴────────┐                    │
        │         v                 v                    │
        │  [dazed/hurt/down]   [REVERSAL]                │
        │         │                 │                    │
        │         │            (switch offense,          │
        │         │             set level)               │
        │         │                 │                    │
        │         v                 └────────────────────┘
        │  ┌──────────────┐
        │  │ CHECK_PIN?   │──(if PIN/finisher on defense)──> PIN_ATTEMPT
        │  │ CHECK_ADD1?  │                                     │
        │  └──────┬───────┘                           ┌────────┴────────┐
        │         │                                   v                 v
        │         │                               [PINNED]         [KICKED OUT]
        │         │                                   │                 │
        └─────────┘                                   v                 │
                                              ┌──────────────┐         │
                                              │  MATCH_END   │         │
                                              └──────────────┘         │
                                                                       │
                                                       ┌───────────────┘
                                                       │ (add fatigue,
                                                       │  opponent L3 offense)
                                                       └────────────────────> back to ATTACKER_ROLL
```

Key state transitions not shown above (to avoid spaghetti):
- **Chart situations** (ropes/turnbuckle/ring/deathjump): roll 2d6, look up on chart by rating, resolve outcome (which may itself trigger PIN, DQ, control switch, or another chart)
- **DQ checks**: roll 2d6 vs DQ rating, if ≤ rating -> MATCH_END by DQ
- **Count-out checks**: roll 2d6 vs PIN rating, if ≤ rating -> MATCH_END by count-out
- **Tag**: on offense = free switch; on defense = roll 2d6, ≤4 = success
- **Pin saves** (tag only): roll 2d6 on pin saves chart before resolving PIN
- **Interference/Distraction**: optional player actions at specific moments

## Charts Data

All charts will be defined as lookup tables in `charts.go`. Each chart is a map from `Rating` to a list of `{minRoll, maxRoll, outcome}` entries. The outcome is a function or struct describing what happens (control switch, PIN attempt, DQ check, etc.).

Charts to implement:
1. **Into the Ropes** (Rating A/B/C, roll 2d6)
2. **Into the Turnbuckle** (Rating A/B/C, roll 2d6)
3. **Out of the Ring** (Rating A/B/C, roll 2d6)
4. **Deathjump** (Rating A/B/C, roll 2d6)
5. **Choice Situations** (A-H, two move options each)
6. **Feud Table** (roll 2d6, post-match)
7. **Pin Saves** (roll 2d6, tag matches)
8. **Outside Interference** (roll 2d6)

## Implementation Milestones

### Milestone 1: Core Engine (no UI)
- [ ] `go mod init`, project scaffolding
- [ ] `WrestlerCard` and `Move` data models
- [ ] YAML card loader with example card
- [ ] Dice rolling
- [ ] Basic match loop: offense roll -> defense roll -> reversal or continue
- [ ] PIN resolution (basic rules)
- [ ] Finisher resolution
- [ ] Event system
- [ ] Run a match from `main.go` printing events to stdout

### Milestone 2: Full Rules Engine
- [ ] All 8 charts implemented
- [ ] DQ and count-out mechanics
- [ ] Choice situations (ch)
- [ ] Agility/Power checks (ag/pw)
- [ ] Add1 moves
- [ ] Advanced fatigue system (PIN increments)
- [ ] Roll finishers
- [ ] Leaving the ring (lv)

### Milestone 3: Tag Team
- [ ] Multi-wrestler sides
- [ ] Tag in on offense
- [ ] Tag out on defense (roll check)
- [ ] Pin saves chart
- [ ] Tag-only moves
- [ ] Double DQ

### Milestone 4: Special Situations
- [ ] Outside interference chart
- [ ] Distraction mechanic
- [ ] Ringside ally interactions
- [ ] Referee down (from deathjump)
- [ ] Feud table (post-match)
- [ ] Injury tracking across cards

### Milestone 5: Ebitengine UI
- [ ] Basic window with ring sprite
- [ ] Scrolling commentary log
- [ ] Step-through mode (click/key to advance)
- [ ] Auto-play with speed control
- [ ] Main menu (pick wrestlers, match type)

### Milestone 6: Special Match Types & Card Editor
- [ ] Cage match rules
- [ ] No-DQ match rules
- [ ] In-app card editor GUI
- [ ] Card validation
