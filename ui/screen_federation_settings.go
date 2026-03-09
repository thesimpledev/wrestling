package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
	"wrestling/loader"
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

type FederationSettingsScreen struct {
	fed   *engine.Federation
	save  *engine.FederationSave
	field SettingsField
	phase SettingsPhase

	input *TextInput

	// PPV name editing
	ppvCursor int
	ppvInput  *TextInput
	ppvNames  []string
}

func NewFederationSettingsScreen(fed *engine.Federation, save *engine.FederationSave) *FederationSettingsScreen {
	return &FederationSettingsScreen{
		fed:      fed,
		save:     save,
		input:    NewTextInput(30),
		ppvInput: NewTextInput(30),
	}
}

func (fs *FederationSettingsScreen) Update(g *Game) error {
	switch fs.phase {
	case SettingsNav:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.SetScreen(NewCareerScreen(fs.fed, fs.save))
			return nil
		}
		fs.field = SettingsField(handleListInput(int(fs.field), settingsFieldCount))

		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			switch fs.field {
			case SettingsFieldName:
				fs.input.Reset()
				fs.input.Text = fs.fed.Name
				fs.phase = SettingsEditing
			case SettingsFieldShowName:
				fs.input.Reset()
				fs.input.Text = fs.fed.WeeklyShowName
				fs.phase = SettingsEditing
			case SettingsFieldPPVFreq:
				fs.input.Reset()
				fs.input.MaxLength = 2
				fs.input.Text = strconv.Itoa(fs.fed.PPVFrequency)
				fs.phase = SettingsEditing
			case SettingsFieldPPVNames:
				fs.ppvNames = make([]string, len(fs.fed.PPVNames))
				copy(fs.ppvNames, fs.fed.PPVNames)
				fs.ppvCursor = 0
				fs.ppvInput.Reset()
				fs.phase = SettingsEditPPVNames
			case SettingsFieldSave:
				loader.SaveFederations(g.Store, fs.save)
				g.SetScreen(NewCareerScreen(fs.fed, fs.save))
			}
		}

	case SettingsEditing:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			fs.phase = SettingsNav
			fs.input.MaxLength = 30
			return nil
		}
		fs.input.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(fs.input.Text) > 0 {
			switch fs.field {
			case SettingsFieldName:
				fs.fed.Name = fs.input.Text
			case SettingsFieldShowName:
				fs.fed.WeeklyShowName = fs.input.Text
			case SettingsFieldPPVFreq:
				n, err := strconv.Atoi(fs.input.Text)
				if err == nil && n >= 2 {
					fs.fed.PPVFrequency = n
				}
			}
			fs.phase = SettingsNav
			fs.input.MaxLength = 30
		}

	case SettingsEditPPVNames:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			// Save ppv names back
			fs.fed.PPVNames = fs.ppvNames
			fs.phase = SettingsNav
			return nil
		}
		fs.ppvInput.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && len(fs.ppvInput.Text) > 0 {
			fs.ppvNames = append(fs.ppvNames, fs.ppvInput.Text)
			fs.ppvInput.Reset()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyD) && len(fs.ppvNames) > 0 && len(fs.ppvInput.Text) == 0 {
			fs.ppvNames = fs.ppvNames[:len(fs.ppvNames)-1]
		}
	}

	return nil
}

func (fs *FederationSettingsScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, fmt.Sprintf("          FEDERATION SETTINGS — %s", strings.ToUpper(fs.fed.Name)), Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
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
		{"Federation Name", fs.fed.Name},
		{"Weekly Show Name", fs.fed.WeeklyShowName},
		{"PPV Frequency", fmt.Sprintf("Every %d weeks", fs.fed.PPVFrequency)},
		{"PPV Names", fmt.Sprintf("%d events", len(fs.fed.PPVNames))},
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
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Edit  [ESC] Back (unsaved)", Margin, statusY)
}

func (fs *FederationSettingsScreen) drawEditing(screen *ebiten.Image, g *Game, y int) {
	labels := []string{"FEDERATION NAME", "WEEKLY SHOW NAME", "PPV FREQUENCY (weeks)"}
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

	for i, name := range fs.ppvNames {
		DrawText(screen, fmt.Sprintf("  %d. %s", i+1, name), Margin, y)
		y += LineHeight
	}
	y += LineHeight
	DrawText(screen, "Add PPV: > "+fs.ppvInput.DisplayText(), Margin, y)

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[TYPE] PPV Name  [ENTER] Add  [D] Delete Last  [ESC] Save & Back", Margin, statusY)
}
