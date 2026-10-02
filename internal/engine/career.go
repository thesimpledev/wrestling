package engine

import (
	"encoding/json"
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
	Name     string `json:"name"`
	Champion string `json:"champion"` // "" if vacant

	// ContestedSincePPV is set by a title match and cleared when a PPV ends.
	// A champion who reaches the end of a PPV without it is stripped.
	ContestedSincePPV bool `json:"contested_since_ppv"`

	History []TitleChange `json:"history"`
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
	Type         MatchType `json:"type"`
	IsTitle      bool      `json:"is_title"`
	TitleIndex   int       `json:"title_index"` // -1 = not title, 0+ = championship index
	Side1        []string  `json:"side1"`       // wrestler names
	Side2        []string  `json:"side2"`       // wrestler names
	BREntrants   []string  `json:"br_entrants"` // battle royal only
	IsTournament bool      `json:"is_tournament"`
	TournSize    int       `json:"tourn_size"`
	TournSeeds   []string  `json:"tourn_seeds"`

	// Tag matches only: whether each side is a regular tag team.
	Side1Regular bool `json:"side1_regular"`
	Side2Regular bool `json:"side2_regular"`
}

// Federation holds the entire career state for one federation.
type Federation struct {
	Name            string                     `json:"name"`
	Roster          []string                   `json:"roster"`           // wrestler names in this fed
	WeeklyShowName  string                     `json:"weekly_show_name"` // e.g. "Monday Night Raw"
	PPVFrequency    int                        `json:"ppv_frequency"`    // PPV every N weeks (default 4)
	Week            int                        `json:"week"`
	Records         map[string]*WrestlerRecord `json:"records"`
	Championships   []Championship             `json:"championships"`
	Rivalries       map[string]map[string]int  `json:"rivalries"`
	MatchHistory    []MatchHistoryEntry        `json:"match_history"`
	PPVNames        []string                   `json:"ppv_names"`
	PPVIndex        int                        `json:"ppv_index"`
	TitleShotEarned string                     `json:"title_shot_earned"` // BR/tournament winner
}

// FederationSave is the top-level save structure containing all federations.
type FederationSave struct {
	Federations []*Federation `json:"federations"`
	ActiveIndex int           `json:"active_index"`
}

// ActiveFederation returns the currently active federation.
func (fs *FederationSave) ActiveFederation() *Federation {
	if fs.ActiveIndex < 0 || fs.ActiveIndex >= len(fs.Federations) {
		return nil
	}
	return fs.Federations[fs.ActiveIndex]
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

// ─── CONSTRUCTORS ───────────────────────────────────────────────────────────

// FederationConfig holds the creation parameters for a new federation.
type FederationConfig struct {
	Name           string
	RosterNames    []string
	ChampNames     []string
	WeeklyShowName string
	PPVFrequency   int
	PPVNames       []string
}

// NewFederation creates a new federation from the given config.
func NewFederation(cfg FederationConfig) *Federation {
	records := make(map[string]*WrestlerRecord, len(cfg.RosterNames))
	for _, n := range cfg.RosterNames {
		records[n] = &WrestlerRecord{}
	}

	champs := make([]Championship, len(cfg.ChampNames))
	for i, cn := range cfg.ChampNames {
		champs[i] = Championship{Name: cn}
	}

	roster := make([]string, len(cfg.RosterNames))
	copy(roster, cfg.RosterNames)

	weeklyName := cfg.WeeklyShowName
	if weeklyName == "" {
		weeklyName = "RING WARS WEEKLY"
	}

	ppvFreq := cfg.PPVFrequency
	if ppvFreq < 2 {
		ppvFreq = 4
	}

	ppvNames := cfg.PPVNames
	if len(ppvNames) == 0 {
		ppvNames = DefaultPPVNames
	}

	return &Federation{
		Name:           cfg.Name,
		Roster:         roster,
		WeeklyShowName: weeklyName,
		PPVFrequency:   ppvFreq,
		Week:           1,
		Records:        records,
		Championships:  champs,
		Rivalries:      make(map[string]map[string]int),
		MatchHistory:   []MatchHistoryEntry{},
		PPVNames:       ppvNames,
		PPVIndex:       0,
	}
}

// ─── QUERIES ────────────────────────────────────────────────────────────────

// IsPPV returns true if the current week is a PPV week.
func (c *Federation) IsPPV() bool {
	freq := c.PPVFrequency
	if freq < 2 {
		freq = 4
	}
	return c.Week%freq == 0
}

// CurrentPPVName returns the name of the current PPV (only meaningful on PPV weeks).
func (c *Federation) CurrentPPVName() string {
	if len(c.PPVNames) == 0 {
		return "PAY-PER-VIEW"
	}
	return c.PPVNames[c.PPVIndex%len(c.PPVNames)]
}

// ShowName returns a display name for the current week's show.
func (c *Federation) ShowName() string {
	if c.IsPPV() {
		return c.CurrentPPVName()
	}
	if c.WeeklyShowName != "" {
		return c.WeeklyShowName
	}
	return "RING WARS WEEKLY"
}

// MainChampion returns the main (index 0) champion name, or "" if vacant.
func (c *Federation) MainChampion() string {
	if len(c.Championships) > 0 {
		return c.Championships[0].Champion
	}
	return ""
}

// ChampionOf returns the champion for the given championship index.
func (c *Federation) ChampionOf(idx int) string {
	if idx >= 0 && idx < len(c.Championships) {
		return c.Championships[idx].Champion
	}
	return ""
}

// RankedWrestlers returns wrestler names sorted by win percentage (min 3 matches).
func (c *Federation) RankedWrestlers() []string {
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

// TopContenderFor returns the #1 ranked wrestler who isn't the champion for the given title.
func (c *Federation) TopContenderFor(champIdx int) string {
	champ := c.ChampionOf(champIdx)
	for _, name := range c.RankedWrestlers() {
		if name != champ {
			return name
		}
	}
	return ""
}

// RivalryScore returns the rivalry score between two wrestlers.
func (c *Federation) RivalryScore(a, b string) int {
	if m, ok := c.Rivalries[a]; ok {
		return m[b]
	}
	return 0
}

// IsRival returns true if two wrestlers have an active rivalry (score >= 3).
func (c *Federation) IsRival(a, b string) bool {
	return c.RivalryScore(a, b) >= 3
}

// ActiveRivals returns all active rivalry pairs.
func (c *Federation) ActiveRivals() [][2]string {
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

const matchHistoryLimit = 100

// RecordResult updates the win and loss records after a match and adds the
// match to the history under the current week.
func (c *Federation) RecordResult(entry MatchHistoryEntry) {
	c.ensureRecord(entry.Winner)
	c.ensureRecord(entry.Loser)

	wr := c.Records[entry.Winner]
	wr.Wins++
	if wr.CurrentStreak > 0 {
		wr.CurrentStreak++
	} else {
		wr.CurrentStreak = 1
	}
	switch entry.Method {
	case "pinfall":
		wr.WinsByPin++
	case "dq":
		wr.WinsByDQ++
	case "countout":
		wr.WinsByCountout++
	}

	lr := c.Records[entry.Loser]
	lr.Losses++
	if lr.CurrentStreak < 0 {
		lr.CurrentStreak--
	} else {
		lr.CurrentStreak = -1
	}

	entry.Week = c.Week
	c.MatchHistory = append(c.MatchHistory, entry)
	if len(c.MatchHistory) > matchHistoryLimit {
		c.MatchHistory = c.MatchHistory[len(c.MatchHistory)-matchHistoryLimit:]
	}
}

// RecordDraw updates records for a draw.
func (c *Federation) RecordDraw(a, b string) {
	c.ensureRecord(a)
	c.ensureRecord(b)
	c.Records[a].Draws++
	c.Records[a].CurrentStreak = 0
	c.Records[b].Draws++
	c.Records[b].CurrentStreak = 0
}

func (c *Federation) ensureRecord(name string) {
	if _, ok := c.Records[name]; !ok {
		c.Records[name] = &WrestlerRecord{}
	}
}

// ─── RIVALRY UPDATES ────────────────────────────────────────────────────────

// AddRivalry adds points to the rivalry between two wrestlers (bidirectional).
func (c *Federation) AddRivalry(a, b string, points int) {
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
func (c *Federation) DecayRivalries() {
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

// ChangeTitleHolder records a title change for the given championship index.
func (c *Federation) ChangeTitleHolder(champIdx int, winner, loser, method string) {
	if champIdx < 0 || champIdx >= len(c.Championships) {
		return
	}
	ch := &c.Championships[champIdx]
	ch.Champion = winner
	ch.ContestedSincePPV = true
	ch.History = append(ch.History, TitleChange{
		Week:   c.Week,
		Winner: winner,
		Loser:  loser,
		Method: method,
	})
	c.ensureRecord(winner)
	c.Records[winner].TitleReigns++
}

// VacateTitle vacates the championship at the given index.
func (c *Federation) VacateTitle(champIdx int) {
	if champIdx < 0 || champIdx >= len(c.Championships) {
		return
	}
	ch := &c.Championships[champIdx]
	ch.History = append(ch.History, TitleChange{
		Week:   c.Week,
		Winner: "",
		Loser:  ch.Champion,
		Method: "vacated",
	})
	ch.Champion = ""
}

// RecordTitleMatch notes that the championship was on the line in a match,
// which is what a champion needs by the end of each PPV to keep the title.
func (c *Federation) RecordTitleMatch(champIdx int) {
	if champIdx < 0 || champIdx >= len(c.Championships) {
		return
	}
	c.Championships[champIdx].ContestedSincePPV = true
}

// HoldsTitle reports whether the wrestler is the champion of any title.
func (c *Federation) HoldsTitle(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range c.Championships {
		if ch.Champion == name {
			return true
		}
	}
	return false
}

// ─── WEEK ADVANCE ───────────────────────────────────────────────────────────

// AdvanceWeek closes the current week's show and moves to the next week.
func (c *Federation) AdvanceWeek() {
	if c.IsPPV() {
		c.closePPV()
	}
	c.Week++
	if c.IsPPV() {
		c.DecayRivalries()
	}
}

// closePPV strips every champion whose title was not on the line since the
// last PPV, then starts a new cycle with the next PPV name.
func (c *Federation) closePPV() {
	for i := range c.Championships {
		ch := &c.Championships[i]
		if ch.Champion != "" && !ch.ContestedSincePPV {
			c.VacateTitle(i)
		}
		ch.ContestedSincePPV = false
	}
	c.PPVIndex++
}

// Clone returns a copy of the federation that shares nothing with it.
func (c *Federation) Clone() (*Federation, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var clone Federation
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, err
	}
	return &clone, nil
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
