package engine

import (
	"fmt"
	"testing"
)

func bookingRoster(size int) ([]*WrestlerCard, []string) {
	cards := make([]*WrestlerCard, size)
	names := make([]string, size)
	for i := range cards {
		names[i] = fmt.Sprintf("W%02d", i)
		cards[i] = &WrestlerCard{Name: names[i]}
	}
	return cards, names
}

// ppvFederation is a federation standing on its first PPV week.
func ppvFederation(size int, belts ...string) (*Federation, []*WrestlerCard) {
	cards, names := bookingRoster(size)
	fed := NewFederation(FederationConfig{Name: "Test Fed", RosterNames: names, ChampNames: belts, PPVFrequency: 4})
	fed.Week = 4
	return fed, cards
}

func namesOn(match BookedMatch) []string {
	names := append([]string{}, match.Side1...)
	names = append(names, match.Side2...)
	names = append(names, match.BREntrants...)
	return append(names, match.TournSeeds...)
}

func wantNoWrestlerTwice(t *testing.T, card []BookedMatch) {
	t.Helper()
	seen := map[string]int{}
	for i, match := range card {
		for _, name := range namesOn(match) {
			if earlier, ok := seen[name]; ok {
				t.Fatalf("%s is booked in match %d and match %d", name, earlier+1, i+1)
			}
			seen[name] = i
		}
	}
}

func titleMatchFor(card []BookedMatch, titleIndex int) (BookedMatch, bool) {
	for _, match := range card {
		if match.IsTitle && match.TitleIndex == titleIndex {
			return match, true
		}
	}
	return BookedMatch{}, false
}

func onCard(card []BookedMatch, name string) bool {
	for _, match := range card {
		for _, booked := range namesOn(match) {
			if booked == name {
				return true
			}
		}
	}
	return false
}

func giveRecord(fed *Federation, name string, wins int) {
	fed.Records[name] = &WrestlerRecord{Wins: wins}
}

func TestPPVBooksATournamentForAVacantMainTitle(t *testing.T) {
	cases := []struct {
		roster   int
		belts    []string
		wantSize int
	}{
		{4, []string{"World"}, 4},
		{6, []string{"World"}, 4},
		{8, []string{"World"}, 8},
		{10, []string{"World"}, 8},
		{8, []string{"World", "TV"}, 4},
		{10, []string{"World", "TV"}, 8},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("roster %d with %d belts", tc.roster, len(tc.belts)), func(t *testing.T) {
			for attempt := 0; attempt < 20; attempt++ {
				fed, cards := ppvFederation(tc.roster, tc.belts...)
				card := fed.AutoBook(cards)
				if len(card) == 0 {
					t.Fatal("empty card")
				}
				mainEvent := card[len(card)-1]
				if !mainEvent.IsTournament || !mainEvent.IsTitle || mainEvent.TitleIndex != 0 {
					t.Fatalf("last match is not the title tournament: %+v", mainEvent)
				}
				if mainEvent.TournSize != tc.wantSize || len(mainEvent.TournSeeds) != tc.wantSize {
					t.Fatalf("tournament size %d with %d seeds, want %d", mainEvent.TournSize, len(mainEvent.TournSeeds), tc.wantSize)
				}
				wantNoWrestlerTwice(t, card)
			}
		})
	}
}

func TestPPVBooksEveryTitleBeforeAnythingElse(t *testing.T) {
	for attempt := 0; attempt < 20; attempt++ {
		fed, cards := ppvFederation(12, "World", "TV", "Tag")
		fed.Championships[0].Champion = "W00"
		fed.Championships[1].Champion = "W01"
		for _, pair := range [][2]string{{"W02", "W03"}, {"W04", "W05"}, {"W06", "W07"}, {"W08", "W09"}, {"W10", "W11"}} {
			fed.AddRivalry(pair[0], pair[1], 3)
		}

		card := fed.AutoBook(cards)
		wantNoWrestlerTwice(t, card)

		mainEvent := card[len(card)-1]
		if !mainEvent.IsTitle || mainEvent.TitleIndex != 0 || mainEvent.Side1[0] != "W00" {
			t.Fatalf("main event should be W00 defending the main title, got %+v", mainEvent)
		}
		defense, ok := titleMatchFor(card, 1)
		if !ok || defense.Side1[0] != "W01" {
			t.Fatalf("no defense booked for the second title: %+v", card)
		}
		if _, ok := titleMatchFor(card, 2); !ok {
			t.Fatalf("no match booked for the vacant third title: %+v", card)
		}
	}
}

func TestContendersAreNotPulledAwayFromTheirOwnTitleDefense(t *testing.T) {
	fed, cards := ppvFederation(8, "World", "TV")
	fed.Championships[0].Champion = "W00"
	fed.Championships[1].Champion = "W01"
	giveRecord(fed, "W01", 9)
	giveRecord(fed, "W02", 4)

	card := fed.AutoBook(cards)
	mainEvent, _ := titleMatchFor(card, 0)
	if got := mainEvent.Side2[0]; got != "W02" {
		t.Fatalf("main title contender is %s, want W02 (W01 has his own title to defend)", got)
	}
	if defense, ok := titleMatchFor(card, 1); !ok || defense.Side1[0] != "W01" {
		t.Fatalf("W01 is not defending his title: %+v", card)
	}
}

func TestTitleShotGoesToTheWrestlerWhoEarnedItAndIsKeptUntilUsed(t *testing.T) {
	fed, cards := ppvFederation(8, "World")
	fed.Championships[0].Champion = "W00"
	fed.TitleShotEarned = "W05"

	card := fed.AutoBook(cards)
	mainEvent, ok := titleMatchFor(card, 0)
	if !ok || mainEvent.Side2[0] != "W05" {
		t.Fatalf("main event: %+v, want W00 against W05", mainEvent)
	}
	if fed.TitleShotEarned != "W05" {
		t.Fatal("booking a card used up the title shot before the match took place")
	}
}

func TestAutoBookOnlyUsesTheWrestlersItIsGiven(t *testing.T) {
	for attempt := 0; attempt < 20; attempt++ {
		fed, cards := ppvFederation(9, "World", "TV")
		fed.Championships[0].Champion = "W00"
		fed.Championships[1].Champion = "W01"
		giveRecord(fed, "W02", 9)
		fed.AddRivalry("W02", "W03", 5)

		var available []*WrestlerCard
		for _, card := range cards {
			if card.Name != "W00" && card.Name != "W02" {
				available = append(available, card)
			}
		}
		card := fed.AutoBook(available)
		for _, missing := range []string{"W00", "W02"} {
			if onCard(card, missing) {
				t.Fatalf("%s was booked without being available: %+v", missing, card)
			}
		}
		if _, ok := titleMatchFor(card, 0); ok {
			t.Fatalf("main title match booked without the champion: %+v", card)
		}
		wantNoWrestlerTwice(t, card)
	}
}

func TestWeeklyShowHasNoTitleMatches(t *testing.T) {
	for attempt := 0; attempt < 20; attempt++ {
		fed, cards := ppvFederation(10, "World")
		fed.Week = 2
		fed.Championships[0].Champion = "W00"
		fed.AddRivalry("W01", "W02", 5)

		card := fed.AutoBook(cards)
		if len(card) == 0 || len(card) > 3 {
			t.Fatalf("weekly card has %d matches, want 1 to 3", len(card))
		}
		for _, match := range card {
			if match.IsTitle || match.IsTournament || len(match.BREntrants) > 0 {
				t.Fatalf("weekly show booked %+v", match)
			}
		}
		wantNoWrestlerTwice(t, card)
	}
}

func TestChampionKeepsTheTitleBetweenPPVs(t *testing.T) {
	fed, _ := ppvFederation(4, "World")
	fed.Week = 1
	fed.ChangeTitleHolder(0, "W00", "", "tournament")
	for fed.Week < 4 {
		fed.AdvanceWeek()
		if fed.MainChampion() != "W00" {
			t.Fatalf("champion stripped on a weekly show, week %d", fed.Week)
		}
	}
}

func TestChampionIsStrippedWhenAPPVPassesWithoutATitleMatch(t *testing.T) {
	fed, _ := ppvFederation(4, "World")
	fed.ChangeTitleHolder(0, "W00", "", "tournament")

	fed.AdvanceWeek()
	if fed.MainChampion() != "W00" {
		t.Fatal("a champion crowned on this PPV was stripped at the end of it")
	}

	fed.Week = 8
	fed.RecordTitleMatch(0)
	fed.AdvanceWeek()
	if fed.MainChampion() != "W00" {
		t.Fatal("champion was stripped after defending on the PPV")
	}

	fed.Week = 12
	fed.AdvanceWeek()
	if fed.MainChampion() != "" {
		t.Fatal("champion kept the title through a PPV with no title match")
	}
	last := fed.Championships[0].History[len(fed.Championships[0].History)-1]
	if last.Method != "vacated" || last.Loser != "W00" || last.Week != 12 {
		t.Fatalf("title history: %+v", last)
	}
}

func TestPPVNamesAreUsedInOrderFromTheFirst(t *testing.T) {
	_, names := bookingRoster(4)
	fed := NewFederation(FederationConfig{
		Name: "Test Fed", RosterNames: names, ChampNames: []string{"World"},
		PPVFrequency: 2, PPVNames: []string{"FIRST", "SECOND"},
	})
	var seen []string
	for fed.Week <= 6 {
		if fed.IsPPV() {
			seen = append(seen, fed.ShowName())
		}
		fed.AdvanceWeek()
	}
	want := []string{"FIRST", "SECOND", "FIRST"}
	if fmt.Sprint(seen) != fmt.Sprint(want) {
		t.Fatalf("PPV names: %v, want %v", seen, want)
	}
}

func TestRecordResultKeepsTheMatchType(t *testing.T) {
	fed, _ := ppvFederation(4, "World")
	fed.RecordResult(MatchHistoryEntry{Winner: "W00", Loser: "W01", Method: "pinfall", MatchType: "CAGE", IsTitle: true})

	entry := fed.MatchHistory[0]
	if entry.MatchType != "CAGE" || entry.Week != 4 || !entry.IsTitle {
		t.Fatalf("history entry: %+v", entry)
	}
	if fed.Records["W00"].Wins != 1 || fed.Records["W00"].WinsByPin != 1 || fed.Records["W01"].Losses != 1 {
		t.Fatalf("records: %+v %+v", fed.Records["W00"], fed.Records["W01"])
	}
}

func TestCloneIsIndependentOfTheOriginal(t *testing.T) {
	fed, _ := ppvFederation(4, "World")
	fed.ChangeTitleHolder(0, "W00", "", "tournament")
	fed.AddRivalry("W00", "W01", 3)

	snapshot, err := fed.Clone()
	if err != nil {
		t.Fatal(err)
	}
	fed.RecordResult(MatchHistoryEntry{Winner: "W01", Loser: "W00", Method: "pinfall"})
	fed.ChangeTitleHolder(0, "W01", "W00", "pinfall")
	fed.AddRivalry("W00", "W01", 2)
	fed.AdvanceWeek()

	if snapshot.MainChampion() != "W00" || snapshot.Week != 4 {
		t.Fatalf("snapshot changed: champion %q week %d", snapshot.MainChampion(), snapshot.Week)
	}
	if snapshot.Records["W01"].Wins != 0 || len(snapshot.MatchHistory) != 0 {
		t.Fatalf("snapshot records changed: %+v", snapshot.Records["W01"])
	}
	if snapshot.RivalryScore("W00", "W01") != 3 {
		t.Fatalf("snapshot rivalry changed: %d", snapshot.RivalryScore("W00", "W01"))
	}
}

func TestPPVFrequencyUnderTwoFallsBackToFour(t *testing.T) {
	_, names := bookingRoster(4)
	for _, frequency := range []int{-1, 0, 1} {
		fed := NewFederation(FederationConfig{Name: "Test Fed", RosterNames: names, PPVFrequency: frequency})
		if fed.PPVFrequency != 4 {
			t.Errorf("frequency %d: federation uses %d, want 4", frequency, fed.PPVFrequency)
		}
	}
}
