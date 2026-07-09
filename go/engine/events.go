package engine

import "fmt"

// EventType categorizes match events for the UI/commentary system.
type EventType int

const (
	EventMatchStart    EventType = iota
	EventRoll                    // A die/dice was rolled
	EventMove                    // An offensive move was performed
	EventDefense                 // Defense outcome resolved
	EventControlSwitch           // Offense control changes sides
	EventPin                     // PIN attempt
	EventFinisher                // Finisher attempted
	EventDQ                      // Disqualification check
	EventCountOut                // Count-out check
	EventChart                   // Chart consulted (ropes/turnbuckle/etc.)
	EventTagIn                   // Tag partner enters
	EventTagAttempt              // Attempted tag on defense
	EventPinSave                 // Tag partner pin save attempt
	EventInterference            // Outside interference
	EventDistraction             // Manager/ally distraction
	EventRefDown                 // Referee knocked out
	EventRefRecover              // Referee recovers
	EventFatigue                 // PIN rating increased
	EventMatchEnd                // Match is over
)

// Event represents something that happened during a match.
// The engine emits these; the UI consumes them for commentary and display.
type Event struct {
	Type     EventType
	Text     string         // Human-readable commentary line
	Attacker string         // Name of wrestler on offense (if applicable)
	Defender string         // Name of wrestler on defense (if applicable)
	Roll     int            // Die/dice result (if applicable)
	Level    int            // Offense/defense level (if applicable)
}

func (e Event) String() string {
	return e.Text
}

// Convenience constructors

func newEvent(typ EventType, format string, args ...any) Event {
	return Event{
		Type: typ,
		Text: fmt.Sprintf(format, args...),
	}
}

func moveEvent(attacker, defender, moveName string, power, level int) Event {
	return Event{
		Type:     EventMove,
		Text:     fmt.Sprintf("%s hits %s with a %s!", attacker, defender, moveName),
		Attacker: attacker,
		Defender: defender,
		Level:    level,
	}
}

func defenseEvent(wrestler string, outcome DefenseType) Event {
	return Event{
		Type:     EventDefense,
		Text:     CommentaryDefense(wrestler, outcome),
		Defender: wrestler,
	}
}

func reversalEvent(wrestler string) Event {
	return Event{
		Type: EventControlSwitch,
		Text: CommentaryDefense(wrestler, DefReversal),
	}
}

func pinEvent(wrestler string, roll, threshold int, pinned bool) Event {
	return Event{
		Type: EventPin,
		Text: CommentaryPin(wrestler, roll, threshold, pinned),
		Roll: roll,
	}
}

func finisherEvent(attacker, finisherName string) Event {
	return Event{
		Type:     EventFinisher,
		Text:     CommentaryFinisher(attacker, finisherName),
		Attacker: attacker,
	}
}

func matchEndEvent(winner, loser, method string) Event {
	return Event{
		Type: EventMatchEnd,
		Text: CommentaryMatchEnd(winner, loser, method),
	}
}
