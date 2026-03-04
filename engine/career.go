package engine

import (
	"math/rand"
	"sort"
)

// ─── TYPES ──────────────────────────────────────────────────────────────────

type WrestlerRecord struct {
	Wins           int `json:"wins"`
	Losses         int `json:"losses"`
	Draws          int `json:"draws"`
	WinsByPin      int `json:"wins_by_pin"`
	WinsByDQ       int `json:"wins_by_dq"`
	WinsByCountout int `json:"wins_by_countout"`
	TitleReigns    int `json:"title_reigns"`
	CurrentStreak  int `json:"current_streak"` // positive = wins, negative = losses
}

type TitleChange struct {
	Week   int    `json:"week"`
	Winner string `json:"winner"`
	Loser  string `json:"loser"`
	Method string `json:"method"` // "pinfall", "tournament", "vacated", etc.
}

type Championship struct {
	Name         string        `json:"name"`
	Champion     string        `json:"champion"`      // "" if vacant
	DefensesLeft int           `json:"defenses_left"` // weeks until mandatory defense
	History      []TitleChange `json:"history"`
}

type MatchHistoryEntry struct {
	Week      int    `json:"week"`
	Winner    string `json:"winner"`
	Loser     string `json:"loser"`
	Method    string `json:"method"`
	MatchType string `json:"match_type"`
	IsTitle   bool   `json:"is_title"`
}

// BookedMatch represents a single match on a fight card.
type BookedMatch struct {
	Type       MatchType        `json:"type"`
	IsTitle    bool             `json:"is_title"`
	Side1      []string         `json:"side1"`       // wrestler names
	Side2      []string         `json:"side2"`       // wrestler names
	BREntrants []string         `json:"br_entrants"` // battle royal only
	IsTournament bool           `json:"is_tournament"`
	TournSize    int            `json:"tourn_size"`
	TournSeeds   []string       `json:"tourn_seeds"`
}

// CareerSave holds the entire career state.
type CareerSave struct {
	Week          int                          `json:"week"`
	Records       map[string]*WrestlerRecord   `json:"records"`
	Championships []Championship               `json:"championships"`
	Rivalries     map[string]map[string]int    `json:"rivalries"`
	MatchHistory  []MatchHistoryEntry          `json:"match_history"`
	PPVNames      []string                     `json:"ppv_names"`
	PPVIndex      int                          `json:"ppv_index"`
	TitleShotEarned string                     `json:"title_shot_earned"` // BR/tournament winner
}

// ─── PPV NAMES ──────────────────────────────────────────────────────────────

var DefaultPPVNames = []string{
	"SLAM FEST",
	"RING WARS RUMBLE",
	"SHOWDOWN",
	"WAR GAMES",
	"TOTAL MAYHEM",
	"STEEL CAGE FURY",
	"CHAMPIONSHIP CLASH",
	"BATTLE LINES",
	"VENDETTA",
	"KING OF THE RING",
	"FINAL STAND",
	"WRESTLE WARS",
}

// ─── CONSTRUCTOR ────────────────────────────────────────────────────────────

func NewCareer(rosterNames []string) *CareerSave {
	records := make(map[string]*WrestlerRecord, len(rosterNames))
	for _, name := range rosterNames {
		records[name] = &WrestlerRecord{}
	}

	return &CareerSave{
		Week:    1,
		Records: records,
		Championships: []Championship{
			{
				Name:         "World Heavyweight Championship",
				Champion:     "",
				DefensesLeft: 4, // First PPV will have inaugural tournament
			},
		},
		Rivalries:    make(map[string]map[string]int),
		MatchHistory: []MatchHistoryEntry{},
		PPVNames:     DefaultPPVNames,
		PPVIndex:     0,
	}
}

// ─── QUERIES ────────────────────────────────────────────────────────────────

// IsPPV returns true if the current week is a PPV week (every 4 weeks).
func (c *CareerSave) IsPPV() bool {
	return c.Week%4 == 0
}

// CurrentPPVName returns the name of the current PPV (only meaningful on PPV weeks).
func (c *CareerSave) CurrentPPVName() string {
	if len(c.PPVNames) == 0 {
		return "PAY-PER-VIEW"
	}
	return c.PPVNames[c.PPVIndex%len(c.PPVNames)]
}

// ShowName returns a display name for the current week's show.
func (c *CareerSave) ShowName() string {
	if c.IsPPV() {
		return c.CurrentPPVName()
	}
	return "RING WARS WEEKLY"
}

// WorldChampion returns the current world champion name, or "" if vacant.
func (c *CareerSave) WorldChampion() string {
	if len(c.Championships) > 0 {
		return c.Championships[0].Champion
	}
	return ""
}

// RankedWrestlers returns wrestler names sorted by win percentage (min 3 matches).
func (c *CareerSave) RankedWrestlers() []string {
	type ranked struct {
		name   string
		winPct float64
		wins   int
	}
	var list []ranked
	for name, rec := range c.Records {
		total := rec.Wins + rec.Losses + rec.Draws
		if total < 3 {
			continue
		}
		pct := float64(rec.Wins) / float64(total)
		list = append(list, ranked{name, pct, rec.Wins})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].winPct != list[j].winPct {
			return list[i].winPct > list[j].winPct
		}
		return list[i].wins > list[j].wins
	})
	names := make([]string, len(list))
	for i, r := range list {
		names[i] = r.name
	}
	return names
}

// TopContender returns the #1 ranked wrestler who isn't the champion.
func (c *CareerSave) TopContender() string {
	champ := c.WorldChampion()
	for _, name := range c.RankedWrestlers() {
		if name != champ {
			return name
		}
	}
	return ""
}

// RivalryScore returns the rivalry score between two wrestlers.
func (c *CareerSave) RivalryScore(a, b string) int {
	if m, ok := c.Rivalries[a]; ok {
		return m[b]
	}
	return 0
}

// IsRival returns true if two wrestlers have an active rivalry (score >= 3).
func (c *CareerSave) IsRival(a, b string) bool {
	return c.RivalryScore(a, b) >= 3
}

// ActiveRivals returns all active rivalry pairs.
func (c *CareerSave) ActiveRivals() [][2]string {
	seen := make(map[[2]string]bool)
	var pairs [][2]string
	for a, m := range c.Rivalries {
		for b, score := range m {
			if score < 3 {
				continue
			}
			pair := [2]string{a, b}
			if a > b {
				pair = [2]string{b, a}
			}
			if !seen[pair] {
				seen[pair] = true
				pairs = append(pairs, pair)
			}
		}
	}
	return pairs
}

// ─── RECORD UPDATES ─────────────────────────────────────────────────────────

// RecordResult updates win/loss records after a match.
func (c *CareerSave) RecordResult(winner, loser, method string, isTitle bool) {
	c.ensureRecord(winner)
	c.ensureRecord(loser)

	wr := c.Records[winner]
	wr.Wins++
	if wr.CurrentStreak > 0 {
		wr.CurrentStreak++
	} else {
		wr.CurrentStreak = 1
	}
	switch method {
	case "pinfall":
		wr.WinsByPin++
	case "dq":
		wr.WinsByDQ++
	case "countout":
		wr.WinsByCountout++
	}

	lr := c.Records[loser]
	lr.Losses++
	if lr.CurrentStreak < 0 {
		lr.CurrentStreak--
	} else {
		lr.CurrentStreak = -1
	}

	// Add to match history (keep last 100)
	entry := MatchHistoryEntry{
		Week:    c.Week,
		Winner:  winner,
		Loser:   loser,
		Method:  method,
		IsTitle: isTitle,
	}
	c.MatchHistory = append(c.MatchHistory, entry)
	if len(c.MatchHistory) > 100 {
		c.MatchHistory = c.MatchHistory[len(c.MatchHistory)-100:]
	}
}

// RecordDraw updates records for a draw.
func (c *CareerSave) RecordDraw(a, b string) {
	c.ensureRecord(a)
	c.ensureRecord(b)
	c.Records[a].Draws++
	c.Records[a].CurrentStreak = 0
	c.Records[b].Draws++
	c.Records[b].CurrentStreak = 0
}

func (c *CareerSave) ensureRecord(name string) {
	if _, ok := c.Records[name]; !ok {
		c.Records[name] = &WrestlerRecord{}
	}
}

// ─── RIVALRY UPDATES ────────────────────────────────────────────────────────

// AddRivalry adds points to the rivalry between two wrestlers (bidirectional).
func (c *CareerSave) AddRivalry(a, b string, points int) {
	if c.Rivalries[a] == nil {
		c.Rivalries[a] = make(map[string]int)
	}
	if c.Rivalries[b] == nil {
		c.Rivalries[b] = make(map[string]int)
	}
	c.Rivalries[a][b] += points
	c.Rivalries[b][a] += points
}

// DecayRivalries reduces all rivalry scores by 1 (called each PPV cycle).
func (c *CareerSave) DecayRivalries() {
	for a, m := range c.Rivalries {
		for b, score := range m {
			if score <= 1 {
				delete(m, b)
			} else {
				m[b] = score - 1
			}
		}
		if len(m) == 0 {
			delete(c.Rivalries, a)
		}
	}
}

// ─── CHAMPIONSHIP UPDATES ───────────────────────────────────────────────────

// TitleChange records a title change.
func (c *CareerSave) ChangeTitleHolder(winner, loser, method string) {
	if len(c.Championships) == 0 {
		return
	}
	ch := &c.Championships[0]
	ch.Champion = winner
	ch.DefensesLeft = 4
	ch.History = append(ch.History, TitleChange{
		Week:   c.Week,
		Winner: winner,
		Loser:  loser,
		Method: method,
	})
	c.ensureRecord(winner)
	c.Records[winner].TitleReigns++
}

// VacateTitle vacates the championship.
func (c *CareerSave) VacateTitle() {
	if len(c.Championships) == 0 {
		return
	}
	ch := &c.Championships[0]
	ch.History = append(ch.History, TitleChange{
		Week:   c.Week,
		Winner: "",
		Loser:  ch.Champion,
		Method: "vacated",
	})
	ch.Champion = ""
	ch.DefensesLeft = 4
}

// ─── WEEK ADVANCE ───────────────────────────────────────────────────────────

// AdvanceWeek increments the week counter and handles PPV rotation.
func (c *CareerSave) AdvanceWeek() {
	c.Week++
	if len(c.Championships) > 0 {
		c.Championships[0].DefensesLeft--
	}
	// On PPV weeks, decay rivalries and advance PPV name
	if c.IsPPV() {
		c.DecayRivalries()
		c.PPVIndex++
	}
}

// ─── AUTO-BOOKING ───────────────────────────────────────────────────────────

// AutoBook generates a fight card for the current week.
func (c *CareerSave) AutoBook(roster []*WrestlerCard) []BookedMatch {
	var card []BookedMatch
	used := make(map[string]bool)
	isPPV := c.IsPPV()
	champ := c.WorldChampion()

	maxMatches := 4
	if isPPV {
		maxMatches = 6
	}

	// 1. Title match at PPV
	if isPPV && champ != "" {
		contender := c.TopContender()
		// If someone earned a title shot (BR/tournament winner), they get priority
		if c.TitleShotEarned != "" && c.TitleShotEarned != champ {
			contender = c.TitleShotEarned
			c.TitleShotEarned = ""
		}
		if contender != "" {
			matchType := MatchSingles
			// Cage match for heated rivalries
			if c.RivalryScore(champ, contender) >= 5 {
				matchType = MatchCage
			}
			card = append(card, BookedMatch{
				Type:    matchType,
				IsTitle: true,
				Side1:   []string{champ},
				Side2:   []string{contender},
			})
			used[champ] = true
			used[contender] = true
		}
	}

	// 2. Vacant title → tournament
	if isPPV && champ == "" {
		size := 4
		if len(roster) >= 8 {
			size = 8
		}
		seeds := c.pickTournamentSeeds(roster, size, used)
		if len(seeds) == size {
			names := make([]string, size)
			for i, w := range seeds {
				names[i] = w.Name
				used[w.Name] = true
			}
			card = append(card, BookedMatch{
				IsTournament: true,
				TournSize:    size,
				TournSeeds:   names,
				IsTitle:      true,
			})
		}
	}

	// 3. Book rivalry matches
	for _, pair := range c.ActiveRivals() {
		if len(card) >= maxMatches {
			break
		}
		if used[pair[0]] || used[pair[1]] {
			continue
		}
		matchType := MatchSingles
		score := c.RivalryScore(pair[0], pair[1])
		if score >= 5 {
			matchType = MatchNoDQ
		}
		card = append(card, BookedMatch{
			Type:  matchType,
			Side1: []string{pair[0]},
			Side2: []string{pair[1]},
		})
		used[pair[0]] = true
		used[pair[1]] = true
	}

	// 4. PPV battle royal with unused wrestlers
	if isPPV && len(card) < maxMatches {
		var brEntrants []string
		for _, w := range roster {
			if !used[w.Name] {
				brEntrants = append(brEntrants, w.Name)
			}
		}
		if len(brEntrants) >= 4 {
			// Cap at 8
			if len(brEntrants) > 8 {
				rand.Shuffle(len(brEntrants), func(i, j int) {
					brEntrants[i], brEntrants[j] = brEntrants[j], brEntrants[i]
				})
				brEntrants = brEntrants[:8]
			}
			card = append(card, BookedMatch{
				Type:       MatchSingles, // BR uses singles sub-matches
				BREntrants: brEntrants,
			})
			for _, name := range brEntrants {
				used[name] = true
			}
		}
	}

	// 5. Fill remaining with singles matches from unused wrestlers
	var available []string
	for _, w := range roster {
		if !used[w.Name] {
			available = append(available, w.Name)
		}
	}
	rand.Shuffle(len(available), func(i, j int) {
		available[i], available[j] = available[j], available[i]
	})

	for i := 0; i+1 < len(available) && len(card) < maxMatches; i += 2 {
		card = append(card, BookedMatch{
			Type:  MatchSingles,
			Side1: []string{available[i]},
			Side2: []string{available[i+1]},
		})
	}

	return card
}

func (c *CareerSave) pickTournamentSeeds(roster []*WrestlerCard, size int, used map[string]bool) []*WrestlerCard {
	// Prefer top-ranked wrestlers
	ranked := c.RankedWrestlers()
	var seeds []*WrestlerCard
	rosterMap := make(map[string]*WrestlerCard, len(roster))
	for _, w := range roster {
		rosterMap[w.Name] = w
	}

	for _, name := range ranked {
		if len(seeds) >= size {
			break
		}
		if used[name] {
			continue
		}
		if w, ok := rosterMap[name]; ok {
			seeds = append(seeds, w)
		}
	}

	// Fill remaining from roster randomly
	if len(seeds) < size {
		var remaining []*WrestlerCard
		seedSet := make(map[string]bool)
		for _, s := range seeds {
			seedSet[s.Name] = true
		}
		for _, w := range roster {
			if !seedSet[w.Name] && !used[w.Name] {
				remaining = append(remaining, w)
			}
		}
		rand.Shuffle(len(remaining), func(i, j int) {
			remaining[i], remaining[j] = remaining[j], remaining[i]
		})
		for _, w := range remaining {
			if len(seeds) >= size {
				break
			}
			seeds = append(seeds, w)
		}
	}

	return seeds
}

// MatchTypeString returns a display string for a match type.
func MatchTypeString(mt MatchType) string {
	switch mt {
	case MatchSingles:
		return "SINGLES"
	case MatchTag:
		return "TAG TEAM"
	case MatchCage:
		return "CAGE"
	case MatchNoDQ:
		return "NO DQ"
	default:
		return "SINGLES"
	}
}
