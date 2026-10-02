package engine

import (
	"strings"
	"testing"
)

func taggedMove(name string, power, defLevel int, tags ...MoveTag) Move {
	return Move{Name: name, Power: power, DefLevel: defLevel, Tags: tags}
}

func chartMove(chart string) Move {
	return Move{Name: "Chart Move", Power: 2, DefLevel: 2, Tags: []MoveTag{TagChart}, ChartType: chart}
}

func choiceMove(key string) Move {
	return Move{Name: "Choice", Power: 2, DefLevel: 2, Tags: []MoveTag{TagChoice}, ChoiceKey: key}
}

func layeredDefense() [3][6]DefenseOutcome {
	var grid [3][6]DefenseOutcome
	for slot := 0; slot < 6; slot++ {
		grid[0][slot] = DefenseOutcome{Type: DefDazed, Power: 1}
		grid[1][slot] = DefenseOutcome{Type: DefHurt, Power: 2}
		grid[2][slot] = DefenseOutcome{Type: DefDown, Power: 3}
	}
	return grid
}

func lastMoveText(m *Match) string {
	for i := len(m.Events) - 1; i >= 0; i-- {
		if m.Events[i].Type == EventMove && m.Events[i].Attacker != "" {
			return m.Events[i].Text
		}
	}
	return ""
}

func TestOffenseLevelFollowsDefenseNumber(t *testing.T) {
	cases := []struct {
		name       string
		startLevel int
		outcome    DefenseOutcome
		wantLevel  int
	}{
		{"dazed 1 drops level 3 to 1", 3, DefenseOutcome{Type: DefDazed, Power: 1}, 1},
		{"dazed 1 keeps level 1", 1, DefenseOutcome{Type: DefDazed, Power: 1}, 1},
		{"hurt 2 drops level 3 to 2", 3, DefenseOutcome{Type: DefHurt, Power: 2}, 2},
		{"hurt 2 raises level 1 to 2", 1, DefenseOutcome{Type: DefHurt, Power: 2}, 2},
		{"down 3 raises level 1 to 3", 1, DefenseOutcome{Type: DefDown, Power: 3}, 3},
		{"down 2 drops level 3 to 2", 3, DefenseOutcome{Type: DefDown, Power: 2}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			att, def := testCard("A"), testCard("D")
			def.Defense = allDefense(tc.outcome)
			m := singlesMatch(att, def, script(1, 1))
			m.offLevel = tc.startLevel - 1
			m.executeTurn()
			wantOffense(t, m, 0, tc.wantLevel)
		})
	}
}

func TestReversalGivesDefenderTheLevelAfterHisName(t *testing.T) {
	for level := 1; level <= 3; level++ {
		att, def := testCard("A"), testCard("D")
		def.Defense = allDefense(DefenseOutcome{Type: DefReversal, Power: level})
		m := singlesMatch(att, def, script(1, 1))
		m.executeTurn()
		wantOffense(t, m, 1, level)
	}
}

func TestKickOutKeepsAttackerOnLevelThree(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	def.Defense = allDefense(DefenseOutcome{Type: DefPIN})
	m := singlesMatch(att, def, script(1, 1, 6, 6))
	m.executeTurn()
	wantOffense(t, m, 0, 3)
	if got := m.Sides[1].Active().CurrentPIN; got != 4 {
		t.Fatalf("PIN after kick-out: got %d want 4", got)
	}
	if m.Over() {
		t.Fatal("match ended on a kick-out")
	}
}

func TestPinAtOrUnderRatingEndsMatch(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	def.Defense = allDefense(DefenseOutcome{Type: DefPIN})
	m := singlesMatch(att, def, script(1, 1, 1, 2))
	m.executeTurn()
	if !m.Over() || m.Result().Winner != "A" || m.Result().Method != "pinfall" {
		t.Fatalf("result: %+v", m.Result())
	}
}

func TestFinisherAddsRatingAndKickOutKeepsAttacker(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(plainMove("TEST FINISH", 3, 3))

	pinned := singlesMatch(att, def, script(1, 2, 3))
	pinned.offLevel = 2
	pinned.executeTurn()
	if !pinned.Over() || pinned.Result().Winner != "A" {
		t.Fatalf("roll 5 against PIN 3 plus finisher 2 should pin: %+v", pinned.Result())
	}

	kicked := singlesMatch(att, def, script(1, 3, 3))
	kicked.offLevel = 2
	kicked.executeTurn()
	wantOffense(t, kicked, 0, 3)
	if got := kicked.Sides[1].Active().CurrentPIN; got != 4 {
		t.Fatalf("PIN after finisher kick-out: got %d want 4", got)
	}
}

func TestCapitalMoveBelowLevelThreeIsNotAFinisher(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(plainMove("DDT", 2, 1))
	for _, level := range []int{1, 2} {
		m := singlesMatch(att, def, script(1, 1))
		m.offLevel = level - 1
		m.executeTurn()
		for _, e := range m.Events {
			if e.Type == EventFinisher {
				t.Fatalf("level %d: finisher triggered by %q", level, e.Text)
			}
		}
		wantOffense(t, m, 0, 1)
	}
}

func TestRollFinisherUsesTheDieAsItsRating(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Finisher = Finisher{Name: "RISKY FINISH", Rating: 0, IsRoll: true, RollMin: 2, RollMax: 6}
	att.Offense = allMoves(plainMove("RISKY FINISH", 3, 3))

	pinned := singlesMatch(att, def, script(1, 4, 3, 4))
	pinned.offLevel = 2
	pinned.executeTurn()
	if !pinned.Over() {
		t.Fatalf("roll 7 against PIN 3 plus rolled rating 4 should pin\n%s", eventTexts(pinned))
	}

	kicked := singlesMatch(att, def, script(1, 4, 4, 4))
	kicked.offLevel = 2
	kicked.executeTurn()
	wantOffense(t, kicked, 0, 3)

	missed := singlesMatch(att, def, script(1, 1))
	missed.offLevel = 2
	missed.executeTurn()
	wantOffense(t, missed, 1, 3)
}

func TestAgilityAndPowerLowerIsBetter(t *testing.T) {
	cases := []struct {
		name         string
		tag          MoveTag
		attacker     int
		defender     int
		wantSide     int
		wantLevel    int
		defenseRolls []int
	}{
		{"ag: opponent better counters", TagAgility, 0, -3, 1, 2, nil},
		{"ag: opponent worse, move works", TagAgility, 0, 3, 0, 1, []int{1}},
		{"ag: same rating, move works", TagAgility, 0, 0, 0, 1, []int{1}},
		{"pw: opponent better counters", TagPower, -2, -3, 1, 2, nil},
		{"pw: opponent worse, move works", TagPower, -2, 0, 0, 1, []int{1}},
		{"pw: same rating, move works", TagPower, 2, 2, 0, 1, []int{1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			att, def := testCard("A"), testCard("D")
			att.Offense = allMoves(taggedMove("Special", 2, 1, tc.tag))
			att.Agility, att.Power = tc.attacker, tc.attacker
			def.Agility, def.Power = tc.defender, tc.defender
			rolls := append([]int{1}, tc.defenseRolls...)
			m := singlesMatch(att, def, script(rolls...))
			m.executeTurn()
			wantOffense(t, m, tc.wantSide, tc.wantLevel)
		})
	}
}

func TestRopesPowerRowUsesLowerIsBetter(t *testing.T) {
	cases := []struct {
		name        string
		throwerPow  int
		victimPower int
		wantSide    int
	}{
		{"victim better power wins", 0, -2, 1},
		{"same power goes to thrower", 0, 0, 0},
		{"victim worse power loses", 0, 2, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			att, def := testCard("A"), testCard("D")
			att.Offense = allMoves(chartMove("ropes"))
			att.Power, def.Power = tc.throwerPow, tc.victimPower
			m := singlesMatch(att, def, script(1, 4, 5))
			m.executeTurn()
			wantOffense(t, m, tc.wantSide, 2)
		})
	}
}

func TestDeathjumpStruggleUsesLowerIsBetter(t *testing.T) {
	cases := []struct {
		name     string
		victimAg int
		wantSide int
	}{
		{"victim better agility wins", -1, 1},
		{"same agility goes to jumper", 0, 0},
		{"victim worse agility loses", 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			att, def := testCard("A"), testCard("D")
			att.Offense = allMoves(chartMove("deathjump"))
			def.Agility = tc.victimAg
			m := singlesMatch(att, def, script(1, 5, 5))
			m.executeTurn()
			wantOffense(t, m, tc.wantSide, 3)
		})
	}
}

func TestChoiceThresholdAddsOpponentRating(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(choiceMove("C"))
	def.Agility, def.Power = -2, 1
	def.Defense = layeredDefense()

	works := singlesMatch(att, def, script(1, 4, 4, 1))
	works.executeTurn()
	wantLog(t, works, "Kick to Knee")
	wantLog(t, works, "Defense Level 2")
	wantOffense(t, works, 0, 2)

	fails := singlesMatch(att, def, script(1, 4, 5))
	fails.executeTurn()
	wantOffense(t, fails, 1, 2)
}

func TestChoicePrefersTheMoveMoreLikelyToWork(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(choiceMove("C"))
	def.Agility, def.Power = 2, -2
	def.Defense = layeredDefense()
	m := singlesMatch(att, def, script(1, 4, 5, 1))
	m.executeTurn()
	wantLog(t, m, "Moonsault")
	wantLog(t, m, "Defense Level 3")
	wantOffense(t, m, 0, 3)
}

func TestTagOnlyMoveIsRerolledInSingles(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	for slot := 0; slot < 3; slot++ {
		att.Offense[0][slot] = taggedMove("Double Team", 2, 1, TagTagTeam)
		att.Offense[0][slot+3] = plainMove("Solo Hit", 1, 1)
	}
	dice := script(1, 5, 1)
	m := singlesMatch(att, def, dice)
	m.executeTurn()
	if got := lastMoveText(m); !strings.Contains(got, "Solo Hit") {
		t.Fatalf("move performed: %q, want Solo Hit", got)
	}
	if dice.remaining() != 0 {
		t.Fatalf("dice left unrolled: %d", dice.remaining())
	}
}

func TestSinglesOnlyMoveIsRerolledInTagMatch(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	for slot := 0; slot < 3; slot++ {
		att.Offense[0][slot] = taggedMove("Solo Special", 2, 1, TagSingles)
		att.Offense[0][slot+3] = plainMove("Team Hit", 1, 1)
	}
	dice := script(2, 6, 1)
	m := NewTagMatch(att, testCard("A2"), def, testCard("D2"))
	m.SetDice(dice)
	m.executeTurn()
	if got := lastMoveText(m); !strings.Contains(got, "Team Hit") {
		t.Fatalf("move performed: %q, want Team Hit", got)
	}
}

func TestLevelWithNoUsableMoveStillPlays(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(taggedMove("Double Team", 2, 1, TagTagTeam))
	dice := script(3, 1)
	m := singlesMatch(att, def, dice)
	m.executeTurn()
	if dice.remaining() != 0 {
		t.Fatalf("dice left unrolled: %d", dice.remaining())
	}
}

func TestLeavingTheRingUsesTheLeaversOwnRating(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Ring, def.Ring = RatingC, RatingA
	def.Defense = allDefense(DefenseOutcome{Type: DefDown, Power: 3, Tags: []MoveTag{TagLeave}})
	m := singlesMatch(att, def, script(1, 1, 2, 2))
	m.executeTurn()
	wantLog(t, m, "Rating A")
	wantOffense(t, m, 1, 3)
}

func TestRopesPinKickOutKeepsThePinnerOnOffense(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(chartMove("ropes"))
	def.Ropes = RatingA
	m := singlesMatch(att, def, script(1, 1, 1, 6, 6))
	m.executeTurn()
	wantOffense(t, m, 1, 3)
	if got := m.Sides[0].Active().CurrentPIN; got != 4 {
		t.Fatalf("thrower PIN after kick-out: got %d want 4", got)
	}
}

func TestRefereeMissesExactlyTheMovesRolled(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	m := singlesMatch(att, def, script(1, 1, 1, 1, 1, 1))
	m.refDown = true
	m.refDownTurns = 2
	for turn := 1; turn <= 3; turn++ {
		m.executeTurn()
		recovered := strings.Contains(eventTexts(m), "referee recovers")
		if turn <= 2 && recovered {
			t.Fatalf("referee back on move %d, should miss 2 moves", turn)
		}
		if turn == 3 && !recovered {
			t.Fatal("referee still down for the third move")
		}
	}
}
