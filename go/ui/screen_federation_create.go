package ui

import (
	"fmt"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
	"wrestling/loader"
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
	switch fc.phase {
	case CreatePhaseName:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.SetScreen(NewFederationSelectScreen(g))
			return nil
		}
		fc.nameInput.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(fc.nameInput.Text) > 0 {
			fc.phase = CreatePhaseRoster
			fc.rosterCursor = 0
			fc.rosterSelect = make([]bool, len(g.Roster))
		}

	case CreatePhaseRoster:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			fc.phase = CreatePhaseName
			return nil
		}
		fc.rosterCursor = handleListInput(fc.rosterCursor, len(g.Roster))
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			fc.rosterSelect[fc.rosterCursor] = !fc.rosterSelect[fc.rosterCursor]
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && fc.selectedCount() >= 4 {
			fc.phase = CreatePhaseChampionships
			fc.beltInput.Reset()
		}

	case CreatePhaseChampionships:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			fc.phase = CreatePhaseRoster
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) && len(fc.belts) >= 1 {
			fc.phase = CreatePhaseSchedule
			fc.showNameInput.Reset()
			fc.ppvFreqInput.Reset()
			fc.ppvFreqInput.Text = "4"
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyD) && len(fc.belts) > 0 && len(fc.beltInput.Text) == 0 {
			fc.belts = fc.belts[:len(fc.belts)-1]
			return nil
		}
		fc.beltInput.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(fc.beltInput.Text) > 0 {
			fc.belts = append(fc.belts, fc.beltInput.Text)
			fc.beltInput.Reset()
		}

	case CreatePhaseSchedule:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			fc.phase = CreatePhaseChampionships
			return nil
		}
		// Tab switches between the two inputs
		fc.showNameInput.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(fc.showNameInput.Text) > 0 {
			fc.phase = CreatePhasePPVNames
			fc.ppvNameInput.Reset()
		}

	case CreatePhasePPVNames:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			fc.phase = CreatePhaseSchedule
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			fc.phase = CreatePhaseConfirm
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyD) && len(fc.ppvNames) > 0 && len(fc.ppvNameInput.Text) == 0 {
			fc.ppvNames = fc.ppvNames[:len(fc.ppvNames)-1]
			return nil
		}
		fc.ppvNameInput.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(fc.ppvNameInput.Text) > 0 {
			fc.ppvNames = append(fc.ppvNames, fc.ppvNameInput.Text)
			fc.ppvNameInput.Reset()
		}

	case CreatePhaseConfirm:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			fc.phase = CreatePhasePPVNames
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			cfg := engine.FederationConfig{
				Name:           fc.nameInput.Text,
				RosterNames:    fc.selectedRosterNames(g),
				ChampNames:     fc.belts,
				WeeklyShowName: fc.showNameInput.Text,
				PPVFrequency:   fc.parsePPVFreq(),
				PPVNames:       fc.ppvNames,
			}
			fed := engine.NewFederation(cfg)
			fc.save.Federations = append(fc.save.Federations, fed)
			fc.save.ActiveIndex = len(fc.save.Federations) - 1
			loader.SaveFederations(g.Store, fc.save)
			g.SetScreen(NewCareerScreen(fed, fc.save))
		}
	}

	return nil
}

func (fc *FederationCreateScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                 CREATE FEDERATION", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	switch fc.phase {
	case CreatePhaseName:
		DrawText(screen, "FEDERATION NAME:", Margin, y)
		y += LineHeight * 2
		DrawText(screen, "> "+fc.nameInput.DisplayText(), Margin, y)

		statusY := g.screenH - LineHeight - Margin
		DrawText(screen, "[TYPE] Enter Name  [ENTER] Confirm  [ESC] Cancel", Margin, statusY)

	case CreatePhaseRoster:
		DrawText(screen, fmt.Sprintf("SELECT ROSTER (min 4) — %s:", fc.nameInput.Text), Margin, y)
		y += LineHeight * 2

		for i, card := range g.Roster {
			prefix := "  "
			if i == fc.rosterCursor {
				prefix = "> "
			}
			check := "[ ]"
			if fc.rosterSelect[i] {
				check = "[x]"
			}
			DrawText(screen, fmt.Sprintf("%s%s %s", prefix, check, card.Name), Margin, y)
			y += LineHeight
		}
		y += LineHeight
		DrawText(screen, fmt.Sprintf("Selected: %d", fc.selectedCount()), Margin, y)

		statusY := g.screenH - LineHeight - Margin
		DrawText(screen, "[UP/DOWN] Move  [SPACE] Toggle  [ENTER] Confirm  [ESC] Back", Margin, statusY)

	case CreatePhaseChampionships:
		DrawText(screen, "CHAMPIONSHIP BELTS:", Margin, y)
		y += LineHeight * 2

		for i, belt := range fc.belts {
			DrawText(screen, fmt.Sprintf("  %d. %s", i+1, belt), Margin, y)
			y += LineHeight
		}
		y += LineHeight
		DrawText(screen, "Add belt: > "+fc.beltInput.DisplayText(), Margin, y)

		statusY := g.screenH - LineHeight - Margin
		DrawText(screen, "[TYPE] Belt Name  [ENTER] Add  [D] Delete Last  [TAB] Done  [ESC] Back", Margin, statusY)

	case CreatePhaseSchedule:
		DrawText(screen, "SHOW SCHEDULE:", Margin, y)
		y += LineHeight * 2

		DrawText(screen, "Weekly show name:", Margin, y)
		y += LineHeight
		DrawText(screen, "> "+fc.showNameInput.DisplayText(), Margin, y)
		y += LineHeight * 2

		DrawText(screen, fmt.Sprintf("PPV every N weeks: %s", fc.ppvFreqInput.Text), Margin, y)
		y += LineHeight
		DrawText(screen, "(edit this in Federation Settings later)", Margin, y)

		statusY := g.screenH - LineHeight - Margin
		DrawText(screen, "[TYPE] Show Name  [ENTER] Confirm  [ESC] Back", Margin, statusY)

	case CreatePhasePPVNames:
		DrawText(screen, "PPV EVENT NAMES:", Margin, y)
		y += LineHeight
		DrawText(screen, "(leave empty and press [SPACE] for defaults)", Margin, y)
		y += LineHeight * 2

		for i, name := range fc.ppvNames {
			DrawText(screen, fmt.Sprintf("  %d. %s", i+1, name), Margin, y)
			y += LineHeight
		}
		y += LineHeight
		DrawText(screen, "Add PPV: > "+fc.ppvNameInput.DisplayText(), Margin, y)

		statusY := g.screenH - LineHeight - Margin
		DrawText(screen, "[TYPE] PPV Name  [ENTER] Add  [D] Delete Last  [TAB] Done  [ESC] Back", Margin, statusY)

	case CreatePhaseConfirm:
		DrawText(screen, "CONFIRM FEDERATION:", Margin, y)
		y += LineHeight * 2

		DrawText(screen, fmt.Sprintf("  Name: %s", fc.nameInput.Text), Margin, y)
		y += LineHeight
		DrawText(screen, fmt.Sprintf("  Roster: %d wrestlers", fc.selectedCount()), Margin, y)
		y += LineHeight
		DrawText(screen, fmt.Sprintf("  Championships: %d", len(fc.belts)), Margin, y)
		y += LineHeight
		for i, belt := range fc.belts {
			DrawText(screen, fmt.Sprintf("    %d. %s", i+1, belt), Margin, y)
			y += LineHeight
		}
		DrawText(screen, fmt.Sprintf("  Weekly Show: %s", fc.showNameInput.Text), Margin, y)
		y += LineHeight
		DrawText(screen, fmt.Sprintf("  PPV Frequency: Every %d weeks", fc.parsePPVFreq()), Margin, y)
		y += LineHeight
		ppvCount := len(fc.ppvNames)
		if ppvCount == 0 {
			ppvCount = len(engine.DefaultPPVNames)
			DrawText(screen, fmt.Sprintf("  PPV Names: %d (defaults)", ppvCount), Margin, y)
		} else {
			DrawText(screen, fmt.Sprintf("  PPV Names: %d", ppvCount), Margin, y)
		}

		statusY := g.screenH - LineHeight - Margin
		DrawText(screen, "[ENTER] Create  [ESC] Back", Margin, statusY)
	}
}
