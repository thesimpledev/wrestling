package ui

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
)

// strongCard reverses every move thrown at him, so he beats any pinCard.
func strongCard(name string) *engine.WrestlerCard {
	card := fakeCard(name)
	for lvl := range card.Offense {
		for slot := range card.Offense[lvl] {
			card.Offense[lvl][slot] = engine.Move{Name: "Jab", Power: 1, DefLevel: 1}
			card.Defense[lvl][slot] = engine.DefenseOutcome{Type: engine.DefReversal, Power: 1}
		}
	}
	return card
}

// fedGame is a game whose roster is one federation with the given belts.
func fedGame(belts []string, cards ...*engine.WrestlerCard) (*Game, *fakeInput, *engine.Federation, *engine.FederationSave) {
	in := &fakeInput{}
	g := NewGame(cards, newMemStore())
	g.in = in
	names := make([]string, len(cards))
	for i, card := range cards {
		names[i] = card.Name
	}
	fed := engine.NewFederation(engine.FederationConfig{Name: "Test Fed", RosterNames: names, ChampNames: belts, PPVFrequency: 4})
	save := &engine.FederationSave{Federations: []*engine.Federation{fed}}
	return g, in, fed, save
}

func singles(a, b string) engine.BookedMatch {
	return engine.BookedMatch{Type: engine.MatchSingles, TitleIndex: -1, Side1: []string{a}, Side2: []string{b}}
}

func titleMatch(titleIndex int, champion, challenger string) engine.BookedMatch {
	return engine.BookedMatch{Type: engine.MatchSingles, IsTitle: true, TitleIndex: titleIndex, Side1: []string{champion}, Side2: []string{challenger}}
}

func runShow(t *testing.T, g *Game, in *fakeInput, show *CareerShowScreen) {
	t.Helper()
	g.SetScreen(show)
	for presses := 0; show.phase != ShowComplete; presses++ {
		if presses > 5000 {
			t.Fatalf("show did not finish; on match %d of %d", show.currentIdx+1, len(show.card))
		}
		press(t, g, in, ebiten.KeySpace)
	}
}

func pinCards(names ...string) []*engine.WrestlerCard {
	cards := make([]*engine.WrestlerCard, len(names))
	for i, name := range names {
		cards[i] = pinCard(name)
	}
	return cards
}

func TestSimulatedShowRunsTheMatchAfterABattleRoyal(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, pinCards("A", "B", "C", "D", "E", "F", "G", "H")...)
	card := []engine.BookedMatch{
		singles("A", "B"),
		{Type: engine.MatchSingles, TitleIndex: -1, BREntrants: []string{"C", "D", "E", "F"}},
		singles("G", "H"),
	}
	show := NewCareerShowScreen(fed, save, card, ShowModeSimulate, g)
	runShow(t, g, in, show)

	if len(show.results) != 3 {
		t.Fatalf("results: %v, want one line per match", show.results)
	}
	if fed.Records["G"].Wins+fed.Records["H"].Wins != 1 {
		t.Fatalf("the match after the battle royal was not played: %v", show.results)
	}
	if fed.Week != 2 {
		t.Fatalf("week %d after the show, want 2", fed.Week)
	}
}

func TestBattleRoyalRecordsEachRoundsRealWinnerAndLoser(t *testing.T) {
	for _, mode := range []ShowMode{ShowModeSimulate, ShowModeWatch} {
		g, in, fed, save := fedGame([]string{"World"}, pinCards("A", "B", "C", "D", "E")...)
		card := []engine.BookedMatch{{Type: engine.MatchSingles, TitleIndex: -1, BREntrants: []string{"A", "B", "C", "D", "E"}}}
		show := NewCareerShowScreen(fed, save, card, mode, g)
		runShow(t, g, in, show)

		if len(fed.MatchHistory) != 4 {
			t.Fatalf("mode %d: %d rounds in the history, want 4", mode, len(fed.MatchHistory))
		}
		for i, entry := range fed.MatchHistory {
			result := show.cardResults[i]
			if entry.Winner != result.Winner || entry.Loser != result.Loser || entry.Method != result.Method {
				t.Fatalf("mode %d round %d: history %+v, match result %+v", mode, i+1, entry, result)
			}
			if entry.MatchType != "BATTLE ROYAL" {
				t.Fatalf("mode %d round %d: match type %q", mode, i+1, entry.MatchType)
			}
		}
		if fed.TitleShotEarned != show.brScreen.champion.Name {
			t.Fatalf("mode %d: title shot went to %q, winner was %s", mode, fed.TitleShotEarned, show.brScreen.champion.Name)
		}
	}
}

func TestShowRecordsTheMatchTypeInTheHistory(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, strongCard("A"), pinCard("B"))
	cage := singles("A", "B")
	cage.Type = engine.MatchCage
	runShow(t, g, in, NewCareerShowScreen(fed, save, []engine.BookedMatch{cage}, ShowModeSimulate, g))

	if len(fed.MatchHistory) != 1 || fed.MatchHistory[0].MatchType != "CAGE" || fed.MatchHistory[0].Winner != "A" {
		t.Fatalf("history: %+v", fed.MatchHistory)
	}
	lines := strings.Join(matchHistoryLines(fed), "\n")
	if !strings.Contains(lines, "CAGE") {
		t.Fatalf("history screen does not show the match type:\n%s", lines)
	}
}

func TestTournamentInAShowCrownsTheChampionAndRecordsEveryMatch(t *testing.T) {
	for _, mode := range []ShowMode{ShowModeSimulate, ShowModeWatch} {
		g, in, fed, save := fedGame([]string{"World"}, pinCards("A", "B", "C", "D")...)
		fed.Week = 4
		card := []engine.BookedMatch{{IsTournament: true, TournSize: 4, TournSeeds: []string{"A", "B", "C", "D"}, IsTitle: true}}
		show := NewCareerShowScreen(fed, save, card, mode, g)
		runShow(t, g, in, show)

		champion := fed.MainChampion()
		if champion == "" {
			t.Fatalf("mode %d: tournament crowned nobody: %v", mode, show.results)
		}
		if len(fed.MatchHistory) != 3 {
			t.Fatalf("mode %d: %d tournament matches in the history, want 3", mode, len(fed.MatchHistory))
		}
		if final := fed.MatchHistory[2]; final.Winner != champion || final.MatchType != "TOURNAMENT" {
			t.Fatalf("mode %d: final %+v, champion %s", mode, final, champion)
		}
		if len(show.cardResults) != 3 {
			t.Fatalf("mode %d: %d results kept for the fight card, want 3", mode, len(show.cardResults))
		}
	}
}

func TestEscapeDuringAShowAsksFirst(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, strongCard("A"), pinCard("B"))
	show := NewCareerShowScreen(fed, save, []engine.BookedMatch{singles("A", "B")}, ShowModeWatch, g)
	g.SetScreen(show)

	press(t, g, in, ebiten.KeyEscape)
	if g.screen != Screen(show) || !show.askingToLeave {
		t.Fatalf("ESC left the show without asking (screen %T)", g.screen)
	}
	press(t, g, in, ebiten.KeyEscape)
	if g.screen != Screen(show) || show.askingToLeave {
		t.Fatal("a second ESC should go back to the show")
	}
}

func TestCancellingAShowPutsTheFederationBackAsItWas(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, strongCard("A"), pinCard("B"), strongCard("C"), pinCard("D"))
	fed.Week = 4
	fed.ChangeTitleHolder(0, "B", "", "tournament")
	card := []engine.BookedMatch{titleMatch(0, "B", "A"), singles("C", "D")}
	show := NewCareerShowScreen(fed, save, card, ShowModeMainEvent, g)
	g.SetScreen(show)
	if fed.MainChampion() != "A" {
		t.Fatalf("setup: the simulated title match should have changed the title, champion %q", fed.MainChampion())
	}

	press(t, g, in, ebiten.KeyEscape)
	press(t, g, in, ebiten.KeyC)

	if _, ok := g.screen.(*CareerBookScreen); !ok {
		t.Fatalf("screen is %T, want the booking screen", g.screen)
	}
	if fed.MainChampion() != "B" || fed.Week != 4 {
		t.Fatalf("after cancelling: champion %q week %d", fed.MainChampion(), fed.Week)
	}
	if fed.Records["A"].Wins != 0 || len(fed.MatchHistory) != 0 {
		t.Fatalf("after cancelling: records %+v, history %+v", fed.Records["A"], fed.MatchHistory)
	}
	if save.Federations[0] != fed {
		t.Fatal("the save no longer points at the federation")
	}
	if g.Store.(*memStore).career != nil {
		t.Fatal("a cancelled show was saved")
	}
}

func TestSkippingAShowSimulatesTheRest(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, pinCards("A", "B", "C", "D", "E", "F", "G", "H", "I", "J")...)
	fed.Week = 4
	card := []engine.BookedMatch{
		singles("A", "B"),
		{Type: engine.MatchSingles, TitleIndex: -1, BREntrants: []string{"C", "D", "E", "F"}},
		{IsTournament: true, TournSize: 4, TournSeeds: []string{"G", "H", "I", "J"}, IsTitle: true},
	}
	show := NewCareerShowScreen(fed, save, card, ShowModeWatch, g)
	g.SetScreen(show)
	press(t, g, in, ebiten.KeySpace)

	press(t, g, in, ebiten.KeyEscape)
	press(t, g, in, ebiten.KeyS)

	if show.phase != ShowComplete {
		t.Fatalf("phase %d after skipping, want the results", show.phase)
	}
	if len(show.results) != 3 {
		t.Fatalf("results: %v", show.results)
	}
	if fed.MainChampion() == "" || fed.Week != 5 {
		t.Fatalf("champion %q week %d", fed.MainChampion(), fed.Week)
	}
	if g.Store.(*memStore).career == nil {
		t.Fatal("the finished show was not saved")
	}
}

func TestSkippingPartWayThroughABattleRoyalKeepsTheRoundsAlreadyShown(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, pinCards("A", "B", "C", "D")...)
	card := []engine.BookedMatch{{Type: engine.MatchSingles, TitleIndex: -1, BREntrants: []string{"A", "B", "C", "D"}}}
	show := NewCareerShowScreen(fed, save, card, ShowModeWatch, g)
	g.SetScreen(show)
	for show.brScreen.phase != BRMatchResult {
		press(t, g, in, ebiten.KeySpace)
	}
	firstRound := *show.brScreen.cardResults[0]

	press(t, g, in, ebiten.KeyEscape)
	press(t, g, in, ebiten.KeyS)

	if len(fed.MatchHistory) != 3 {
		t.Fatalf("%d rounds recorded, want 3", len(fed.MatchHistory))
	}
	if first := fed.MatchHistory[0]; first.Winner != firstRound.Winner || first.Loser != firstRound.Loser {
		t.Fatalf("first round was replayed: shown %+v, recorded %+v", firstRound, first)
	}
}

func TestResultsNameTheShowJustCompleted(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, strongCard("A"), pinCard("B"))
	fed.Week = 4
	fed.PPVNames = []string{"FIRST PPV", "SECOND PPV"}
	show := NewCareerShowScreen(fed, save, []engine.BookedMatch{singles("A", "B")}, ShowModeSimulate, g)
	runShow(t, g, in, show)

	text := strings.Join(show.completeLines(), "\n")
	for _, want := range []string{"FIRST PPV: RESULTS", "Week 4 complete", "A def. B"} {
		if !strings.Contains(text, want) {
			t.Errorf("results do not contain %q:\n%s", want, text)
		}
	}
}

func TestTagMatchInAShowIsTwoOnTwo(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, strongCard("A"), strongCard("B"), pinCard("C"), pinCard("D"))
	tag := engine.BookedMatch{
		Type: engine.MatchTag, TitleIndex: -1,
		Side1: []string{"A", "B"}, Side2: []string{"C", "D"}, Side1Regular: true,
	}
	show := NewCareerShowScreen(fed, save, []engine.BookedMatch{tag}, ShowModeSimulate, g)
	if got := len(show.match.Sides[0].Wrestlers); got != 2 {
		t.Fatalf("side 1 has %d wrestlers, want 2", got)
	}
	if !show.match.Sides[0].RegularPartners || show.match.Sides[1].RegularPartners {
		t.Fatal("regular-partner answers were not carried into the match")
	}
	runShow(t, g, in, show)

	if len(fed.MatchHistory) != 1 || fed.MatchHistory[0].MatchType != "TAG TEAM" {
		t.Fatalf("history: %+v", fed.MatchHistory)
	}
	winner := fed.MatchHistory[0].Winner
	if winner != "A" && winner != "B" {
		t.Fatalf("winner %q is not on the winning team", winner)
	}
}

func TestTitleMatchOnAPPVKeepsOrChangesTheTitle(t *testing.T) {
	t.Run("champion retains", func(t *testing.T) {
		g, in, fed, save := fedGame([]string{"World"}, strongCard("A"), pinCard("B"))
		fed.Week = 8
		fed.Championships[0].Champion = "A"
		fed.TitleShotEarned = "B"
		runShow(t, g, in, NewCareerShowScreen(fed, save, []engine.BookedMatch{titleMatch(0, "A", "B")}, ShowModeSimulate, g))
		if fed.MainChampion() != "A" {
			t.Fatalf("champion %q, want A to keep the title he defended", fed.MainChampion())
		}
		if fed.TitleShotEarned != "" {
			t.Fatal("the title shot was not used up by the title match")
		}
	})
	t.Run("challenger wins", func(t *testing.T) {
		g, in, fed, save := fedGame([]string{"World"}, pinCard("A"), strongCard("B"))
		fed.Week = 8
		fed.Championships[0].Champion = "A"
		runShow(t, g, in, NewCareerShowScreen(fed, save, []engine.BookedMatch{titleMatch(0, "A", "B")}, ShowModeSimulate, g))
		if fed.MainChampion() != "B" {
			t.Fatalf("champion %q, want B", fed.MainChampion())
		}
	})
	t.Run("no title match", func(t *testing.T) {
		g, in, fed, save := fedGame([]string{"World"}, strongCard("A"), pinCard("B"))
		fed.Week = 8
		fed.Championships[0].Champion = "A"
		show := NewCareerShowScreen(fed, save, []engine.BookedMatch{singles("A", "B")}, ShowModeSimulate, g)
		runShow(t, g, in, show)
		if fed.MainChampion() != "" {
			t.Fatalf("champion %q kept the title through a PPV with no title match", fed.MainChampion())
		}
		if text := strings.Join(show.completeLines(), "\n"); !strings.Contains(text, "A stripped") {
			t.Fatalf("results do not say the champion was stripped:\n%s", text)
		}
	})
}

func TestNextShowLeavesSuspendedWrestlersOffTheCard(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, pinCards("A", "B", "C", "D", "E", "F")...)
	g.Injuries.RecordSuspension("A", 2)
	g.Injuries.RecordSuspension("D", 1)
	g.SetScreen(NewCareerScreen(fed, save))
	enter(t, g, in)

	book, ok := g.screen.(*CareerBookScreen)
	if !ok {
		t.Fatalf("screen is %T, want the booking screen", g.screen)
	}
	if len(book.card) == 0 {
		t.Fatal("empty card")
	}
	for _, match := range book.card {
		for _, name := range append(append([]string{}, match.Side1...), match.Side2...) {
			if name == "A" || name == "D" {
				t.Fatalf("suspended wrestler %s was booked: %+v", name, book.card)
			}
		}
	}
}

func bookScreen(card []engine.BookedMatch, belts []string, names ...string) (*Game, *fakeInput, *CareerBookScreen) {
	g, in, fed, save := fedGame(belts, pinCards(names...)...)
	book := NewCareerBookScreen(fed, save, card, g)
	g.SetScreen(book)
	return g, in, book
}

func TestEditingAMatchBlocksBookedAndSuspendedWrestlers(t *testing.T) {
	card := []engine.BookedMatch{singles("A", "B"), singles("C", "D")}
	g, in, book := bookScreen(card, []string{"World"}, "A", "B", "C", "D", "E", "F")
	g.Injuries.RecordSuspension("E", 2)

	enter(t, g, in)
	enter(t, g, in)
	if book.phase != BookEditWrestlers {
		t.Fatalf("phase %d, want wrestler selection", book.phase)
	}

	down(t, g, in, 2)
	enter(t, g, in)
	if len(book.editPicks) != 0 {
		t.Fatalf("C is in match 2 but was picked: %v", book.editPicks)
	}
	down(t, g, in, 2)
	enter(t, g, in)
	if len(book.editPicks) != 0 {
		t.Fatalf("E is suspended but was picked: %v", book.editPicks)
	}
	down(t, g, in, 1)
	enter(t, g, in)
	if len(book.editPicks) != 1 || book.editPicks[0] != "F" {
		t.Fatalf("picks: %v, want F", book.editPicks)
	}
}

func TestEditingAMatchBlocksPickingTheSameWrestlerTwice(t *testing.T) {
	card := []engine.BookedMatch{singles("A", "B"), singles("C", "D")}
	g, in, book := bookScreen(card, []string{"World"}, "A", "B", "C", "D", "E", "F")

	enter(t, g, in)
	enter(t, g, in)
	down(t, g, in, 5)
	enter(t, g, in)
	if book.editCursor != 0 {
		t.Fatalf("cursor on %d after the first pick, want the first free wrestler (A)", book.editCursor)
	}

	down(t, g, in, 5)
	enter(t, g, in)
	if len(book.editPicks) != 1 {
		t.Fatalf("F was picked twice for one match: %v", book.editPicks)
	}

	for step := 0; step < 4; step++ {
		press(t, g, in, ebiten.KeyUp)
	}
	enter(t, g, in)
	if book.phase != BookViewCard {
		t.Fatalf("phase %d, want the card view after the last pick", book.phase)
	}
	if got := book.card[0]; got.Side1[0] != "F" || got.Side2[0] != "B" {
		t.Fatalf("match 1 is %v vs %v, want F vs B", got.Side1, got.Side2)
	}
}

func TestDashboardShowsChampionsContenderAndRivalries(t *testing.T) {
	g, _, fed, save := fedGame([]string{"World", "TV"}, pinCards("A", "B", "C", "D")...)
	fed.Championships[0].Champion = "A"
	fed.TitleShotEarned = "B"
	fed.AddRivalry("C", "D", 4)
	g.Injuries.RecordInjury("A", 2)

	screen := NewCareerScreen(fed, save)
	text := strings.Join(screen.dashboardLines(g), "\n")
	for _, want := range []string{
		"TEST FED", "Week 1", "World: A  [INJURED 2]", "TV: VACANT",
		"#1 Contender: B", "C vs D (intensity: 4)", "> Next Show", "  Quit Federation",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("dashboard does not contain %q:\n%s", want, text)
		}
	}
}

func TestEditingAMatchToTagTeamAsksForFourWrestlersAndPartnerStatus(t *testing.T) {
	g, in, book := bookScreen([]engine.BookedMatch{singles("A", "B")}, []string{"World"}, "A", "B", "C", "D", "E")

	enter(t, g, in)
	down(t, g, in, 1)
	enter(t, g, in)
	for pick := 0; pick < 4; pick++ {
		enter(t, g, in)
	}
	if book.phase != BookEditPartners {
		t.Fatalf("phase %d, want the regular-partners question", book.phase)
	}
	if got := book.card[0]; got.Type != engine.MatchSingles || len(got.Side1) != 1 {
		t.Fatalf("the card changed before the edit was finished: %+v", got)
	}
	enter(t, g, in)
	down(t, g, in, 1)
	enter(t, g, in)

	got := book.card[0]
	if got.Type != engine.MatchTag {
		t.Fatalf("type %v, want tag", got.Type)
	}
	if strings.Join(got.Side1, ",") != "A,B" || strings.Join(got.Side2, ",") != "C,D" {
		t.Fatalf("sides %v vs %v, want A,B vs C,D", got.Side1, got.Side2)
	}
	if !got.Side1Regular || got.Side2Regular {
		t.Fatalf("regular partners: %v %v, want yes then no", got.Side1Regular, got.Side2Regular)
	}
	if line := bookedMatchLine(book.fed, g, got); !strings.Contains(line, "A & B vs C & D") {
		t.Fatalf("card line: %q", line)
	}
}

func TestTagMatchNeedsFourFreeWrestlers(t *testing.T) {
	card := []engine.BookedMatch{singles("A", "B"), singles("C", "D")}
	g, in, book := bookScreen(card, []string{"World"}, "A", "B", "C", "D", "E")

	enter(t, g, in)
	down(t, g, in, 1)
	enter(t, g, in)
	if book.phase != BookEditType {
		t.Fatalf("phase %d: a tag match was started with only 3 free wrestlers", book.phase)
	}
	if !strings.Contains(g.notice, "needs 4") {
		t.Fatalf("notice: %q", g.notice)
	}
}

func TestEscapeDuringAnEditLeavesTheMatchAlone(t *testing.T) {
	g, in, book := bookScreen([]engine.BookedMatch{singles("A", "B")}, []string{"World"}, "A", "B", "C", "D")

	enter(t, g, in)
	down(t, g, in, 2)
	enter(t, g, in)
	down(t, g, in, 2)
	enter(t, g, in)
	press(t, g, in, ebiten.KeyEscape)

	if book.phase != BookViewCard {
		t.Fatalf("phase %d after ESC", book.phase)
	}
	if got := book.card[0]; got.Type != engine.MatchSingles || got.Side1[0] != "A" || got.Side2[0] != "B" {
		t.Fatalf("match changed by a cancelled edit: %+v", got)
	}
}

func TestTakingTheChampionOutOfATitleMatchMakesItANonTitleMatch(t *testing.T) {
	g, in, book := bookScreen([]engine.BookedMatch{titleMatch(0, "A", "B")}, []string{"World"}, "A", "B", "C", "D")
	book.fed.Championships[0].Champion = "A"

	enter(t, g, in)
	enter(t, g, in)
	down(t, g, in, 2)
	enter(t, g, in)
	enter(t, g, in)

	got := book.card[0]
	if got.Side1[0] != "C" || got.Side2[0] != "A" {
		t.Fatalf("sides %v vs %v, want C vs A", got.Side1, got.Side2)
	}
	if !got.IsTitle {
		t.Fatal("the champion is still in the match, so it should still be for the title")
	}

	enter(t, g, in)
	enter(t, g, in)
	down(t, g, in, 1)
	enter(t, g, in)
	down(t, g, in, 3)
	enter(t, g, in)

	got = book.card[0]
	if got.Side1[0] != "B" || got.Side2[0] != "D" {
		t.Fatalf("sides %v vs %v, want B vs D", got.Side1, got.Side2)
	}
	if got.IsTitle || got.TitleIndex != -1 {
		t.Fatalf("a match without the champion is still a title match: %+v", got)
	}
	if !strings.Contains(g.notice, "no longer a title match") {
		t.Fatalf("notice: %q", g.notice)
	}
}

func TestBattleRoyalsAndTournamentsCannotBeEdited(t *testing.T) {
	card := []engine.BookedMatch{
		{Type: engine.MatchSingles, TitleIndex: -1, BREntrants: []string{"A", "B", "C", "D"}},
		{IsTournament: true, TournSize: 4, TournSeeds: []string{"E", "F", "G", "H"}, IsTitle: true},
	}
	g, in, book := bookScreen(card, []string{"World"}, "A", "B", "C", "D", "E", "F", "G", "H")

	for row := 0; row < 2; row++ {
		enter(t, g, in)
		if book.phase != BookViewCard {
			t.Fatalf("row %d opened for editing", row+1)
		}
		if !strings.Contains(g.notice, "cannot be edited") {
			t.Fatalf("row %d notice: %q", row+1, g.notice)
		}
		down(t, g, in, 1)
	}
}

func TestFederationSettingsEscapeDiscardsChanges(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, pinCards("A", "B", "C", "D")...)
	g.SetScreen(NewFederationSettingsScreen(fed, save))

	enter(t, g, in)
	typeText(t, g, in, " Two")
	enter(t, g, in)
	press(t, g, in, ebiten.KeyEscape)

	if fed.Name != "Test Fed" {
		t.Fatalf("ESC kept the change: federation is now %q", fed.Name)
	}
	if _, ok := g.screen.(*CareerScreen); !ok {
		t.Fatalf("screen is %T, want the federation dashboard", g.screen)
	}
}

func TestFederationSettingsSaveAppliesChanges(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, pinCards("A", "B", "C", "D")...)
	g.SetScreen(NewFederationSettingsScreen(fed, save))

	enter(t, g, in)
	typeText(t, g, in, " Two")
	enter(t, g, in)

	down(t, g, in, 2)
	enter(t, g, in)
	press(t, g, in, ebiten.KeyBackspace)
	typeText(t, g, in, "6")
	enter(t, g, in)

	down(t, g, in, 1)
	enter(t, g, in)
	typeText(t, g, in, "DOOMSDAY")
	enter(t, g, in)
	press(t, g, in, ebiten.KeyEscape)

	down(t, g, in, 1)
	enter(t, g, in)

	if fed.Name != "Test Fed Two" || fed.PPVFrequency != 6 {
		t.Fatalf("name %q frequency %d", fed.Name, fed.PPVFrequency)
	}
	if last := fed.PPVNames[len(fed.PPVNames)-1]; last != "DOOMSDAY" {
		t.Fatalf("last PPV name %q, want DOOMSDAY (a name starting with D)", last)
	}
	if g.Store.(*memStore).career == nil {
		t.Fatal("settings were not saved")
	}
}

func TestFederationSettingsRejectPPVFrequencyUnderTwo(t *testing.T) {
	g, in, fed, save := fedGame([]string{"World"}, pinCards("A", "B", "C", "D")...)
	screen := NewFederationSettingsScreen(fed, save)
	g.SetScreen(screen)

	down(t, g, in, 2)
	enter(t, g, in)
	press(t, g, in, ebiten.KeyBackspace)
	typeText(t, g, in, "1")
	enter(t, g, in)

	if !strings.Contains(g.notice, "2 or more") {
		t.Fatalf("notice: %q", g.notice)
	}
	if screen.ppvFrequency != 4 {
		t.Fatalf("frequency %d was accepted", screen.ppvFrequency)
	}
}

func TestBackspaceOnAnEmptyBoxRemovesTheLastName(t *testing.T) {
	in := &fakeInput{}
	input := NewTextInput(30)
	names := []string{"World"}

	for _, r := range "Diamond" {
		in.set(nil, string(r))
		names = updateNameList(in, input, names)
	}
	if input.Text != "Diamond" || len(names) != 1 {
		t.Fatalf("typing a name starting with D: text %q, names %v", input.Text, names)
	}
	in.set([]ebiten.Key{ebiten.KeyEnter}, "")
	names = updateNameList(in, input, names)
	if strings.Join(names, ",") != "World,Diamond" || input.Text != "" {
		t.Fatalf("after ENTER: names %v, text %q", names, input.Text)
	}

	in.set([]ebiten.Key{ebiten.KeyBackspace}, "")
	names = updateNameList(in, input, names)
	if strings.Join(names, ",") != "World" {
		t.Fatalf("after BACKSPACE on an empty box: %v", names)
	}
}

func TestCreatingAFederation(t *testing.T) {
	g, in, _, _ := fedGame(nil, pinCards("A", "B", "C", "D", "E")...)
	save := &engine.FederationSave{}
	g.SetScreen(NewFederationCreateScreen(save))

	typeText(t, g, in, "Dynamite")
	enter(t, g, in)
	for pick := 0; pick < 4; pick++ {
		press(t, g, in, ebiten.KeySpace)
		down(t, g, in, 1)
	}
	enter(t, g, in)
	typeText(t, g, in, "Diamond Belt")
	enter(t, g, in)
	typeText(t, g, in, "Oops")
	enter(t, g, in)
	press(t, g, in, ebiten.KeyBackspace)
	press(t, g, in, ebiten.KeyTab)
	typeText(t, g, in, "Weekly")
	enter(t, g, in)
	press(t, g, in, ebiten.KeyTab)
	enter(t, g, in)

	if len(save.Federations) != 1 {
		t.Fatalf("%d federations created", len(save.Federations))
	}
	fed := save.Federations[0]
	if fed.Name != "Dynamite" || len(fed.Roster) != 4 || fed.WeeklyShowName != "Weekly" {
		t.Fatalf("federation: %+v", fed)
	}
	if len(fed.Championships) != 1 || fed.Championships[0].Name != "Diamond Belt" {
		t.Fatalf("belts: %+v", fed.Championships)
	}
	if len(fed.PPVNames) != len(engine.DefaultPPVNames) {
		t.Fatalf("PPV names: %v, want the defaults", fed.PPVNames)
	}
	if _, ok := g.screen.(*CareerScreen); !ok {
		t.Fatalf("screen is %T, want the federation dashboard", g.screen)
	}
}

func TestCorruptFederationSaveShowsANotice(t *testing.T) {
	g, _ := testGame("A", "B")
	g.Store.(*memStore).career = []byte(`{"federations": [[[`)
	screen := NewFederationSelectScreen(g)
	if !strings.Contains(g.notice, "federations could not be read") {
		t.Fatalf("notice: %q", g.notice)
	}
	if len(screen.save.Federations) != 0 {
		t.Fatalf("federations: %+v", screen.save.Federations)
	}
}
