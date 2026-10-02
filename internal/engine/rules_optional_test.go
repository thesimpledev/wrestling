package engine

import (
	"strings"
	"testing"
)

func optionalChartMove(chart string) Move {
	return Move{Name: "Risky Chart Move", Power: 2, DefLevel: 2, Tags: []MoveTag{TagChart, TagChartChoice}, ChartType: chart}
}

// levelTwoSpecial builds an attacker whose Level 2 is all one special move
// and whose other levels are plain jabs.
func levelTwoSpecial(special Move) *WrestlerCard {
	card := testCard("A")
	for slot := range card.Offense[1] {
		card.Offense[1][slot] = special
	}
	return card
}

func TestDefaultRulesTurnEveryOptionalRuleOn(t *testing.T) {
	want := Rules{ChartChoice: true, AvoidDisMoves: true, CustomDisNumbers: true}
	if got := DefaultRules(); got != want {
		t.Fatalf("default rules: %+v", got)
	}
	if got := NewMatch(testCard("A"), testCard("D")).Rules; got != want {
		t.Fatalf("new match rules: %+v", got)
	}
}

func TestOptionalChartMoveIsSkippedAgainstAnExcellentRating(t *testing.T) {
	att, def := levelTwoSpecial(optionalChartMove("ropes")), testCard("D")
	def.Ropes = RatingA
	dice := script(1, 1, 1)
	m := singlesMatch(att, def, dice)
	m.offLevel = 1
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantLog(t, m, "decides against")
	wantNoLog(t, m, "ropes Chart")
	if got := lastMoveText(m); !strings.Contains(got, "Level 1") || !strings.Contains(got, "Jab") {
		t.Fatalf("move performed: %q, want a Level 1 Jab", got)
	}
	wantOffense(t, m, 0, 1)
}

func TestOptionalChartMoveIsUsedWhenThereIsNoReasonToSkip(t *testing.T) {
	cases := []struct {
		name   string
		rating Rating
		rules  Rules
		move   Move
	}{
		{"average rating", RatingB, DefaultRules(), optionalChartMove("ropes")},
		{"poor rating", RatingC, DefaultRules(), optionalChartMove("ropes")},
		{"rule switched off", RatingA, Rules{}, optionalChartMove("ropes")},
		{"move is not optional", RatingA, DefaultRules(), chartMove("ropes")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			att, def := levelTwoSpecial(tc.move), testCard("D")
			def.Ropes = tc.rating
			m := singlesMatch(att, def, script(1, 6, 6))
			m.Rules = tc.rules
			m.offLevel = 1
			m.executeTurn()
			wantLog(t, m, "ropes Chart")
			wantNoLog(t, m, "decides against")
		})
	}
}

func TestOnlyOneSkipPerTurn(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(optionalChartMove("ropes"))
	def.Ropes = RatingA
	dice := script(1, 1, 6, 6)
	m := singlesMatch(att, def, dice)
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantLog(t, m, "decides against")
	wantLog(t, m, "ropes Chart")
}

func TestOptionalOutOfRingMoveIsNotSkippedInACage(t *testing.T) {
	att, def := levelTwoSpecial(optionalChartMove("ring")), testCard("D")
	def.Ring = RatingA
	m := cageMatch(att, def, script(1, 1))
	m.offLevel = 1
	m.executeTurn()
	wantLog(t, m, "face-first into the cage")
	wantNoLog(t, m, "decides against")
}

func TestRiskyDisMoveIsSkipped(t *testing.T) {
	att, def := levelTwoSpecial(taggedMove("Eye Gouge", 2, 2, TagDQ)), testCard("D")
	att.DQ = 6
	dice := script(1, 1, 1)
	m := singlesMatch(att, def, dice)
	m.offLevel = 1
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantLog(t, m, "decides against")
	wantNoLog(t, m, "for disqualification")
}

func TestDisMoveIsUsedWhenThereIsNoReasonToSkip(t *testing.T) {
	cases := []struct {
		name      string
		dq        int
		rules     Rules
		matchType MatchType
		wantDQLog bool
	}{
		{"low DQ rating", 5, DefaultRules(), MatchSingles, true},
		{"rule switched off", 6, Rules{}, MatchSingles, true},
		{"no DQ match", 12, DefaultRules(), MatchNoDQ, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			att, def := levelTwoSpecial(taggedMove("Eye Gouge", 2, 2, TagDQ)), testCard("D")
			att.DQ = tc.dq
			m := singlesMatch(att, def, script(1, 6, 6, 1))
			m.Type = tc.matchType
			m.Rules = tc.rules
			m.offLevel = 1
			m.executeTurn()
			wantNoLog(t, m, "decides against")
			wantLog(t, m, "Eye Gouge")
			if got := strings.Contains(eventTexts(m), "for disqualification"); got != tc.wantDQLog {
				t.Fatalf("DQ roll in log: %v, want %v\n%s", got, tc.wantDQLog, eventTexts(m))
			}
		})
	}
}

func TestCustomDisNumberReplacesTheCardRating(t *testing.T) {
	special := taggedMove("Chair Shot", 2, 2, TagDQ)
	special.DQNumber = 9

	withRule := singlesMatch(levelTwoSpecial(special), testCard("D"), script(1, 4, 4))
	withRule.Sides[0].Active().Card.DQ = 2
	withRule.Rules = Rules{CustomDisNumbers: true}
	withRule.offLevel = 1
	withRule.executeTurn()
	wantResult(t, withRule, "D", "dq")

	withoutRule := singlesMatch(levelTwoSpecial(special), testCard("D"), script(1, 4, 4, 1))
	withoutRule.Sides[0].Active().Card.DQ = 2
	withoutRule.Rules = Rules{}
	withoutRule.offLevel = 1
	withoutRule.executeTurn()
	if withoutRule.Over() {
		t.Fatalf("roll 8 against card DQ 2 should be safe\n%s", eventTexts(withoutRule))
	}
}

func TestSkipDecisionUsesTheCustomDisNumber(t *testing.T) {
	special := taggedMove("Chair Shot", 2, 2, TagDQ)
	special.DQNumber = 7
	att := levelTwoSpecial(special)
	att.DQ = 2
	m := singlesMatch(att, testCard("D"), script(1, 1, 1))
	m.offLevel = 1
	m.executeTurn()
	wantLog(t, m, "decides against")
}
