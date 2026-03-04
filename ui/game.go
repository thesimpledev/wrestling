package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
	"wrestling/loader"
	"wrestling/storage"
)

const (
	Version      = "0.1.0"
	WindowWidth  = 1280
	WindowHeight = 720
	CharWidth    = 6
	LineHeight   = 16
	Margin       = 8
	BgColor      = 0x10
)

var Background = color.RGBA{BgColor, BgColor, BgColor, 0xFF}

// Screen is implemented by each game screen (menu, match, editor).
type Screen interface {
	Update(g *Game) error
	Draw(screen *ebiten.Image, g *Game)
}

// Game is the top-level Ebitengine game. It delegates to the active Screen.
type Game struct {
	screen   Screen
	screenW  int
	screenH  int
	scale    int
	Roster   []*engine.WrestlerCard
	Store    storage.Store
	Injuries loader.InjuryStore
}

func NewGame(roster []*engine.WrestlerCard, store storage.Store) *Game {
	g := &Game{
		Roster:   roster,
		Store:    store,
		scale:    2,
		Injuries: loader.LoadInjuries(store),
	}
	g.screen = NewMenuScreen()
	return g
}

func (g *Game) SetScreen(s Screen) {
	g.screen = s
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		if g.scale == 2 {
			g.scale = 1
		} else {
			g.scale = 2
		}
	}
	return g.screen.Update(g)
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.screenW = screen.Bounds().Dx()
	g.screenH = screen.Bounds().Dy()
	g.screen.Draw(screen, g)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth / g.scale, outsideHeight / g.scale
}

// DrawText is a helper to draw a line at a position.
func DrawText(screen *ebiten.Image, text string, x, y int) {
	ebitenutil.DebugPrintAt(screen, text, x, y)
}

func reloadRoster(g *Game) {
	roster, err := loader.LoadAllCards(g.Store)
	if err == nil && len(roster) > 0 {
		g.Roster = roster
	}
}
