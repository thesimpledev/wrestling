package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"wrestling/internal/engine"
	"wrestling/internal/loader"
	"wrestling/internal/storage"
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
	notice   string
	in       Input
	Roster   []*engine.WrestlerCard
	Store    storage.Store
	Injuries loader.InjuryStore
}

func NewGame(roster []*engine.WrestlerCard, store storage.Store) *Game {
	g := &Game{
		Roster: roster,
		Store:  store,
		scale:  2,
		in:     ebitenInput{},
	}
	injuries, err := loader.LoadInjuries(store)
	if err != nil {
		g.SetNotice("Saved injuries could not be read and will be replaced on the next save.")
	}
	g.Injuries = injuries
	g.screen = NewMenuScreen()
	return g
}

// EndFightCard closes out one fight card: every running injury and
// suspension counts down by one card, then the injuries and suspensions that
// happened on this card are recorded at their full length, and all is saved.
func (g *Game) EndFightCard(results []*engine.MatchResult) {
	g.Injuries.DecrementAll()
	for _, result := range results {
		if result == nil {
			continue
		}
		g.Injuries.RecordInjury(result.InjuredWrestler, result.InjuryCards)
		for _, name := range result.SuspendedWrestlers {
			g.Injuries.RecordSuspension(name, result.SuspensionCards)
		}
	}
	g.SaveInjuries()
}

// statusMarkers is the injury or suspension tag shown after a wrestler's
// name in lists.
func statusMarkers(g *Game, name string) string {
	markers := ""
	if g.Injuries.IsInjured(name) {
		markers += fmt.Sprintf("  [INJURED %d]", g.Injuries.InjuryCards(name))
	}
	if g.Injuries.IsSuspended(name) {
		markers += fmt.Sprintf("  [SUSPENDED %d]", g.Injuries.SuspensionCards(name))
	}
	return markers
}

func (g *Game) SetScreen(s Screen) {
	g.screen = s
}

// SetNotice shows a banner message on top of the active screen until the
// next keypress.
func (g *Game) SetNotice(msg string) {
	g.notice = msg
}

// SaveInjuries persists the injury store and shows a banner if it fails.
func (g *Game) SaveInjuries() {
	if err := loader.SaveInjuries(g.Store, g.Injuries); err != nil {
		g.SetNotice("SAVE FAILED (injuries): " + err.Error())
	}
}

// SaveFederations persists career data and shows a banner if it fails.
func (g *Game) SaveFederations(save *engine.FederationSave) {
	if err := loader.SaveFederations(g.Store, save); err != nil {
		g.SetNotice("SAVE FAILED (career): " + err.Error())
	}
}

func (g *Game) Update() error {
	if g.notice != "" && g.in.AnyJustPressed() {
		g.notice = ""
	}
	if g.in.JustPressed(ebiten.KeyF1) {
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
	if g.notice != "" {
		banner := screen.SubImage(image.Rect(0, 0, g.screenW, LineHeight+Margin)).(*ebiten.Image)
		banner.Fill(color.RGBA{0xA0, 0x10, 0x10, 0xFF})
		DrawText(screen, g.notice, Margin, Margin/2)
	}
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

// FilterRoster returns only the wrestlers whose names are in the given list.
func FilterRoster(full []*engine.WrestlerCard, names []string) []*engine.WrestlerCard {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	var filtered []*engine.WrestlerCard
	for _, w := range full {
		if nameSet[w.Name] {
			filtered = append(filtered, w)
		}
	}
	return filtered
}
