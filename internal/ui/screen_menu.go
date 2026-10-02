package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
)

var menuOptions = []string{
	"Federation",
	"Singles Match",
	"Tag Team Match",
	"Cage Match",
	"No DQ Match",
	"Feud Match",
	"Battle Royal",
	"Tournament",
	"Create New Card",
	"Edit Existing Card",
	"Settings",
}

const (
	menuCareer      = 0
	menuSingles     = 1
	menuTag         = 2
	menuCage        = 3
	menuNoDQ        = 4
	menuFeud        = 5
	menuBattleRoyal = 6
	menuTournament  = 7
	menuNewCard     = 8
	menuEditCard    = 9
	menuSettings    = 10
)

// exhibition is the kind of match a main-menu option sets up.
type exhibition struct {
	label     string
	matchType engine.MatchType
	isFeud    bool
}

var exhibitions = map[int]exhibition{
	menuSingles: {"Singles", engine.MatchSingles, false},
	menuTag:     {"Tag Team", engine.MatchTag, false},
	menuCage:    {"Cage", engine.MatchCage, false},
	menuNoDQ:    {"No DQ", engine.MatchNoDQ, false},
	menuFeud:    {"Feud", engine.MatchSingles, true},
}

type menuStep int

const (
	stepMainMenu menuStep = iota
	stepMatchSetup
	stepBattleRoyal
	stepPickCardToEdit
)

type questionKind int

const (
	askWrestler questionKind = iota
	askAlly
	askRegularPartners
)

// setupQuestion is one thing asked while setting up an exhibition match.
type setupQuestion struct {
	kind   questionKind
	prompt string
	label  string // how the answer is listed once given
}

const (
	noAlly              = -1
	answerRegular       = 0
	minBattleRoyalField = 3
)

var regularPartnerChoices = []string{"Yes: regular tag partners", "No: thrown together for this match"}

func setupQuestions(matchType engine.MatchType) []setupQuestion {
	if matchType == engine.MatchTag {
		return []setupQuestion{
			{askWrestler, "SELECT TEAM 1 - WRESTLER A:", "Team 1A"},
			{askWrestler, "SELECT TEAM 1 - WRESTLER B:", "Team 1B"},
			{askWrestler, "SELECT TEAM 2 - WRESTLER A:", "Team 2A"},
			{askWrestler, "SELECT TEAM 2 - WRESTLER B:", "Team 2B"},
			{askRegularPartners, "IS TEAM 1 A REGULAR TAG TEAM?", "Team 1 regular partners"},
			{askRegularPartners, "IS TEAM 2 A REGULAR TAG TEAM?", "Team 2 regular partners"},
		}
	}
	return []setupQuestion{
		{askWrestler, "SELECT WRESTLER 1:", "Wrestler 1"},
		{askWrestler, "SELECT WRESTLER 2:", "Wrestler 2"},
		{askAlly, "SELECT RINGSIDE ALLY FOR WRESTLER 1 (optional):", "Ally 1"},
		{askAlly, "SELECT RINGSIDE ALLY FOR WRESTLER 2 (optional):", "Ally 2"},
	}
}

// MenuScreen is the main menu and the setup steps for exhibition matches.
type MenuScreen struct {
	step   menuStep
	cursor int

	// Match setup: the questions for the chosen match and the answers so
	// far. A wrestler or ally answer is a roster index (noAlly for none); a
	// regular-partners answer is an index into regularPartnerChoices.
	match     exhibition
	questions []setupQuestion
	answers   []int

	multiSelect []bool // Battle Royal entrants
}

func NewMenuScreen() *MenuScreen {
	return &MenuScreen{}
}

func (m *MenuScreen) Update(g *Game) error {
	switch m.step {
	case stepMatchSetup:
		m.updateMatchSetup(g)
	case stepBattleRoyal:
		m.updateBattleRoyal(g)
	case stepPickCardToEdit:
		m.updatePickCard(g)
	default:
		return m.updateMainMenu(g)
	}
	return nil
}

func confirmPressed(in Input) bool {
	return in.JustPressed(ebiten.KeyEnter) || in.JustPressed(ebiten.KeySpace)
}

func (m *MenuScreen) backToMainMenu() {
	m.step = stepMainMenu
	m.cursor = 0
	m.answers = nil
	m.multiSelect = nil
}

// ─── Main menu ──────────────────────────────────────────────────────────────

func (m *MenuScreen) updateMainMenu(g *Game) error {
	if g.in.JustPressed(ebiten.KeyEscape) && g.CanQuit {
		return ebiten.Termination
	}
	m.cursor = handleListInput(g.in, m.cursor, len(menuOptions))
	if confirmPressed(g.in) {
		m.openOption(g, m.cursor)
	}
	return nil
}

func (m *MenuScreen) openOption(g *Game, option int) {
	if match, ok := exhibitions[option]; ok {
		m.match = match
		m.questions = setupQuestions(match.matchType)
		m.answers = nil
		m.step = stepMatchSetup
		m.cursor = m.firstFreeChoice(g)
		return
	}

	switch option {
	case menuCareer:
		g.SetScreen(NewFederationSelectScreen(g))
	case menuTournament:
		g.SetScreen(NewTournamentScreen(g))
	case menuNewCard:
		g.SetScreen(NewCardEditorScreen(nil))
	case menuSettings:
		g.SetScreen(NewSettingsScreen())
	case menuBattleRoyal:
		m.step = stepBattleRoyal
		m.cursor = 0
		m.multiSelect = make([]bool, len(g.Roster))
	case menuEditCard:
		m.step = stepPickCardToEdit
		m.cursor = 0
	}
}

// ─── Match setup ────────────────────────────────────────────────────────────

func (m *MenuScreen) currentQuestion() setupQuestion {
	return m.questions[len(m.answers)]
}

// choiceCount is the number of entries in the list for the current question.
func (m *MenuScreen) choiceCount(g *Game) int {
	switch m.currentQuestion().kind {
	case askAlly:
		return len(g.Roster) + 1
	case askRegularPartners:
		return len(regularPartnerChoices)
	default:
		return len(g.Roster)
	}
}

// choiceAnswer turns a list position into the answer it stands for.
func (m *MenuScreen) choiceAnswer(position int) int {
	if m.currentQuestion().kind == askAlly {
		return position - 1
	}
	return position
}

// rosterIndexTaken reports whether a wrestler is already in the match or at
// ringside, so he cannot be picked again.
func (m *MenuScreen) rosterIndexTaken(rosterIndex int) bool {
	for i, answer := range m.answers {
		if m.questions[i].kind != askRegularPartners && answer == rosterIndex {
			return true
		}
	}
	return false
}

func (m *MenuScreen) choiceAllowed(position int) bool {
	if m.currentQuestion().kind == askRegularPartners {
		return true
	}
	answer := m.choiceAnswer(position)
	return answer == noAlly || !m.rosterIndexTaken(answer)
}

func (m *MenuScreen) firstFreeChoice(g *Game) int {
	for position := 0; position < m.choiceCount(g); position++ {
		if m.choiceAllowed(position) {
			return position
		}
	}
	return 0
}

func (m *MenuScreen) updateMatchSetup(g *Game) {
	if g.in.JustPressed(ebiten.KeyEscape) {
		if len(m.answers) == 0 {
			m.backToMainMenu()
			return
		}
		m.answers = m.answers[:len(m.answers)-1]
		m.cursor = m.firstFreeChoice(g)
		return
	}

	m.cursor = handleListInput(g.in, m.cursor, m.choiceCount(g))
	if !confirmPressed(g.in) || !m.choiceAllowed(m.cursor) {
		return
	}
	m.answers = append(m.answers, m.choiceAnswer(m.cursor))
	if len(m.answers) == len(m.questions) {
		m.startMatch(g)
		return
	}
	m.cursor = m.firstFreeChoice(g)
}

func (m *MenuScreen) startMatch(g *Game) {
	var ms *MatchScreen
	if m.match.matchType == engine.MatchTag {
		ms = m.newTagMatchScreen(g)
	} else {
		ms = m.newSinglesMatchScreen(g)
	}
	ms.RunMatch(g)
	g.SetScreen(ms)
}

func (m *MenuScreen) newTagMatchScreen(g *Game) *MatchScreen {
	a := m.answers
	match := engine.NewTagMatch(g.Roster[a[0]], g.Roster[a[1]], g.Roster[a[2]], g.Roster[a[3]])
	match.Sides[0].RegularPartners = a[4] == answerRegular
	match.Sides[1].RegularPartners = a[5] == answerRegular
	return NewTagMatchScreen(match, g)
}

func (m *MenuScreen) newSinglesMatchScreen(g *Game) *MatchScreen {
	a := m.answers
	ms := NewMatchScreen(g.Roster[a[0]], g.Roster[a[1]], m.match.matchType, g)
	for side, ally := range a[2:4] {
		if ally != noAlly {
			ms.match.Sides[side].Ally = g.Roster[ally]
		}
	}
	ms.match.IsFeud = m.match.isFeud
	return ms
}

// ─── Battle Royal and card editing ──────────────────────────────────────────

func (m *MenuScreen) updateBattleRoyal(g *Game) {
	if g.in.JustPressed(ebiten.KeyEscape) {
		m.backToMainMenu()
		return
	}
	m.cursor = handleListInput(g.in, m.cursor, len(g.Roster))
	if g.in.JustPressed(ebiten.KeySpace) {
		m.multiSelect[m.cursor] = !m.multiSelect[m.cursor]
	}
	if !g.in.JustPressed(ebiten.KeyEnter) {
		return
	}
	var entrants []*engine.WrestlerCard
	for i, picked := range m.multiSelect {
		if picked {
			entrants = append(entrants, g.Roster[i])
		}
	}
	if len(entrants) >= minBattleRoyalField {
		g.SetScreen(NewBattleRoyalScreen(entrants, g))
	}
}

func (m *MenuScreen) updatePickCard(g *Game) {
	if g.in.JustPressed(ebiten.KeyEscape) {
		m.backToMainMenu()
		return
	}
	m.cursor = handleListInput(g.in, m.cursor, len(g.Roster))
	if confirmPressed(g.in) {
		g.SetScreen(NewCardEditorScreen(g.Roster[m.cursor]))
	}
}

// ─── Drawing ────────────────────────────────────────────────────────────────

func (m *MenuScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, fmt.Sprintf("                   RING WARS v%s", Version), Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	for _, line := range m.headerLines(g) {
		DrawText(screen, line, Margin, y)
		y += LineHeight
	}
	y += LineHeight
	drawList(screen, g, y, m.listItems(g), m.cursor)

	DrawText(screen, m.statusLine(g), Margin, g.screenH-LineHeight-Margin)
}

// headerLines is the text above the list: the prompt and, during match
// setup, the answers given so far.
func (m *MenuScreen) headerLines(g *Game) []string {
	switch m.step {
	case stepMatchSetup:
		lines := []string{"Match Type: " + m.match.label}
		for i, answer := range m.answers {
			lines = append(lines, m.questions[i].label+": "+m.answerText(g, i, answer))
		}
		return append(lines, "", m.currentQuestion().prompt)
	case stepBattleRoyal:
		return []string{fmt.Sprintf("BATTLE ROYAL: SELECT WRESTLERS (min %d, selected %d):", minBattleRoyalField, m.selectedCount())}
	case stepPickCardToEdit:
		return []string{"SELECT CARD TO EDIT:"}
	default:
		return []string{"MAIN MENU:"}
	}
}

func (m *MenuScreen) answerText(g *Game, question, answer int) string {
	switch {
	case m.questions[question].kind == askRegularPartners:
		return regularPartnerChoices[answer]
	case answer == noAlly:
		return "None"
	default:
		name := g.Roster[answer].Name
		return name + statusMarkers(g, name)
	}
}

func (m *MenuScreen) selectedCount() int {
	count := 0
	for _, picked := range m.multiSelect {
		if picked {
			count++
		}
	}
	return count
}

// listItems is the list the cursor moves through on the current step.
func (m *MenuScreen) listItems(g *Game) []string {
	switch m.step {
	case stepMatchSetup:
		return m.setupChoices(g)
	case stepBattleRoyal:
		return m.battleRoyalChoices(g)
	case stepPickCardToEdit:
		return rosterNames(g, nil)
	default:
		return menuOptions
	}
}

func (m *MenuScreen) setupChoices(g *Game) []string {
	switch m.currentQuestion().kind {
	case askRegularPartners:
		return regularPartnerChoices
	case askAlly:
		return append([]string{"No Ally"}, rosterNames(g, m.rosterIndexTaken)...)
	default:
		return rosterNames(g, m.rosterIndexTaken)
	}
}

func (m *MenuScreen) battleRoyalChoices(g *Game) []string {
	names := rosterNames(g, nil)
	for i := range names {
		box := "[ ] "
		if m.multiSelect[i] {
			box = "[x] "
		}
		names[i] = box + names[i]
	}
	return names
}

// rosterNames lists the roster with injury and suspension markers, and marks
// the wrestlers that taken reports as already in use.
func rosterNames(g *Game, taken func(rosterIndex int) bool) []string {
	names := make([]string, len(g.Roster))
	for i, card := range g.Roster {
		names[i] = card.Name + statusMarkers(g, card.Name)
		if taken != nil && taken(i) {
			names[i] += "  (already in this match)"
		}
	}
	return names
}

func (m *MenuScreen) statusLine(g *Game) string {
	switch {
	case m.step == stepBattleRoyal:
		return "[UP/DOWN] Move  [SPACE] Toggle  [ENTER] Confirm  [ESC] Back"
	case m.step != stepMainMenu:
		return "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Back"
	case g.CanQuit:
		return "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Quit"
	default:
		return "[UP/DOWN] Select  [ENTER] Confirm"
	}
}

func handleListInput(in Input, cursor, length int) int {
	if in.JustPressed(ebiten.KeyDown) {
		cursor++
		if cursor >= length {
			cursor = 0
		}
	}
	if in.JustPressed(ebiten.KeyUp) {
		cursor--
		if cursor < 0 {
			cursor = length - 1
		}
	}
	return cursor
}
