package ui

import (
	"fmt"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
)

type CreatePhase int

const (
	CreatePhaseName CreatePhase = iota
	CreatePhaseRoster
	CreatePhaseChampionships
	CreatePhaseSchedule
	CreatePhasePPVNames
	CreatePhaseConfirm
)

const minFederationRoster = 4

type FederationCreateScreen struct {
	save  *engine.FederationSave
	phase CreatePhase

	// Phase 1: Name
	nameInput *TextInput

	// Phase 2: Roster
	rosterCursor int
	rosterSelect []bool

	// Phase 3: Championships
	beltInput *TextInput
	belts     []string

	// Phase 4: Schedule
	showNameInput *TextInput
	ppvFreqInput  *TextInput

	// Phase 5: PPV Names
	ppvNameInput *TextInput
	ppvNames     []string
}

func NewFederationCreateScreen(save *engine.FederationSave) *FederationCreateScreen {
	return &FederationCreateScreen{
		save:          save,
		nameInput:     NewTextInput(30),
		beltInput:     NewTextInput(40),
		showNameInput: NewTextInput(30),
		ppvFreqInput:  NewTextInput(2),
		ppvNameInput:  NewTextInput(30),
	}
}

func (fc *FederationCreateScreen) selectedRosterNames(g *Game) []string {
	var names []string
	for i, sel := range fc.rosterSelect {
		if sel && i < len(g.Roster) {
			names = append(names, g.Roster[i].Name)
		}
	}
	return names
}

func (fc *FederationCreateScreen) selectedCount() int {
	count := 0
	for _, sel := range fc.rosterSelect {
		if sel {
			count++
		}
	}
	return count
}

func (fc *FederationCreateScreen) parsePPVFreq() int {
	n, err := strconv.Atoi(fc.ppvFreqInput.Text)
	if err != nil || n < 2 {
		return 4
	}
	return n
}

func (fc *FederationCreateScreen) Update(g *Game) error {
	if g.in.JustPressed(ebiten.KeyEscape) {
		fc.stepBack(g)
		return nil
	}
	switch fc.phase {
	case CreatePhaseName:
		fc.updateName(g)
	case CreatePhaseRoster:
		fc.updateRoster(g)
	case CreatePhaseChampionships:
		fc.updateChampionships(g)
	case CreatePhaseSchedule:
		fc.updateSchedule(g)
	case CreatePhasePPVNames:
		fc.updatePPVNames(g)
	case CreatePhaseConfirm:
		fc.updateConfirm(g)
	}
	return nil
}

func (fc *FederationCreateScreen) stepBack(g *Game) {
	if fc.phase == CreatePhaseName {
		g.SetScreen(NewFederationSelectScreen(g))
		return
	}
	fc.phase--
}

func (fc *FederationCreateScreen) updateName(g *Game) {
	fc.nameInput.Update(g.in)
	if g.in.JustPressed(ebiten.KeyEnter) && len(fc.nameInput.Text) > 0 {
		fc.phase = CreatePhaseRoster
		fc.rosterCursor = 0
		fc.rosterSelect = make([]bool, len(g.Roster))
	}
}

func (fc *FederationCreateScreen) updateRoster(g *Game) {
	fc.rosterCursor = handleListInput(g.in, fc.rosterCursor, len(g.Roster))
	if g.in.JustPressed(ebiten.KeySpace) {
		fc.rosterSelect[fc.rosterCursor] = !fc.rosterSelect[fc.rosterCursor]
	}
	if g.in.JustPressed(ebiten.KeyEnter) && fc.selectedCount() >= minFederationRoster {
		fc.phase = CreatePhaseChampionships
		fc.beltInput.Reset()
	}
}

func (fc *FederationCreateScreen) updateChampionships(g *Game) {
	if g.in.JustPressed(ebiten.KeyTab) && len(fc.belts) >= 1 {
		fc.phase = CreatePhaseSchedule
		fc.showNameInput.Reset()
		fc.ppvFreqInput.Reset()
		fc.ppvFreqInput.Text = "4"
		return
	}
	fc.belts = updateNameList(g.in, fc.beltInput, fc.belts)
}

func (fc *FederationCreateScreen) updateSchedule(g *Game) {
	fc.showNameInput.Update(g.in)
	if g.in.JustPressed(ebiten.KeyEnter) && len(fc.showNameInput.Text) > 0 {
		fc.phase = CreatePhasePPVNames
		fc.ppvNameInput.Reset()
	}
}

func (fc *FederationCreateScreen) updatePPVNames(g *Game) {
	if g.in.JustPressed(ebiten.KeyTab) {
		fc.phase = CreatePhaseConfirm
		return
	}
	fc.ppvNames = updateNameList(g.in, fc.ppvNameInput, fc.ppvNames)
}

func (fc *FederationCreateScreen) updateConfirm(g *Game) {
	if !g.in.JustPressed(ebiten.KeyEnter) {
		return
	}
	fed := engine.NewFederation(engine.FederationConfig{
		Name:           fc.nameInput.Text,
		RosterNames:    fc.selectedRosterNames(g),
		ChampNames:     fc.belts,
		WeeklyShowName: fc.showNameInput.Text,
		PPVFrequency:   fc.parsePPVFreq(),
		PPVNames:       fc.ppvNames,
	})
	fc.save.Federations = append(fc.save.Federations, fed)
	fc.save.ActiveIndex = len(fc.save.Federations) - 1
	g.SaveFederations(fc.save)
	g.SetScreen(NewCareerScreen(fed, fc.save))
}

// ─── Drawing ────────────────────────────────────────────────────────────────

func (fc *FederationCreateScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, showDivider, Margin, y)
	y += LineHeight
	DrawText(screen, "                 CREATE FEDERATION", Margin, y)
	y += LineHeight
	DrawText(screen, showDivider, Margin, y)
	y += LineHeight * 2

	if fc.phase == CreatePhaseRoster {
		DrawText(screen, fmt.Sprintf("SELECT ROSTER (min %d, selected %d): %s:", minFederationRoster, fc.selectedCount(), fc.nameInput.Text), Margin, y)
		drawList(screen, g, y+LineHeight*2, fc.rosterChoices(g), fc.rosterCursor)
	} else {
		for _, line := range fc.bodyLines() {
			DrawText(screen, line, Margin, y)
			y += LineHeight
		}
	}
	DrawText(screen, fc.statusLine(), Margin, g.screenH-LineHeight-Margin)
}

func (fc *FederationCreateScreen) rosterChoices(g *Game) []string {
	choices := rosterNames(g, nil)
	for i := range choices {
		box := "[ ] "
		if fc.rosterSelect[i] {
			box = "[x] "
		}
		choices[i] = box + choices[i]
	}
	return choices
}

func numberedLines(names []string) []string {
	lines := make([]string, len(names))
	for i, name := range names {
		lines[i] = fmt.Sprintf("  %d. %s", i+1, name)
	}
	return lines
}

func (fc *FederationCreateScreen) bodyLines() []string {
	switch fc.phase {
	case CreatePhaseName:
		return []string{"FEDERATION NAME:", "", "> " + fc.nameInput.DisplayText()}
	case CreatePhaseChampionships:
		lines := append([]string{"CHAMPIONSHIP BELTS:", ""}, numberedLines(fc.belts)...)
		return append(lines, "", "Add belt: > "+fc.beltInput.DisplayText())
	case CreatePhaseSchedule:
		return []string{
			"SHOW SCHEDULE:", "",
			"Weekly show name:", "> " + fc.showNameInput.DisplayText(), "",
			fmt.Sprintf("PPV every N weeks: %s", fc.ppvFreqInput.Text),
			"(edit this in Federation Settings later)",
		}
	case CreatePhasePPVNames:
		lines := append([]string{"PPV EVENT NAMES:", "(add none and press [TAB] to use the default names)", ""}, numberedLines(fc.ppvNames)...)
		return append(lines, "", "Add PPV: > "+fc.ppvNameInput.DisplayText())
	default:
		return fc.confirmLines()
	}
}

func (fc *FederationCreateScreen) confirmLines() []string {
	lines := []string{
		"CONFIRM FEDERATION:", "",
		fmt.Sprintf("  Name: %s", fc.nameInput.Text),
		fmt.Sprintf("  Roster: %d wrestlers", fc.selectedCount()),
		fmt.Sprintf("  Championships: %d", len(fc.belts)),
	}
	for i, belt := range fc.belts {
		lines = append(lines, fmt.Sprintf("    %d. %s", i+1, belt))
	}
	lines = append(lines,
		fmt.Sprintf("  Weekly Show: %s", fc.showNameInput.Text),
		fmt.Sprintf("  PPV Frequency: Every %d weeks", fc.parsePPVFreq()),
	)
	if len(fc.ppvNames) == 0 {
		return append(lines, fmt.Sprintf("  PPV Names: %d (defaults)", len(engine.DefaultPPVNames)))
	}
	return append(lines, fmt.Sprintf("  PPV Names: %d", len(fc.ppvNames)))
}

func (fc *FederationCreateScreen) statusLine() string {
	switch fc.phase {
	case CreatePhaseName:
		return "[TYPE] Enter Name  [ENTER] Confirm  [ESC] Cancel"
	case CreatePhaseRoster:
		return "[UP/DOWN] Move  [SPACE] Toggle  [ENTER] Confirm  [ESC] Back"
	case CreatePhaseChampionships:
		return "[TYPE] Belt Name  [ENTER] Add  [BACKSPACE] on an empty box removes the last  [TAB] Done  [ESC] Back"
	case CreatePhaseSchedule:
		return "[TYPE] Show Name  [ENTER] Confirm  [ESC] Back"
	case CreatePhasePPVNames:
		return "[TYPE] PPV Name  [ENTER] Add  [BACKSPACE] on an empty box removes the last  [TAB] Done  [ESC] Back"
	default:
		return "[ENTER] Create  [ESC] Back"
	}
}
