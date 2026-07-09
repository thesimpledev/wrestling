package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
	"wrestling/loader"
)

type FederationSelectScreen struct {
	save   *engine.FederationSave
	cursor int
}

func NewFederationSelectScreen(g *Game) *FederationSelectScreen {
	save := loader.LoadFederations(g.Store)
	if save == nil {
		save = &engine.FederationSave{}
	}
	return &FederationSelectScreen{save: save}
}

func (fs *FederationSelectScreen) itemCount() int {
	return len(fs.save.Federations) + 1 // +1 for "Create New Federation"
}

func (fs *FederationSelectScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewMenuScreen())
		return nil
	}

	fs.cursor = handleListInput(fs.cursor, fs.itemCount())

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if fs.cursor < len(fs.save.Federations) {
			// Select existing federation
			fs.save.ActiveIndex = fs.cursor
			loader.SaveFederations(g.Store, fs.save)
			fed := fs.save.Federations[fs.cursor]
			g.SetScreen(NewCareerScreen(fed, fs.save))
		} else {
			// Create new federation
			g.SetScreen(NewFederationCreateScreen(fs.save))
		}
	}

	return nil
}

func (fs *FederationSelectScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                    FEDERATIONS", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	DrawText(screen, "SELECT FEDERATION:", Margin, y)
	y += LineHeight * 2

	for i, fed := range fs.save.Federations {
		prefix := "  "
		if i == fs.cursor {
			prefix = "> "
		}
		DrawText(screen, fmt.Sprintf("%s%-30s Week %d", prefix, fed.Name, fed.Week), Margin, y)
		y += LineHeight
	}

	// Create new option
	prefix := "  "
	if fs.cursor == len(fs.save.Federations) {
		prefix = "> "
	}
	DrawText(screen, prefix+"Create New Federation", Margin, y)

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Main Menu", Margin, statusY)
}
