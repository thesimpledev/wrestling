package engine

import (
	"reflect"
	"testing"
)

func pinDefense() [3][6]DefenseOutcome {
	return allDefense(DefenseOutcome{Type: DefPIN})
}

func cageMatch(a, b *WrestlerCard, d Dice) *Match {
	m := singlesMatch(a, b, d)
	m.Type = MatchCage
	m.InitForMatchType()
	return m
}

type tagTeams struct {
	a1, a2, d1, d2 *WrestlerCard
}

func newTagTeams() tagTeams {
	return tagTeams{testCard("A1"), testCard("A2"), testCard("D1"), testCard("D2")}
}

func (tt tagTeams) match(d Dice, regular bool) *Match {
	m := NewTagMatch(tt.a1, tt.a2, tt.d1, tt.d2)
	m.SetDice(d)
	m.Sides[0].RegularPartners = regular
	m.Sides[1].RegularPartners = regular
	return m
}

func wantPIN(t *testing.T, ws *WrestlerState, want int) {
	t.Helper()
	if ws.CurrentPIN != want {
		t.Fatalf("%s PIN: got %d want %d", ws.Card.Name, ws.CurrentPIN, want)
	}
}

func wantResult(t *testing.T, m *Match, winner, method string) {
	t.Helper()
	if !m.Over() || m.Result() == nil {
		t.Fatalf("match not over, want %s by %s\n%s", winner, method, eventTexts(m))
	}
	if m.Result().Winner != winner || m.Result().Method != method {
		t.Fatalf("result: got %q by %q, want %q by %q\n%s",
			m.Result().Winner, m.Result().Method, winner, method, eventTexts(m))
	}
}

func wantDiceUsed(t *testing.T, d *scriptedDice) {
	t.Helper()
	if d.remaining() != 0 {
		t.Fatalf("dice left unrolled: %d", d.remaining())
	}
}

// ─── Cage and No DQ ─────────────────────────────────────────────────────────

func TestCageMatchNeverDisqualifies(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(taggedMove("Eye Gouge", 2, 1, TagDQ))
	att.DQ = 12
	dice := script(1, 1)
	m := cageMatch(att, def, dice)
	m.executeTurn()
	if m.Over() {
		t.Fatalf("cage match ended by DQ\n%s", eventTexts(m))
	}
	wantDiceUsed(t, dice)
	wantOffense(t, m, 0, 1)
}

func TestCageOutOfRingMoveIsFaceIntoCageWithLevelThreeDefense(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(chartMove("ring"))
	def.Defense = layeredDefense()
	m := cageMatch(att, def, script(1, 1))
	m.executeTurn()
	wantLog(t, m, "face-first into the cage")
	wantLog(t, m, "Defense Level 3")
	wantNoLog(t, m, "ring Chart")
	wantOffense(t, m, 0, 3)
}

func TestCageTurnbuckleRowDoesNotLeaveTheCage(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(chartMove("turnbuckle"))
	att.Defense = layeredDefense()
	def.Turnbuckle = RatingA
	m := cageMatch(att, def, script(1, 2, 2, 1))
	m.executeTurn()
	wantNoLog(t, m, "ring Chart")
	wantLog(t, m, "A (Defense Level 3")
	wantOffense(t, m, 1, 3)
}

func TestCageIgnoresLeavingTheRing(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	def.Ring = RatingA
	def.Defense = allDefense(DefenseOutcome{Type: DefDown, Power: 3, Tags: []MoveTag{TagLeave}})
	dice := script(1, 1)
	m := cageMatch(att, def, dice)
	m.executeTurn()
	wantNoLog(t, m, "rolls out of the ring")
	wantDiceUsed(t, dice)
	wantOffense(t, m, 0, 3)
}

func TestNoDQMatchIgnoresCountOutRow(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(chartMove("ring"))
	def.Ring = RatingC
	dice := script(1, 5, 6)
	m := singlesMatch(att, def, dice)
	m.Type = MatchNoDQ
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantPIN(t, m.Sides[1].Active(), 3)
	wantOffense(t, m, 0, 3)
}

func TestCountOutUsesPinRating(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(chartMove("ring"))
	def.Ring = RatingC

	beats := singlesMatch(att, def, script(1, 5, 6, 6, 6))
	beats.executeTurn()
	wantPIN(t, beats.Sides[1].Active(), 4)
	wantOffense(t, beats, 0, 3)

	counted := singlesMatch(att, def, script(1, 5, 6, 1, 2))
	counted.executeTurn()
	wantResult(t, counted, "A", "countout")
}

func TestBothRollDisqualification(t *testing.T) {
	cases := []struct {
		name       string
		throwerDQ  int
		victimDQ   int
		extraRolls []int
		check      func(t *testing.T, m *Match)
	}{
		{"only thrower out", 12, 2, nil, func(t *testing.T, m *Match) { wantResult(t, m, "D", "dq") }},
		{"only victim out", 2, 12, nil, func(t *testing.T, m *Match) { wantResult(t, m, "A", "dq") }},
		{"both out is a double DQ", 12, 12, nil, func(t *testing.T, m *Match) {
			if !m.Over() || !m.Result().Draw() || m.Result().Method != "double dq" {
				t.Fatalf("want double dq draw, got %+v", m.Result())
			}
		}},
		{"nobody out, even die: victim wins brawl", 2, 2, []int{2}, func(t *testing.T, m *Match) { wantOffense(t, m, 1, 3) }},
		{"nobody out, odd die: thrower wins brawl", 2, 2, []int{3}, func(t *testing.T, m *Match) { wantOffense(t, m, 0, 3) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			att, def := testCard("A"), testCard("D")
			att.Offense = allMoves(chartMove("ring"))
			att.DQ, def.DQ = tc.throwerDQ, tc.victimDQ
			rolls := append([]int{1, 2, 2, 3, 4, 3, 4}, tc.extraRolls...)
			dice := script(rolls...)
			m := singlesMatch(att, def, dice)
			m.executeTurn()
			wantDiceUsed(t, dice)
			tc.check(t, m)
		})
	}
}

// ─── Tag team ───────────────────────────────────────────────────────────────

func TestTagOutOnDefense(t *testing.T) {
	cases := []struct {
		name       string
		regular    bool
		tagRoll    [2]int
		wantActive int
		wantSide   int
	}{
		{"regular partners tag on 6", true, [2]int{3, 3}, 1, 1},
		{"regular partners miss on 7", true, [2]int{3, 4}, 0, 0},
		{"other teams miss on 6", false, [2]int{3, 3}, 0, 0},
		{"other teams tag on 4", false, [2]int{2, 2}, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			teams := newTagTeams()
			teams.d1.Defense = allDefense(DefenseOutcome{Type: DefDazed, Power: 1, Tags: []MoveTag{TagTagTeam}})
			dice := script(1, 1, tc.tagRoll[0], tc.tagRoll[1])
			m := teams.match(dice, tc.regular)
			m.executeTurn()
			wantDiceUsed(t, dice)
			if got := m.Sides[1].ActiveIndex; got != tc.wantActive {
				t.Fatalf("active wrestler index: got %d want %d\n%s", got, tc.wantActive, eventTexts(m))
			}
			wantOffense(t, m, tc.wantSide, 1)
		})
	}
}

func TestTagInstructionOnDefenseIgnoredInSingles(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	def.Defense = allDefense(DefenseOutcome{Type: DefDazed, Power: 1, Tags: []MoveTag{TagTagTeam}})
	dice := script(1, 1)
	m := singlesMatch(att, def, dice)
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantOffense(t, m, 0, 1)
}

func pinnedTagMatch(regular bool, rolls ...int) (*Match, *scriptedDice) {
	teams := newTagTeams()
	teams.d1.Defense = pinDefense()
	dice := script(append([]int{1, 1, 1, 1}, rolls...)...)
	m := teams.match(dice, regular)
	return m, dice
}

func TestPinSaveBreaksTheCountAndAddsFatigue(t *testing.T) {
	m, dice := pinnedTagMatch(true, 2, 3)
	m.executeTurn()
	wantDiceUsed(t, dice)
	if m.Over() {
		t.Fatalf("match ended despite a pin save\n%s", eventTexts(m))
	}
	wantPIN(t, m.Sides[1].Active(), 4)
	wantOffense(t, m, 0, 3)
}

func TestTeamsThatAreNotRegularPartnersGetNoPinSave(t *testing.T) {
	m, dice := pinnedTagMatch(false)
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantResult(t, m, "A1", "pinfall")
}

func TestPinSaveLimitIsTwoPerTeam(t *testing.T) {
	m, dice := pinnedTagMatch(true)
	m.Sides[1].PinSavesUsed = 2
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantResult(t, m, "A1", "pinfall")
}

func TestPinSaveFailedLetsThePinStand(t *testing.T) {
	m, dice := pinnedTagMatch(true, 4, 4)
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantResult(t, m, "A1", "pinfall")
}

func TestPinSaveBrawl(t *testing.T) {
	doubleDQ, dice := pinnedTagMatch(true, 5, 6, 1, 3)
	doubleDQ.executeTurn()
	wantDiceUsed(t, dice)
	if !doubleDQ.Over() || !doubleDQ.Result().Draw() {
		t.Fatalf("want double DQ, got %+v\n%s", doubleDQ.Result(), eventTexts(doubleDQ))
	}

	pinnedTeamWins, dice := pinnedTagMatch(true, 5, 6, 3, 3, 2)
	pinnedTeamWins.executeTurn()
	wantDiceUsed(t, dice)
	wantPIN(t, pinnedTeamWins.Sides[1].Active(), 4)
	wantOffense(t, pinnedTeamWins, 1, 3)

	otherTeamWins, _ := pinnedTagMatch(true, 5, 6, 3, 3, 5)
	otherTeamWins.executeTurn()
	wantOffense(t, otherTeamWins, 0, 3)
}

func TestPinSaveReversal(t *testing.T) {
	kickOut, dice := pinnedTagMatch(true, 6, 6, 6, 6)
	kickOut.executeTurn()
	wantDiceUsed(t, dice)
	wantPIN(t, kickOut.Sides[1].Active(), 4)
	wantPIN(t, kickOut.Sides[0].Active(), 4)
	wantOffense(t, kickOut, 1, 3)

	pinned, _ := pinnedTagMatch(true, 6, 6, 1, 1)
	pinned.executeTurn()
	wantResult(t, pinned, "D1", "pinfall")
}

func TestPinSaveInterferenceRollsTheInterferenceChart(t *testing.T) {
	m, dice := pinnedTagMatch(true, 1, 2, 4, 4)
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantLog(t, m, "Interference Chart")
	wantPIN(t, m.Sides[1].Active(), 4)
	wantOffense(t, m, 0, 3)
}

func TestFinisherPinCanBeSavedInTagMatch(t *testing.T) {
	teams := newTagTeams()
	teams.a1.Offense = allMoves(plainMove("TEST FINISH", 3, 3))
	dice := script(1, 1, 1, 2, 3)
	m := teams.match(dice, true)
	m.offLevel = 2
	m.executeTurn()
	wantDiceUsed(t, dice)
	if m.Over() {
		t.Fatalf("finisher pin was not saved\n%s", eventTexts(m))
	}
	wantPIN(t, m.Sides[1].Active(), 4)
	wantOffense(t, m, 0, 3)
}

// ─── Allies ─────────────────────────────────────────────────────────────────

func TestDistractionUsesTheAllysRating(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	def.Defense = pinDefense()
	def.PINAdv = 4
	def.Distractor = 1
	ally := testCard("Ally")
	ally.Distractor = 12

	dice := script(1, 1, 6, 6)
	m := singlesMatch(att, def, dice)
	m.Sides[1].Ally = ally
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantLog(t, m, "needs 12 or lower")
	wantPIN(t, m.Sides[1].Active(), 5)
	wantOffense(t, m, 0, 3)
}

func interferenceMatch(rolls ...int) (*Match, *scriptedDice) {
	att, def := testCard("A"), testCard("D")
	def.Defense = pinDefense()
	def.PINAdv = 6
	dice := script(append([]int{1, 1}, rolls...)...)
	m := singlesMatch(att, def, dice)
	m.Sides[1].Ally = testCard("Ally")
	return m, dice
}

func TestInterferenceChartRows(t *testing.T) {
	cases := []struct {
		name  string
		rolls []int
		check func(t *testing.T, m *Match)
	}{
		{"2-3 double team, pin kicked out", []int{1, 1, 5, 5, 6, 6}, func(t *testing.T, m *Match) {
			wantPIN(t, m.Sides[0].Active(), 5)
			wantPIN(t, m.Sides[1].Active(), 7)
			wantOffense(t, m, 1, 3)
		}},
		{"2-3 double team, disqualified on 8", []int{1, 2, 4, 4}, func(t *testing.T, m *Match) {
			wantResult(t, m, "A", "dq")
		}},
		{"4 finisher pin wins", []int{1, 3, 4, 4, 1, 1}, func(t *testing.T, m *Match) {
			wantLog(t, m, "needed 6")
			wantResult(t, m, "D", "pinfall")
		}},
		{"5 disqualified on 6", []int{2, 3, 3, 3}, func(t *testing.T, m *Match) {
			wantResult(t, m, "A", "dq")
		}},
		{"5 pin kicked out", []int{2, 3, 3, 4, 6, 6}, func(t *testing.T, m *Match) {
			wantPIN(t, m.Sides[0].Active(), 4)
			wantOffense(t, m, 1, 3)
		}},
		{"6 safe on 6, level 3 offense", []int{3, 3, 3, 3}, func(t *testing.T, m *Match) {
			wantPIN(t, m.Sides[1].Active(), 7)
			wantOffense(t, m, 1, 3)
		}},
		{"7 brawl even: you take over", []int{3, 4, 3, 3, 2}, func(t *testing.T, m *Match) {
			wantOffense(t, m, 1, 3)
		}},
		{"7 brawl odd: opponent takes over", []int{3, 4, 3, 3, 3}, func(t *testing.T, m *Match) {
			wantOffense(t, m, 0, 3)
		}},
		{"8-9 distraction breaks the count", []int{4, 5}, func(t *testing.T, m *Match) {
			wantPIN(t, m.Sides[1].Active(), 7)
			wantOffense(t, m, 0, 3)
		}},
		{"10 roll your pin, kicked out", []int{4, 6, 6, 6}, func(t *testing.T, m *Match) {
			wantPIN(t, m.Sides[1].Active(), 7)
			wantOffense(t, m, 0, 3)
		}},
		{"11-12 pin plus opponent's finisher", []int{6, 6, 4, 4}, func(t *testing.T, m *Match) {
			wantLog(t, m, "needed 9")
			wantResult(t, m, "A", "pinfall")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, dice := interferenceMatch(tc.rolls...)
			m.executeTurn()
			wantDiceUsed(t, dice)
			wantLog(t, m, "Interference Chart")
			tc.check(t, m)
		})
	}
}

func TestInterferenceOnlyOncePerMatch(t *testing.T) {
	m, _ := interferenceMatch(4, 5, 1, 1, 6, 6, 6, 6)
	m.Sides[1].Ally.Distractor = 2
	m.executeTurn()
	m.executeTurn()
	count := 0
	for _, e := range m.Events {
		if e.Type == EventInterference && e.Text == "D calls for outside interference!" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("interference called %d times, want 1\n%s", count, eventTexts(m))
	}
}

// ─── Feud table ─────────────────────────────────────────────────────────────

func feudMatch(rolls ...int) (*Match, *scriptedDice) {
	att, def := testCard("A"), testCard("D")
	def.Defense = pinDefense()
	dice := script(append([]int{1, 1}, rolls...)...)
	m := singlesMatch(att, def, dice)
	m.IsFeud = true
	return m, dice
}

func wantInjury(t *testing.T, m *Match, name string, cards int) {
	t.Helper()
	r := m.Result()
	if r.InjuredWrestler != name || r.InjuryCards != cards {
		t.Fatalf("injury: got %q for %d, want %q for %d\n%s",
			r.InjuredWrestler, r.InjuryCards, name, cards, eventTexts(m))
	}
}

func TestFeudTableOnlyWhenTheEndingRollIsDoubles(t *testing.T) {
	doubles, dice := feudMatch(1, 1, 1, 2)
	doubles.executeTurn()
	wantDiceUsed(t, dice)
	wantLog(t, doubles, "Feud Table")
	wantInjury(t, doubles, "A", 2)

	plain, dice := feudMatch(1, 2)
	plain.executeTurn()
	wantDiceUsed(t, dice)
	wantNoLog(t, plain, "Feud Table")
	wantInjury(t, plain, "", 0)
}

func TestNoFeudTableOutsideFeudMatches(t *testing.T) {
	m, dice := feudMatch(1, 1)
	m.IsFeud = false
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantNoLog(t, m, "Feud Table")
}

func TestFeudTableTreatsTheDisqualifiedWrestlerAsYou(t *testing.T) {
	att, def := testCard("A"), testCard("D")
	att.Offense = allMoves(taggedMove("Eye Gouge", 2, 1, TagDQ))
	att.DQ = 12
	dice := script(1, 3, 3, 3, 4)
	m := singlesMatch(att, def, dice)
	m.IsFeud = true
	m.Rules.AvoidDisMoves = false
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantResult(t, m, "D", "dq")
	wantInjury(t, m, "D", 1)
}

func TestFeudRowsNeedingAnAbsentAllyAreRerolled(t *testing.T) {
	cases := []struct {
		name       string
		yourAlly   bool
		theirAlly  bool
		feudRolls  []int
		wantName   string
		wantCards  int
		wantInText string
	}{
		{"5-6 without your ally rerolls", false, false, []int{2, 3, 1, 2}, "A", 2, ""},
		{"5-6 with your ally stands", true, false, []int{2, 3}, "", 0, "double-team"},
		{"8-9 needs both allies", true, false, []int{4, 4, 3, 4}, "D", 1, ""},
		{"8-9 with both allies stands", true, true, []int{4, 4}, "", 0, "four"},
		{"10 without their ally rerolls", true, false, []int{4, 6, 1, 1}, "A", 2, ""},
		{"10 with their ally stands", false, true, []int{4, 6}, "A", 2, ""},
		{"7 needs no ally", false, false, []int{3, 4}, "D", 1, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, dice := feudMatch(append([]int{1, 1}, tc.feudRolls...)...)
			if tc.yourAlly {
				m.Sides[0].Ally = testCard("Friend")
			}
			if tc.theirAlly {
				m.Sides[1].Ally = testCard("Enemy")
			}
			m.executeTurn()
			wantDiceUsed(t, dice)
			wantInjury(t, m, tc.wantName, tc.wantCards)
			if tc.wantInText != "" {
				wantLog(t, m, tc.wantInText)
			}
		})
	}
}

func TestFeudGangAttackInjuresOpponentAndSuspendsYouAndAlly(t *testing.T) {
	m, dice := feudMatch(1, 1, 6, 6, 4, 2)
	m.Sides[0].Ally = testCard("Friend")
	m.executeTurn()
	wantDiceUsed(t, dice)
	wantInjury(t, m, "D", 4)
	r := m.Result()
	if !reflect.DeepEqual(r.SuspendedWrestlers, []string{"A", "Friend"}) || r.SuspensionCards != 2 {
		t.Fatalf("suspension: got %v for %d, want [A Friend] for 2", r.SuspendedWrestlers, r.SuspensionCards)
	}
}
