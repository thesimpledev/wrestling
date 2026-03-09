package ui

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
	"wrestling/loader"
)

type CareerMenuOption int

const (
	CareerOptNextShow CareerMenuOption = iota
	CareerOptStandings
	CareerOptHistory
	CareerOptSettings
	CareerOptQuit
)

var careerMenuLabels = []string{
	"Next Show",
	"Standings & Records",
	"Match & Title History",
	"Federation Settings",
	"Quit Federation",
}

type CareerScreen struct {
	fed    *engine.Federation
	save   *engine.FederationSave
	cursor int
}

func NewCareerScreen(fed *engine.Federation, save *engine.FederationSave) *CareerScreen {
	return &CareerScreen{fed: fed, save: save}
}

func (cs *CareerScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewFederationSelectScreen(g))
		return nil
	}

	cs.cursor = handleListInput(cs.cursor, len(careerMenuLabels))

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		fedRoster := FilterRoster(g.Roster, cs.fed.Roster)
		switch CareerMenuOption(cs.cursor) {
		case CareerOptNextShow:
			card := cs.fed.AutoBook(fedRoster)
			g.SetScreen(NewCareerBookScreen(cs.fed, cs.save, card, g))
		case CareerOptStandings:
			g.SetScreen(NewCareerStandingsScreen(cs.fed, cs.save))
		case CareerOptHistory:
			g.SetScreen(NewCareerHistoryScreen(cs.fed, cs.save))
		case CareerOptSettings:
			g.SetScreen(NewFederationSettingsScreen(cs.fed, cs.save))
		case CareerOptQuit:
			loader.SaveFederations(g.Store, cs.save)
			g.SetScreen(NewFederationSelectScreen(g))
		}
	}

	return nil
}

func (cs *CareerScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	title := fmt.Sprintf("                %s", strings.ToUpper(cs.fed.Name))
	DrawText(screen, title, Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	// Week & show info
	DrawText(screen, fmt.Sprintf("Week %d", cs.fed.Week), Margin, y)
	y += LineHeight

	DrawText(screen, fmt.Sprintf("Next: %s", cs.fed.ShowName()), Margin, y)
	y += LineHeight * 2

	// All championships
	for _, ch := range cs.fed.Championships {
		if ch.Champion == "" {
			DrawText(screen, fmt.Sprintf("%s: VACANT", ch.Name), Margin, y)
		} else {
			injured := ""
			if g.Injuries.IsInjured(ch.Champion) {
				injured = fmt.Sprintf(" [INJURED %d]", g.Injuries.InjuryCards(ch.Champion))
			}
			DrawText(screen, fmt.Sprintf("%s: %s%s", ch.Name, ch.Champion, injured), Margin, y)
		}
		y += LineHeight
	}

	// Title shot earned
	if cs.fed.TitleShotEarned != "" {
		y += LineHeight
		DrawText(screen, fmt.Sprintf("#1 Contender: %s (earned title shot)", cs.fed.TitleShotEarned), Margin, y)
	}
	y += LineHeight

	// Active rivalries
	rivals := cs.fed.ActiveRivals()
	if len(rivals) > 0 {
		y += LineHeight
		DrawText(screen, "ACTIVE RIVALRIES:", Margin, y)
		y += LineHeight
		for _, pair := range rivals {
			score := cs.fed.RivalryScore(pair[0], pair[1])
			DrawText(screen, fmt.Sprintf("  %s vs %s (intensity: %d)", pair[0], pair[1], score), Margin, y)
			y += LineHeight
		}
	}
	y += LineHeight

	// Menu options
	for i, label := range careerMenuLabels {
		prefix := "  "
		if i == cs.cursor {
			prefix = "> "
		}
		DrawText(screen, prefix+label, Margin, y)
		y += LineHeight
	}

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Federation Select", Margin, statusY)
}
