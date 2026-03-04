package engine

import (
	"fmt"
	"math/rand/v2"
)

func pick(texts []string) string {
	return texts[rand.IntN(len(texts))]
}

// ─── DEFENSE COMMENTARY ─────────────────────────────────────────────────────

var dazedTexts = []string{
	"%s is dazed but still on his feet!",
	"%s staggers but refuses to go down!",
	"%s shakes it off — not enough to put him away!",
	"%s absorbs the blow and stays standing!",
	"%s is stunned but hangs on!",
	"%s wobbles but won't go down that easy!",
}

var hurtTexts = []string{
	"%s is hurt! That one connected!",
	"%s takes a nasty shot — he's in trouble!",
	"%s is reeling from that devastating blow!",
	"%s is in pain! The punishment continues!",
	"%s feels that one! He's hurting!",
	"%s doubled over in agony!",
}

var downTexts = []string{
	"%s is DOWN! He's in serious trouble now!",
	"%s goes down HARD! This could be it!",
	"%s crashes to the mat! He's not getting up easily!",
	"%s is laid out! The crowd is on their feet!",
	"%s is down and nearly out!",
	"%s hits the canvas — devastating!",
}

var reversalTexts = []string{
	"%s counters and fights back!",
	"%s reverses the hold with authority!",
	"%s blocks the move and turns the tables!",
	"%s escapes and fires back with a fury!",
	"%s with an incredible counter! He takes over!",
	"%s reverses! Now HE'S on the attack!",
	"%s fights out of it! Momentum shift!",
}

// CommentaryDefense returns a varied commentary line for defense outcomes.
func CommentaryDefense(wrestler string, outcome DefenseType) string {
	switch outcome {
	case DefDazed:
		return fmt.Sprintf(pick(dazedTexts), wrestler)
	case DefHurt:
		return fmt.Sprintf(pick(hurtTexts), wrestler)
	case DefDown:
		return fmt.Sprintf(pick(downTexts), wrestler)
	case DefReversal:
		return fmt.Sprintf(pick(reversalTexts), wrestler)
	default:
		return fmt.Sprintf("%s is %s!", wrestler, outcome)
	}
}

// ─── PIN COMMENTARY ──────────────────────────────────────────────────────────

var pinSuccessTexts = []string{
	"PIN ATTEMPT! %s rolls %d (needed %d to kick out) — HE'S BEEN PINNED! 1... 2... 3!",
	"THE COVER! %s rolls %d (needed %d) — IT'S OVER! THREE COUNT!",
	"HE COVERS HIM! %s rolls %d (needed %d) — THAT'S IT! THE PIN IS GOOD!",
	"INTO THE COVER! %s rolls %d (needed %d) — ONE! TWO! THREE! HE GOT HIM!",
}

var pinFailTexts = []string{
	"PIN ATTEMPT! %s rolls %d (needed %d to kick out) — HE KICKS OUT!",
	"THE COVER! %s rolls %d (needed %d) — NO! HE POWERS OUT AT TWO!",
	"HE GOES FOR THE PIN! %s rolls %d (needed %d) — KICKOUT! Just barely!",
	"COVER! %s rolls %d (needed %d) — NOT ENOUGH! He survives!",
}

// CommentaryPin returns varied pin attempt commentary.
func CommentaryPin(wrestler string, roll, threshold int, pinned bool) string {
	if pinned {
		return fmt.Sprintf(pick(pinSuccessTexts), wrestler, roll, threshold+1)
	}
	return fmt.Sprintf(pick(pinFailTexts), wrestler, roll, threshold+1)
}

// ─── FINISHER COMMENTARY ─────────────────────────────────────────────────────

var finisherSetupTexts = []string{
	"%s signals to the crowd — it's %s time!!",
	"%s is going for it! THE %s!!",
	"THIS IS IT! %s sets up the %s!!",
	"The crowd ERUPTS! %s is going for the %s!!",
	"%s has that look in his eye — %s incoming!!",
}

// CommentaryFinisher returns varied finisher setup commentary.
func CommentaryFinisher(attacker, finisherName string) string {
	return fmt.Sprintf(pick(finisherSetupTexts), attacker, finisherName)
}

// ─── MATCH START/END COMMENTARY ─────────────────────────────────────────────

var matchStartTexts = []string{
	"The bell rings! %s vs %s — HERE WE GO!",
	"The match is underway! %s faces off against %s!",
	"AND WE'RE OFF! %s vs %s — this is going to be a war!",
	"The crowd is electric! %s vs %s — LET'S DO THIS!",
}

var matchEndTexts = []string{
	"MATCH OVER! %s defeats %s by %s!",
	"IT'S ALL OVER! %s wins by %s over %s!",
	"THERE IT IS! %s has done it! Victory by %s over %s!",
	"THE WINNER BY %s — %s! What a match against %s!",
}

// CommentaryMatchStart returns varied match start commentary.
func CommentaryMatchStart(name1, name2 string) string {
	return fmt.Sprintf(pick(matchStartTexts), name1, name2)
}

// CommentaryMatchEnd returns varied match end commentary.
func CommentaryMatchEnd(winner, loser, method string) string {
	texts := matchEndTexts
	idx := rand.IntN(len(texts))
	switch idx {
	case 0:
		return fmt.Sprintf("MATCH OVER! %s defeats %s by %s!", winner, loser, method)
	case 1:
		return fmt.Sprintf("IT'S ALL OVER! %s wins by %s over %s!", winner, method, loser)
	case 2:
		return fmt.Sprintf("THERE IT IS! %s has done it! Victory by %s over %s!", winner, method, loser)
	default:
		return fmt.Sprintf("THE WINNER BY %s — %s! What a match against %s!", method, winner, loser)
	}
}

// ─── MOVE COMMENTARY ─────────────────────────────────────────────────────────

var moveConnectTexts = []string{
	"%s hits %s with a %s!",
	"%s nails %s with the %s!",
	"%s delivers a devastating %s to %s!",
	"%s connects with the %s on %s!",
}

// CommentaryMove returns varied move commentary.
func CommentaryMove(attacker, defender, moveName string) string {
	idx := rand.IntN(len(moveConnectTexts))
	switch idx {
	case 0:
		return fmt.Sprintf("%s hits %s with a %s!", attacker, defender, moveName)
	case 1:
		return fmt.Sprintf("%s nails %s with the %s!", attacker, defender, moveName)
	case 2:
		return fmt.Sprintf("%s delivers a devastating %s to %s!", attacker, moveName, defender)
	default:
		return fmt.Sprintf("%s connects with the %s on %s!", attacker, moveName, defender)
	}
}

// ─── CROWD/ATMOSPHERE ────────────────────────────────────────────────────────

var crowdReactions = []string{
	"The crowd is going wild!",
	"The fans are on their feet!",
	"Listen to this crowd!",
	"The atmosphere is electric!",
	"The arena is shaking!",
	"What a match this has been!",
}

// CommentaryCrowdReaction returns a random crowd reaction line.
func CommentaryCrowdReaction() string {
	return pick(crowdReactions)
}
