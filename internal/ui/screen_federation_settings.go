package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
)

type SettingsField int

const (
	SettingsFieldName SettingsField = iota
	SettingsFieldShowName
	SettingsFieldPPVFreq
	SettingsFieldPPVNames
	SettingsFieldSave
)

var settingsFieldCount = 5

type SettingsPhase int

const (
	SettingsNav SettingsPhase = iota
	SettingsEditing
	SettingsEditPPVNames
)

const (
	settingsTextLength = 30
	ppvFreqLength      = 2
	minPPVFrequency    = 2
)

type FederationSettingsScreen struct {
	fed   *engine.Federation
	save  *engine.FederationSave
	field SettingsField
	phase SettingsPhase

	input    *TextInput
	ppvInput *TextInput

	// Edits are kept here and written to the federation only by Save &
	// Return, so ESC leaves the federation as it was.
	name         string
	showName     string
	ppvFrequency int
	ppvNames     []string
}

func NewFederationSettingsScreen(fed *engine.Federation, save *engine.FederationSave) *FederationSettingsScreen {
	return &FederationSettingsScreen{
		fed:          fed,
		save:         save,
		input:        NewTextInput(settingsTextLength),
		ppvInput:     NewTextInput(settingsTextLength),
		name:         fed.Name,
		showName:     fed.WeeklyShowName,
		ppvFrequency: fed.PPVFrequency,
		ppvNames:     append([]string{}, fed.PPVNames...),
	}
}

func (fs *FederationSettingsScreen) Update(g *Game) error {
	switch fs.phase {
	case SettingsEditing:
		fs.updateEditing(g)
	case SettingsEditPPVNames:
		fs.updatePPVNames(g)
	default:
		fs.updateNav(g)
	}
	return nil
}

func (fs *FederationSettingsScreen) updateNav(g *Game) {
	if g.in.JustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewCareerScreen(fs.fed, fs.save))
		return
	}
	fs.field = SettingsField(handleListInput(g.in, int(fs.field), settingsFieldCount))
	if !confirmPressed(g.in) {
		return
	}

	switch fs.field {
	case SettingsFieldName:
		fs.startEditing(fs.name, settingsTextLength)
	case SettingsFieldShowName:
		fs.startEditing(fs.showName, settingsTextLength)
	case SettingsFieldPPVFreq:
		fs.startEditing(strconv.Itoa(fs.ppvFrequency), ppvFreqLength)
	case SettingsFieldPPVNames:
		fs.ppvInput.Reset()
		fs.phase = SettingsEditPPVNames
	case SettingsFieldSave:
		fs.saveAndReturn(g)
	}
}

func (fs *FederationSettingsScreen) startEditing(text string, maxLength int) {
	fs.input.Reset()
	fs.input.MaxLength = maxLength
	fs.input.Text = text
	fs.phase = SettingsEditing
}

func (fs *FederationSettingsScreen) saveAndReturn(g *Game) {
	fs.fed.Name = fs.name
	fs.fed.WeeklyShowName = fs.showName
	fs.fed.PPVFrequency = fs.ppvFrequency
	fs.fed.PPVNames = fs.ppvNames
	g.SaveFederations(fs.save)
	g.SetScreen(NewCareerScreen(fs.fed, fs.save))
}

func (fs *FederationSettingsScreen) updateEditing(g *Game) {
	if g.in.JustPressed(ebiten.KeyEscape) {
		fs.phase = SettingsNav
		return
	}
	fs.input.Update(g.in)
	if !g.in.JustPressed(ebiten.KeyEnter) || fs.input.Text == "" {
		return
	}

	switch fs.field {
	case SettingsFieldName:
		fs.name = fs.input.Text
	case SettingsFieldShowName:
		fs.showName = fs.input.Text
	case SettingsFieldPPVFreq:
		n, err := strconv.Atoi(fs.input.Text)
		if err != nil || n < minPPVFrequency {
			g.SetNotice(fmt.Sprintf("PPV frequency must be a number, %d or more.", minPPVFrequency))
			return
		}
		fs.ppvFrequency = n
	}
	fs.phase = SettingsNav
}

func (fs *FederationSettingsScreen) updatePPVNames(g *Game) {
	if g.in.JustPressed(ebiten.KeyEscape) {
		fs.phase = SettingsNav
		return
	}
	fs.ppvNames = updateNameList(g.in, fs.ppvInput, fs.ppvNames)
}

func (fs *FederationSettingsScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, showDivider, Margin, y)
	y += LineHeight
	DrawText(screen, fmt.Sprintf("          FEDERATION SETTINGS: %s", strings.ToUpper(fs.fed.Name)), Margin, y)
	y += LineHeight
	DrawText(screen, showDivider, Margin, y)
	y += LineHeight * 2

	switch fs.phase {
	case SettingsNav:
		fs.drawNav(screen, g, y)
	case SettingsEditing:
		fs.drawEditing(screen, g, y)
	case SettingsEditPPVNames:
		fs.drawPPVNames(screen, g, y)
	}
}

func (fs *FederationSettingsScreen) drawNav(screen *ebiten.Image, g *Game, y int) {
	fields := []struct {
		label string
		value string
	}{
		{"Federation Name", fs.name},
		{"Weekly Show Name", fs.showName},
		{"PPV Frequency", fmt.Sprintf("Every %d weeks", fs.ppvFrequency)},
		{"PPV Names", fmt.Sprintf("%d events", len(fs.ppvNames))},
		{"Save & Return", ""},
	}

	for i, f := range fields {
		prefix := "  "
		if SettingsField(i) == fs.field {
			prefix = "> "
		}
		if f.value != "" {
			DrawText(screen, fmt.Sprintf("%s%-22s %s", prefix, f.label, f.value), Margin, y)
		} else {
			DrawText(screen, prefix+f.label, Margin, y)
		}
		y += LineHeight
	}

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Edit  [ESC] Back without saving", Margin, statusY)
}

func (fs *FederationSettingsScreen) drawEditing(screen *ebiten.Image, g *Game, y int) {
	labels := []string{"FEDERATION NAME", "WEEKLY SHOW NAME", "PPV FREQUENCY (weeks, 2 or more)"}
	idx := int(fs.field)
	if idx < len(labels) {
		DrawText(screen, labels[idx]+":", Margin, y)
	}
	y += LineHeight * 2
	DrawText(screen, "> "+fs.input.DisplayText(), Margin, y)

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[TYPE] Edit  [ENTER] Confirm  [ESC] Cancel", Margin, statusY)
}

func (fs *FederationSettingsScreen) drawPPVNames(screen *ebiten.Image, g *Game, y int) {
	DrawText(screen, "PPV EVENT NAMES:", Margin, y)
	y += LineHeight * 2

	for _, line := range numberedLines(fs.ppvNames) {
		DrawText(screen, line, Margin, y)
		y += LineHeight
	}
	y += LineHeight
	DrawText(screen, "Add PPV: > "+fs.ppvInput.DisplayText(), Margin, y)

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[TYPE] PPV Name  [ENTER] Add  [BACKSPACE] on an empty box removes the last  [ESC] Done", Margin, statusY)
}
