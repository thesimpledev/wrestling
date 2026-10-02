package ui

import (
	"strings"
	"testing"

	"wrestling/internal/engine"
)

type fixedDice struct {
	rolls []int
	next  int
}

func (d *fixedDice) Roll() int {
	r := d.rolls[d.next%len(d.rolls)]
	d.next++
	return r
}

// pinCard is a wrestler who is pinned by the first move that lands.
func pinCard(name string) *engine.WrestlerCard {
	card := fakeCard(name)
	card.PINAdv = 12
	for lvl := range card.Defense {
		for slot := range card.Defense[lvl] {
			card.Defense[lvl][slot] = engine.DefenseOutcome{Type: engine.DefPIN}
		}
	}
	return card
}

func TestEndFightCardCountsDownThenRecordsWhatHappenedOnTheCard(t *testing.T) {
	g, _ := testGame("A", "B", "C")
	g.Injuries.RecordInjury("A", 2)
	g.Injuries.RecordSuspension("C", 1)

	g.EndFightCard([]*engine.MatchResult{{
		Winner: "A", Loser: "B",
		InjuredWrestler: "B", InjuryCards: 2,
		SuspendedWrestlers: []string{"A"}, SuspensionCards: 3,
	}})

	if got := g.Injuries.InjuryCards("A"); got != 1 {
		t.Errorf("existing injury: %d cards left, want 1", got)
	}
	if got := g.Injuries.InjuryCards("B"); got != 2 {
		t.Errorf("new injury: %d cards, want the full 2", got)
	}
	if g.Injuries.IsSuspended("C") {
		t.Error("one-card suspension should be over")
	}
	if got := g.Injuries.SuspensionCards("A"); got != 3 {
		t.Errorf("new suspension: %d cards, want the full 3", got)
	}
	if saved := g.Store.(*memStore).injuries; !strings.Contains(string(saved), `"B"`) {
		t.Errorf("injuries not saved: %s", saved)
	}
}

func TestEndFightCardSkipsMissingResults(t *testing.T) {
	g, _ := testGame("A", "B")
	g.Injuries.RecordInjury("A", 1)
	g.EndFightCard([]*engine.MatchResult{nil})
	if g.Injuries.IsInjured("A") {
		t.Fatal("one-card injury should be over")
	}
}

func TestFeudInjuryKeepsItsFullLengthAfterTheMatch(t *testing.T) {
	g, _ := testGame()
	a, b := fakeCard("A"), pinCard("B")
	ms := NewMatchScreen(a, b, engine.MatchSingles, g)
	ms.match.IsFeud = true
	ms.match.SetDice(&fixedDice{rolls: []int{1}})
	ms.RunMatch(g)

	result := ms.match.Result()
	if result == nil || result.InjuredWrestler != "A" || result.InjuryCards != 2 {
		t.Fatalf("expected A injured for 2 cards, got %+v", result)
	}
	if got := g.Injuries.InjuryCards("A"); got != 2 {
		t.Fatalf("stored injury: %d cards, want 2", got)
	}
}

func TestBattleRoyalIsOneFightCard(t *testing.T) {
	g, _ := testGame("A", "B", "C", "D")
	g.Injuries.RecordInjury("Bystander", 3)

	br := NewBattleRoyalScreen(g.Roster, g)
	br.champion = br.wrestlers[0]
	for br.nextIdx < len(br.wrestlers) {
		br.startNextMatch(g)
		br.finishSubMatch(g)
	}
	if got := g.Injuries.InjuryCards("Bystander"); got != 2 {
		t.Fatalf("after a 3-round battle royal: %d cards left, want 2", got)
	}
}

func TestEmbeddedBattleRoyalLeavesTheFightCardToTheShow(t *testing.T) {
	g, _ := testGame("A", "B", "C")
	g.Injuries.RecordInjury("Bystander", 3)

	br := NewBattleRoyalScreen(g.Roster, g)
	br.embedded = true
	br.champion = br.wrestlers[0]
	for br.nextIdx < len(br.wrestlers) {
		br.startNextMatch(g)
		br.finishSubMatch(g)
	}
	if got := g.Injuries.InjuryCards("Bystander"); got != 3 {
		t.Fatalf("embedded battle royal counted down injuries: %d left, want 3", got)
	}
	if len(br.cardResults) != 2 {
		t.Fatalf("card results: got %d, want 2", len(br.cardResults))
	}
}

func TestTournamentIsOneFightCard(t *testing.T) {
	g, _ := testGame("A", "B", "C", "D")
	g.Injuries.RecordInjury("Bystander", 3)

	ts := newBracket(4)
	copy(ts.seeds, g.Roster)
	for ts.phase != TournFinished {
		ts.startCurrentMatch(g)
		ts.finishSubMatch(g)
		ts.advanceToNext(g)
	}
	if got := g.Injuries.InjuryCards("Bystander"); got != 2 {
		t.Fatalf("after a 4-man tournament: %d cards left, want 2", got)
	}
}

func TestMatchResultBanner(t *testing.T) {
	cases := []struct {
		name   string
		result *engine.MatchResult
		want   string
	}{
		{"winner", &engine.MatchResult{Winner: "A", Loser: "B", Method: "pinfall"}, "WINNER: A by pinfall"},
		{"double dq", &engine.MatchResult{WinningSide: -1, Method: "double dq"}, "NO WINNER: DOUBLE DISQUALIFICATION"},
		{"time limit", nil, "MATCH ENDED IN A DRAW"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchResultBanner(tc.result); !strings.Contains(got, tc.want) {
				t.Fatalf("banner %q does not contain %q", got, tc.want)
			}
		})
	}
}

func TestStatusMarkers(t *testing.T) {
	g, _ := testGame("A", "B", "C")
	g.Injuries.RecordInjury("A", 2)
	g.Injuries.RecordSuspension("B", 3)
	if got := statusMarkers(g, "A"); got != "  [INJURED 2]" {
		t.Errorf("injured marker: %q", got)
	}
	if got := statusMarkers(g, "B"); got != "  [SUSPENDED 3]" {
		t.Errorf("suspended marker: %q", got)
	}
	if got := statusMarkers(g, "C"); got != "" {
		t.Errorf("healthy wrestler marker: %q", got)
	}
}

func TestNewGameWarnsAboutUnreadableInjuries(t *testing.T) {
	store := newMemStore()
	store.injuries = []byte(`[[[broken`)
	g := NewGame(nil, store)
	if !strings.Contains(g.notice, "injur") {
		t.Fatalf("notice: %q", g.notice)
	}
}
