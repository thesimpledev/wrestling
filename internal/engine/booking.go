package engine

import "math/rand"

const (
	weeklyUndercard = 3
	ppvUndercard    = 5

	battleRoyalMinField = 4
	battleRoyalMaxField = 8

	smallTournament = 4
	largeTournament = 8

	// A rivalry this heated is settled in a No DQ match, or in a cage when
	// the main title is on the line.
	grudgeMatchScore = 5
)

// booker builds one fight card. It books only the wrestlers it was given and
// never books a wrestler twice.
type booker struct {
	fed       *Federation
	roster    []string
	bookable  map[string]bool
	used      map[string]bool
	card      []BookedMatch
	undercard int
}

func newBooker(fed *Federation, roster []*WrestlerCard) *booker {
	b := &booker{
		fed:       fed,
		roster:    make([]string, 0, len(roster)),
		bookable:  make(map[string]bool, len(roster)),
		used:      make(map[string]bool, len(roster)),
		undercard: weeklyUndercard,
	}
	for _, w := range roster {
		b.roster = append(b.roster, w.Name)
		b.bookable[w.Name] = true
	}
	if fed.IsPPV() {
		b.undercard = ppvUndercard
	}
	return b
}

// AutoBook generates a fight card for the current week from the wrestlers
// given. On a PPV the main title match is settled first and runs last, and
// every other title is booked before the rest of the card is filled.
func (c *Federation) AutoBook(roster []*WrestlerCard) []BookedMatch {
	b := newBooker(c, roster)
	if !c.IsPPV() {
		b.bookRivalries()
		b.bookFiller()
		return b.card
	}

	mainEvent, hasMainEvent := b.mainEvent()
	b.bookSecondaryTitles()
	b.bookRivalries()
	b.bookBattleRoyal()
	b.bookFiller()
	if hasMainEvent {
		b.card = append(b.card, mainEvent)
	}
	return b.card
}

func (b *booker) free(name string) bool {
	return b.bookable[name] && !b.used[name]
}

func (b *booker) reserve(match BookedMatch) {
	for _, names := range [][]string{match.Side1, match.Side2, match.BREntrants, match.TournSeeds} {
		for _, name := range names {
			b.used[name] = true
		}
	}
}

func (b *booker) add(match BookedMatch) {
	b.reserve(match)
	b.card = append(b.card, match)
}

func (b *booker) freeRanked() []string {
	var names []string
	for _, name := range b.fed.RankedWrestlers() {
		if b.free(name) {
			names = append(names, name)
		}
	}
	return names
}

func (b *booker) freeUnranked() []string {
	ranked := make(map[string]bool)
	for _, name := range b.fed.RankedWrestlers() {
		ranked[name] = true
	}
	var names []string
	for _, name := range b.roster {
		if b.free(name) && !ranked[name] {
			names = append(names, name)
		}
	}
	return names
}

func (b *booker) freeWrestlers() []string {
	return append(b.freeRanked(), b.freeUnranked()...)
}

// challengersFirst moves the holders of titles to the back of the line, so
// that a champion is only pulled away from his own defense when nobody else
// is left.
func (b *booker) challengersFirst(names []string) []string {
	ordered := make([]string, 0, len(names))
	var champions []string
	for _, name := range names {
		if b.fed.HoldsTitle(name) {
			champions = append(champions, name)
		} else {
			ordered = append(ordered, name)
		}
	}
	return append(ordered, champions...)
}

func (b *booker) topContender() string {
	contenders := b.challengersFirst(b.freeWrestlers())
	if len(contenders) == 0 {
		return ""
	}
	return contenders[0]
}

func (b *booker) mainEvent() (BookedMatch, bool) {
	if len(b.fed.Championships) == 0 {
		return BookedMatch{}, false
	}
	champ := b.fed.MainChampion()
	if champ == "" {
		return b.titleTournament()
	}
	if !b.free(champ) {
		return BookedMatch{}, false
	}
	b.used[champ] = true

	contender := b.fed.TitleShotEarned
	if !b.free(contender) {
		contender = b.topContender()
	}
	if contender == "" {
		b.used[champ] = false
		return BookedMatch{}, false
	}

	match := BookedMatch{Type: MatchSingles, IsTitle: true, Side1: []string{champ}, Side2: []string{contender}}
	if b.fed.RivalryScore(champ, contender) >= grudgeMatchScore {
		match.Type = MatchCage
	}
	b.reserve(match)
	return match, true
}

func (b *booker) titleTournament() (BookedMatch, bool) {
	size := b.tournamentSize()
	if size == 0 {
		return BookedMatch{}, false
	}
	match := BookedMatch{
		IsTournament: true,
		TournSize:    size,
		TournSeeds:   b.tournamentSeeds(size),
		IsTitle:      true,
	}
	b.reserve(match)
	return match, true
}

// tournamentSize picks the large bracket only when that still leaves two
// wrestlers for each of the other titles.
func (b *booker) tournamentSize() int {
	free := len(b.freeWrestlers())
	heldBack := 2 * (len(b.fed.Championships) - 1)
	switch {
	case free-heldBack >= largeTournament:
		return largeTournament
	case free >= smallTournament:
		return smallTournament
	default:
		return 0
	}
}

func (b *booker) tournamentSeeds(size int) []string {
	unranked := b.freeUnranked()
	rand.Shuffle(len(unranked), func(i, j int) { // #nosec G404 -- tournament seed fill order, not security-sensitive
		unranked[i], unranked[j] = unranked[j], unranked[i]
	})
	return b.challengersFirst(append(b.freeRanked(), unranked...))[:size]
}

func (b *booker) bookSecondaryTitles() {
	for i := 1; i < len(b.fed.Championships); i++ {
		first := b.fed.ChampionOf(i)
		if first == "" {
			first = b.topContender()
		}
		if !b.free(first) {
			continue
		}
		b.used[first] = true
		second := b.topContender()
		if second == "" {
			b.used[first] = false
			continue
		}
		b.add(BookedMatch{Type: MatchSingles, IsTitle: true, TitleIndex: i, Side1: []string{first}, Side2: []string{second}})
	}
}

func (b *booker) bookRivalries() {
	for _, pair := range b.fed.ActiveRivals() {
		if len(b.card) >= b.undercard {
			return
		}
		if !b.free(pair[0]) || !b.free(pair[1]) {
			continue
		}
		match := BookedMatch{Type: MatchSingles, TitleIndex: -1, Side1: []string{pair[0]}, Side2: []string{pair[1]}}
		if b.fed.RivalryScore(pair[0], pair[1]) >= grudgeMatchScore {
			match.Type = MatchNoDQ
		}
		b.add(match)
	}
}

func (b *booker) bookBattleRoyal() {
	entrants := b.freeWrestlers()
	if len(b.card) >= b.undercard || len(entrants) < battleRoyalMinField {
		return
	}
	rand.Shuffle(len(entrants), func(i, j int) { // #nosec G404 -- battle royal entrant pick, not security-sensitive
		entrants[i], entrants[j] = entrants[j], entrants[i]
	})
	if len(entrants) > battleRoyalMaxField {
		entrants = entrants[:battleRoyalMaxField]
	}
	b.add(BookedMatch{Type: MatchSingles, TitleIndex: -1, BREntrants: entrants})
}

func (b *booker) bookFiller() {
	available := b.freeWrestlers()
	rand.Shuffle(len(available), func(i, j int) { // #nosec G404 -- match booking order, not security-sensitive
		available[i], available[j] = available[j], available[i]
	})
	for i := 0; i+1 < len(available) && len(b.card) < b.undercard; i += 2 {
		b.add(BookedMatch{Type: MatchSingles, TitleIndex: -1, Side1: []string{available[i]}, Side2: []string{available[i+1]}})
	}
}
