package engine

import "testing"

type chartRow struct {
	min, max int
	kind     ChartOutcomeType
	level    int
}

func checkChart(t *testing.T, name string, chart ChartTable, rating Rating, rows []chartRow) {
	t.Helper()
	covered := map[int]bool{}
	for _, row := range rows {
		for roll := row.min; roll <= row.max; roll++ {
			covered[roll] = true
			got := chart.Lookup(rating, roll)
			if got == nil {
				t.Errorf("%s rating %s roll %d: no outcome", name, rating, roll)
				continue
			}
			if got.Type != row.kind || got.Level != row.level {
				t.Errorf("%s rating %s roll %d: got type %d level %d, want type %d level %d",
					name, rating, roll, got.Type, got.Level, row.kind, row.level)
			}
		}
	}
	for roll := 2; roll <= 12; roll++ {
		if !covered[roll] {
			t.Errorf("%s rating %s: test table does not cover roll %d", name, rating, roll)
		}
	}
}

func TestIntoTheRopesChart(t *testing.T) {
	checkChart(t, "ropes", RopesChart, RatingA, []chartRow{
		{2, 3, ChartRollPIN, 0}, {4, 5, ChartRollOnOffense, 3}, {6, 6, ChartRollOnOffense, 2},
		{7, 7, ChartRollAgain, 0}, {8, 9, ChartOppRollOnOffense, 2}, {10, 10, ChartPowerCheck, 0},
		{11, 11, ChartBetterRating, 0}, {12, 12, ChartOppRollOnOffense, 3},
	})
	checkChart(t, "ropes", RopesChart, RatingB, []chartRow{
		{2, 2, ChartRollPIN, 0}, {3, 4, ChartRollOnOffense, 3}, {5, 5, ChartRollOnOffense, 2},
		{6, 6, ChartRollAgain, 0}, {7, 8, ChartOppRollOnOffense, 2}, {9, 9, ChartPowerCheck, 0},
		{10, 10, ChartBetterRating, 0}, {11, 12, ChartOppRollOnOffense, 3},
	})
	checkChart(t, "ropes", RopesChart, RatingC, []chartRow{
		{2, 3, ChartRollOnOffense, 3}, {4, 4, ChartRollOnOffense, 2}, {5, 5, ChartRollAgain, 0},
		{6, 7, ChartOppRollOnOffense, 2}, {8, 8, ChartPowerCheck, 0}, {9, 9, ChartBetterRating, 0},
		{10, 12, ChartOppRollOnOffense, 3},
	})
}

func TestIntoTheTurnbuckleChart(t *testing.T) {
	checkChart(t, "turnbuckle", TurnbuckleChart, RatingA, []chartRow{
		{2, 3, ChartRollPIN, 0}, {4, 4, ChartOppRollOnChart, 0}, {5, 5, ChartRollOnDefense, 3},
		{6, 6, ChartRollOnOffense, 2}, {7, 7, ChartOppRollOnChart, 0}, {8, 10, ChartOppRollOnOffense, 2},
		{11, 11, ChartBetterRating, 0}, {12, 12, ChartOppRollOnOffense, 3},
	})
	checkChart(t, "turnbuckle", TurnbuckleChart, RatingB, []chartRow{
		{2, 2, ChartRollPIN, 0}, {3, 3, ChartOppRollOnChart, 0}, {4, 4, ChartRollOnDefense, 3},
		{5, 5, ChartRollOnOffense, 2}, {6, 6, ChartOppRollOnChart, 0}, {7, 9, ChartOppRollOnOffense, 2},
		{10, 10, ChartBetterRating, 0}, {11, 12, ChartOppRollOnOffense, 3},
	})
	checkChart(t, "turnbuckle", TurnbuckleChart, RatingC, []chartRow{
		{2, 2, ChartOppRollOnChart, 0}, {3, 3, ChartRollOnDefense, 3}, {4, 4, ChartRollOnOffense, 2},
		{5, 5, ChartOppRollOnChart, 0}, {6, 8, ChartOppRollOnOffense, 2}, {9, 9, ChartBetterRating, 0},
		{10, 12, ChartOppRollOnOffense, 3},
	})
}

func TestTurnbuckleRedirectsGoToTheRightChart(t *testing.T) {
	cases := []struct {
		rating Rating
		roll   int
		chart  string
	}{
		{RatingA, 4, "ring"}, {RatingA, 7, "turnbuckle"},
		{RatingB, 3, "ring"}, {RatingB, 6, "turnbuckle"},
		{RatingC, 2, "ring"}, {RatingC, 5, "turnbuckle"},
	}
	for _, tc := range cases {
		if got := TurnbuckleChart.Lookup(tc.rating, tc.roll).ChartRef; got != tc.chart {
			t.Errorf("turnbuckle rating %s roll %d: redirect to %q, want %q", tc.rating, tc.roll, got, tc.chart)
		}
	}
}

func TestOutOfTheRingChart(t *testing.T) {
	checkChart(t, "ring", OutOfRingChart, RatingA, []chartRow{
		{2, 4, ChartRollOnOffense, 3}, {5, 5, ChartBothRollDQ, 0}, {6, 6, ChartRollDQ, 0},
		{7, 7, ChartBetterRating, 0}, {8, 9, ChartOppRollOnOffense, 3}, {10, 11, ChartOppRollDQ, 0},
		{12, 12, ChartRollCountOut, 0},
	})
	checkChart(t, "ring", OutOfRingChart, RatingB, []chartRow{
		{2, 3, ChartRollOnOffense, 3}, {4, 4, ChartBothRollDQ, 0}, {5, 5, ChartRollDQ, 0},
		{6, 6, ChartBetterRating, 0}, {7, 9, ChartOppRollOnOffense, 3}, {10, 10, ChartOppRollDQ, 0},
		{11, 12, ChartRollCountOut, 0},
	})
	checkChart(t, "ring", OutOfRingChart, RatingC, []chartRow{
		{2, 2, ChartRollOnOffense, 3}, {3, 3, ChartBothRollDQ, 0}, {4, 4, ChartRollDQ, 0},
		{5, 5, ChartBetterRating, 0}, {6, 9, ChartOppRollOnOffense, 3}, {10, 10, ChartOppRollDQ, 0},
		{11, 12, ChartRollCountOut, 0},
	})
}

func TestDeathjumpChart(t *testing.T) {
	checkChart(t, "deathjump", DeathjumpChart, RatingA, []chartRow{
		{2, 2, ChartRefDown, 0}, {3, 4, ChartRollPIN, 0}, {5, 6, ChartRollOnOffense, 3},
		{7, 9, ChartOppRollOnOffense, 3}, {10, 11, ChartAgilityCheck, 0}, {12, 12, ChartRollYourPIN, 0},
	})
	checkChart(t, "deathjump", DeathjumpChart, RatingB, []chartRow{
		{2, 2, ChartRefDown, 0}, {3, 3, ChartRollPIN, 0}, {4, 5, ChartRollOnOffense, 3},
		{6, 9, ChartOppRollOnOffense, 3}, {10, 10, ChartAgilityCheck, 0}, {11, 12, ChartRollYourPIN, 0},
	})
	checkChart(t, "deathjump", DeathjumpChart, RatingC, []chartRow{
		{2, 2, ChartRollPIN, 0}, {3, 4, ChartRollOnOffense, 3}, {5, 8, ChartOppRollOnOffense, 3},
		{9, 9, ChartAgilityCheck, 0}, {10, 12, ChartRollYourPIN, 0},
	})
}

func TestPinSavesChart(t *testing.T) {
	want := map[int]PinSaveOutcomeType{
		2: PinSaveInterference, 3: PinSaveInterference,
		4: PinSaveSaved, 5: PinSaveSaved, 6: PinSaveSaved,
		7: PinSaveFailed, 8: PinSaveFailed, 9: PinSaveFailed, 10: PinSaveFailed,
		11: PinSaveBrawl, 12: PinSaveReversed,
	}
	for roll := 2; roll <= 12; roll++ {
		got := LookupPinSave(roll)
		if got == nil || got.Type != want[roll] {
			t.Errorf("pin save roll %d: got %+v, want type %d", roll, got, want[roll])
		}
	}
}

func TestOutsideInterferenceChart(t *testing.T) {
	type row struct {
		kind InterferenceOutcomeType
		dq   int
	}
	want := map[int]row{
		2: {InterfDoubleTeam, 8}, 3: {InterfDoubleTeam, 8}, 4: {InterfFinisher, 7},
		5: {InterfAttackAndPin, 6}, 6: {InterfAttackAndL3, 5}, 7: {InterfBrawl, 4},
		8: {InterfDistract, 0}, 9: {InterfDistract, 0}, 10: {InterfBackfire, 0},
		11: {InterfBackfireFinish, 0}, 12: {InterfBackfireFinish, 0},
	}
	for roll := 2; roll <= 12; roll++ {
		got := LookupInterference(roll)
		if got == nil || got.Type != want[roll].kind || got.DQThreshold != want[roll].dq {
			t.Errorf("interference roll %d: got %+v, want type %d dq %d", roll, got, want[roll].kind, want[roll].dq)
		}
	}
}

func TestFeudTable(t *testing.T) {
	type row struct {
		kind             FeudOutcomeType
		injury           int
		yourAlly, theirs bool
	}
	want := map[int]row{
		2: {FeudAttackedByLoser, 2, false, false}, 3: {FeudAttackedByLoser, 2, false, false},
		4: {FeudAttackedByLoser, 2, false, false}, 5: {FeudAllyDoubleTeam, 0, true, false},
		6: {FeudAllyDoubleTeam, 0, true, false}, 7: {FeudPostMatchAttack, 1, false, false},
		8: {FeudFourManBrawl, 0, true, true}, 9: {FeudFourManBrawl, 0, true, true},
		10: {FeudOpponentAlly, 2, false, true}, 11: {FeudGangAttack, 0, true, false},
		12: {FeudGangAttack, 0, true, false},
	}
	for roll := 2; roll <= 12; roll++ {
		got := LookupFeud(roll)
		w := want[roll]
		if got == nil || got.Type != w.kind || got.InjuryDays != w.injury ||
			got.NeedsYourAlly != w.yourAlly || got.NeedsOpponentAlly != w.theirs {
			t.Errorf("feud roll %d: got %+v, want %+v", roll, got, w)
		}
	}
}

func TestChoiceSituationsChart(t *testing.T) {
	type option struct {
		name      string
		power     int
		threshold int
		stat      string
		chart     string
	}
	want := map[string][2]option{
		"A": {{"Into the Ropes", 0, 0, "", "ropes"}, {"Belly to Belly Suplex", 2, 8, "pw", ""}},
		"B": {{"Standing Dropkick", 2, 8, "ag", ""}, {"Into the Turnbuckle", 0, 0, "", "turnbuckle"}},
		"C": {{"Moonsault", 3, 7, "ag", ""}, {"Kick to Knee", 2, 7, "pw", ""}},
		"D": {{"Kick to Face", 2, 9, "ag", ""}, {"Cobra Clutch Suplex", 3, 9, "pw", ""}},
		"E": {{"Scorpion Death Lock", 3, 9, "ag", ""}, {"Power Slam", 2, 9, "pw", ""}},
		"F": {{"Leg Drop", 2, 7, "ag", ""}, {"Running Lariat", 3, 7, "pw", ""}},
		"G": {{"Deathjump", 0, 0, "", "deathjump"}, {"Tombstone Piledriver", 3, 8, "pw", ""}},
		"H": {{"Flying Elbow Drop", 3, 8, "ag", ""}, {"Deathjump", 0, 0, "", "deathjump"}},
	}
	if len(ChoiceSituations) != len(want) {
		t.Fatalf("choice situations: got %d, want %d", len(ChoiceSituations), len(want))
	}
	for key, pair := range want {
		got := ChoiceSituations[key]
		for i, opt := range []ChoiceOption{got.Option1, got.Option2} {
			w := pair[i]
			if opt.Name != w.name || opt.Power != w.power || opt.Threshold != w.threshold ||
				opt.StatType != w.stat || opt.ChartRef != w.chart || opt.IsChart != (w.chart != "") {
				t.Errorf("choice %s option %d: got %+v, want %+v", key, i+1, opt, w)
			}
		}
	}
}
