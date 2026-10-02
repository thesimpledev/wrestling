package ui

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
)

type BookPhase int

const (
	BookViewCard BookPhase = iota
	BookEditType
	BookEditWrestlers
	BookEditPartners
)

type CareerBookScreen struct {
	fed    *engine.Federation
	save   *engine.FederationSave
	card   []engine.BookedMatch
	cursor int
	phase  BookPhase
	roster []*engine.WrestlerCard

	// The edit in progress. Nothing is written to the card until every
	// question has been answered, so ESC leaves the match as it was.
	editIdx     int
	editCursor  int
	editType    engine.MatchType
	editPicks   []string
	editRegular []bool
}

func NewCareerBookScreen(fed *engine.Federation, save *engine.FederationSave, card []engine.BookedMatch, g *Game) *CareerBookScreen {
	return &CareerBookScreen{
		fed:    fed,
		save:   save,
		card:   card,
		roster: FilterRoster(g.Roster, fed.Roster),
	}
}

// bookableRoster leaves out suspended wrestlers, who cannot be booked.
func bookableRoster(g *Game, roster []*engine.WrestlerCard) []*engine.WrestlerCard {
	bookable := make([]*engine.WrestlerCard, 0, len(roster))
	for _, w := range roster {
		if !g.Injuries.IsSuspended(w.Name) {
			bookable = append(bookable, w)
		}
	}
	return bookable
}

var showActions = []struct {
	label string
	mode  ShowMode
}{
	{"Watch All Matches", ShowModeWatch},
	{"Simulate All Matches", ShowModeSimulate},
	{"Watch Main Event Only", ShowModeMainEvent},
}

var editableTypes = []engine.MatchType{
	engine.MatchSingles,
	engine.MatchTag,
	engine.MatchCage,
	engine.MatchNoDQ,
}

var editableTypeNames = []string{
	"Singles",
	"Tag Team",
	"Cage",
	"No DQ",
}

func wrestlersNeeded(matchType engine.MatchType) int {
	if matchType == engine.MatchTag {
		return 4
	}
	return 2
}

func (bs *CareerBookScreen) Update(g *Game) error {
	if g.in.JustPressed(ebiten.KeyEscape) {
		if bs.phase != BookViewCard {
			bs.phase = BookViewCard
			return nil
		}
		g.SetScreen(NewCareerScreen(bs.fed, bs.save))
		return nil
	}

	switch bs.phase {
	case BookViewCard:
		bs.updateViewCard(g)
	case BookEditType:
		bs.updateEditType(g)
	case BookEditWrestlers:
		bs.updateEditWrestlers(g)
	case BookEditPartners:
		bs.updateEditPartners(g)
	}
	return nil
}

func (bs *CareerBookScreen) updateViewCard(g *Game) {
	bs.cursor = handleListInput(g.in, bs.cursor, len(bs.card)+len(showActions))
	if !confirmPressed(g.in) {
		return
	}
	if bs.cursor >= len(bs.card) {
		mode := showActions[bs.cursor-len(bs.card)].mode
		g.SetScreen(NewCareerShowScreen(bs.fed, bs.save, bs.card, mode, g))
		return
	}

	match := bs.card[bs.cursor]
	if len(match.BREntrants) > 0 || match.IsTournament {
		g.SetNotice("Battle royals and tournaments cannot be edited.")
		return
	}
	bs.editIdx = bs.cursor
	bs.editCursor = 0
	bs.phase = BookEditType
}

func (bs *CareerBookScreen) updateEditType(g *Game) {
	bs.editCursor = handleListInput(g.in, bs.editCursor, len(editableTypes))
	if !confirmPressed(g.in) {
		return
	}
	bs.editPicks = nil
	bs.editRegular = nil

	chosen := editableTypes[bs.editCursor]
	if free := bs.freeCount(g); free < wrestlersNeeded(chosen) {
		g.SetNotice(fmt.Sprintf("Only %d wrestlers are free for this match: a %s match needs %d.",
			free, editableTypeNames[bs.editCursor], wrestlersNeeded(chosen)))
		return
	}
	bs.editType = chosen
	bs.editCursor = bs.firstFree(g)
	bs.phase = BookEditWrestlers
}

func (bs *CareerBookScreen) updateEditWrestlers(g *Game) {
	bs.editCursor = handleListInput(g.in, bs.editCursor, len(bs.roster))
	if !confirmPressed(g.in) {
		return
	}
	name := bs.roster[bs.editCursor].Name
	if bs.blockedReason(g, name) != "" {
		return
	}
	bs.editPicks = append(bs.editPicks, name)

	switch {
	case len(bs.editPicks) < wrestlersNeeded(bs.editType):
		bs.editCursor = bs.firstFree(g)
	case bs.editType == engine.MatchTag:
		bs.editCursor = 0
		bs.phase = BookEditPartners
	default:
		bs.applyEdit(g)
	}
}

func (bs *CareerBookScreen) updateEditPartners(g *Game) {
	bs.editCursor = handleListInput(g.in, bs.editCursor, len(regularPartnerChoices))
	if !confirmPressed(g.in) {
		return
	}
	bs.editRegular = append(bs.editRegular, bs.editCursor == answerRegular)
	bs.editCursor = 0
	if len(bs.editRegular) == 2 {
		bs.applyEdit(g)
	}
}

// bookedElsewhere returns the number of another match on the card that the
// wrestler is already in, or 0.
func (bs *CareerBookScreen) bookedElsewhere(name string) int {
	for i, match := range bs.card {
		if i == bs.editIdx {
			continue
		}
		for _, names := range [][]string{match.Side1, match.Side2, match.BREntrants, match.TournSeeds} {
			for _, booked := range names {
				if booked == name {
					return i + 1
				}
			}
		}
	}
	return 0
}

// blockedReason says why a wrestler cannot be picked for the match being
// edited, or returns "" when he can.
func (bs *CareerBookScreen) blockedReason(g *Game, name string) string {
	for _, picked := range bs.editPicks {
		if picked == name {
			return "already in this match"
		}
	}
	if g.Injuries.IsSuspended(name) {
		return "suspended"
	}
	if other := bs.bookedElsewhere(name); other > 0 {
		return fmt.Sprintf("booked in match %d", other)
	}
	return ""
}

func (bs *CareerBookScreen) freeCount(g *Game) int {
	count := 0
	for _, w := range bs.roster {
		if bs.blockedReason(g, w.Name) == "" {
			count++
		}
	}
	return count
}

func (bs *CareerBookScreen) firstFree(g *Game) int {
	for i, w := range bs.roster {
		if bs.blockedReason(g, w.Name) == "" {
			return i
		}
	}
	return 0
}

func (bs *CareerBookScreen) applyEdit(g *Game) {
	match := &bs.card[bs.editIdx]
	half := len(bs.editPicks) / 2
	match.Type = bs.editType
	match.Side1 = append([]string{}, bs.editPicks[:half]...)
	match.Side2 = append([]string{}, bs.editPicks[half:]...)
	match.Side1Regular = len(bs.editRegular) == 2 && bs.editRegular[0]
	match.Side2Regular = len(bs.editRegular) == 2 && bs.editRegular[1]

	if match.IsTitle && !bs.titleStillOnTheLine(*match) {
		match.IsTitle = false
		match.TitleIndex = -1
		g.SetNotice(fmt.Sprintf("Match %d is no longer a title match: the champion has to be in it, one on one.", bs.editIdx+1))
	}
	bs.phase = BookViewCard
}

func (bs *CareerBookScreen) titleStillOnTheLine(match engine.BookedMatch) bool {
	if match.Type == engine.MatchTag {
		return false
	}
	champion := bs.fed.ChampionOf(match.TitleIndex)
	return champion == "" || bookedIn(match, champion)
}

// ─── Drawing ────────────────────────────────────────────────────────────────

func (bs *CareerBookScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, showDivider, Margin, y)
	y += LineHeight
	title := fmt.Sprintf("  Week %d - %s", bs.fed.Week, bs.fed.ShowName())
	DrawText(screen, title, Margin, y)
	y += LineHeight
	DrawText(screen, showDivider, Margin, y)
	y += LineHeight * 2

	for _, line := range bs.headerLines() {
		DrawText(screen, line, Margin, y)
		y += LineHeight
	}
	y += LineHeight
	items, cursor := bs.listItems(g)
	drawList(screen, g, y, items, cursor)

	DrawText(screen, bs.statusLine(), Margin, g.screenH-LineHeight-Margin)
}

func (bs *CareerBookScreen) headerLines() []string {
	editing := fmt.Sprintf("EDIT MATCH %d: ", bs.editIdx+1)
	switch bs.phase {
	case BookEditType:
		return []string{editing + "SELECT TYPE:"}
	case BookEditWrestlers:
		question := setupQuestions(bs.editType)[len(bs.editPicks)]
		if len(bs.editPicks) == 0 {
			return []string{editing + question.prompt}
		}
		return []string{editing + question.prompt, "Picked so far: " + strings.Join(bs.editPicks, ", ")}
	case BookEditPartners:
		question := setupQuestions(engine.MatchTag)[wrestlersNeeded(engine.MatchTag)+len(bs.editRegular)]
		return []string{editing + question.prompt}
	default:
		return []string{"FIGHT CARD:"}
	}
}

func (bs *CareerBookScreen) listItems(g *Game) ([]string, int) {
	switch bs.phase {
	case BookEditType:
		return editableTypeNames, bs.editCursor
	case BookEditWrestlers:
		return bs.wrestlerChoices(g), bs.editCursor
	case BookEditPartners:
		return regularPartnerChoices, bs.editCursor
	default:
		return bs.cardLines(g), bs.cursor
	}
}

func (bs *CareerBookScreen) wrestlerChoices(g *Game) []string {
	choices := make([]string, len(bs.roster))
	for i, w := range bs.roster {
		choices[i] = w.Name + statusMarkers(g, w.Name)
		if reason := bs.blockedReason(g, w.Name); reason != "" && reason != "suspended" {
			choices[i] += "  (" + reason + ")"
		}
	}
	return choices
}

func (bs *CareerBookScreen) cardLines(g *Game) []string {
	lines := make([]string, 0, len(bs.card)+len(showActions))
	for i, match := range bs.card {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, bookedMatchLine(bs.fed, g, match)))
	}
	for _, action := range showActions {
		lines = append(lines, action.label)
	}
	return lines
}

func (bs *CareerBookScreen) statusLine() string {
	if bs.phase == BookViewCard {
		return "[UP/DOWN] Select  [ENTER] Edit/Start  [ESC] Back  (* injured, (c) champion)"
	}
	return "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Cancel"
}

// bookedMatchLine describes one match on the card.
func bookedMatchLine(fed *engine.Federation, g *Game, match engine.BookedMatch) string {
	title := ""
	if match.IsTitle && match.TitleIndex >= 0 && match.TitleIndex < len(fed.Championships) {
		title = fed.Championships[match.TitleIndex].Name
	}
	switch {
	case len(match.BREntrants) > 0:
		return fmt.Sprintf("[BATTLE ROYAL] %d-man Battle Royal", len(match.BREntrants))
	case match.IsTournament && title != "":
		return fmt.Sprintf("[%s TOURNAMENT] %d-man Tournament", title, match.TournSize)
	case match.IsTournament:
		return fmt.Sprintf("[TOURNAMENT] %d-man Tournament", match.TournSize)
	}

	champion := ""
	if title != "" {
		champion = fed.ChampionOf(match.TitleIndex)
		title += " - "
	}
	feud := ""
	if match.Type != engine.MatchTag && len(match.Side1) > 0 && len(match.Side2) > 0 && fed.IsRival(match.Side1[0], match.Side2[0]) {
		feud = " [FEUD]"
	}
	return fmt.Sprintf("[%s%s] %s vs %s%s", title, engine.MatchTypeString(match.Type),
		sideText(g, match.Side1, champion), sideText(g, match.Side2, champion), feud)
}

func sideText(g *Game, names []string, champion string) string {
	if len(names) == 0 {
		return "TBD"
	}
	marked := make([]string, len(names))
	for i, name := range names {
		marked[i] = name
		if g.Injuries.IsInjured(name) {
			marked[i] += "*"
		}
		if champion != "" && name == champion {
			marked[i] += " (c)"
		}
	}
	return strings.Join(marked, " & ")
}
