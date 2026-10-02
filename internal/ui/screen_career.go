package ui

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
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
	if g.in.JustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewFederationSelectScreen(g))
		return nil
	}

	cs.cursor = handleListInput(g.in, cs.cursor, len(careerMenuLabels))

	if g.in.JustPressed(ebiten.KeyEnter) || g.in.JustPressed(ebiten.KeySpace) {
		fedRoster := FilterRoster(g.Roster, cs.fed.Roster)
		switch CareerMenuOption(cs.cursor) {
		case CareerOptNextShow:
			card := cs.fed.AutoBook(bookableRoster(g, fedRoster))
			g.SetScreen(NewCareerBookScreen(cs.fed, cs.save, card, g))
		case CareerOptStandings:
			g.SetScreen(NewCareerStandingsScreen(cs.fed, cs.save))
		case CareerOptHistory:
			g.SetScreen(NewCareerHistoryScreen(cs.fed, cs.save))
		case CareerOptSettings:
			g.SetScreen(NewFederationSettingsScreen(cs.fed, cs.save))
		case CareerOptQuit:
			g.SaveFederations(cs.save)
			g.SetScreen(NewFederationSelectScreen(g))
		}
	}

	return nil
}

func (cs *CareerScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	drawLines(screen, cs.dashboardLines(g))
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Federation Select", Margin, g.screenH-LineHeight-Margin)
}

// dashboardLines is the federation dashboard: the week, the champions, the
// earned title shot, the active rivalries, and the menu.
func (cs *CareerScreen) dashboardLines(g *Game) []string {
	lines := []string{
		showDivider,
		fmt.Sprintf("                %s", strings.ToUpper(cs.fed.Name)),
		showDivider,
		"",
		fmt.Sprintf("Week %d", cs.fed.Week),
		fmt.Sprintf("Next: %s", cs.fed.ShowName()),
		"",
	}
	lines = append(lines, cs.championLines(g)...)
	lines = append(lines, "")
	if cs.fed.TitleShotEarned != "" {
		lines = append(lines, fmt.Sprintf("#1 Contender: %s (earned title shot)", cs.fed.TitleShotEarned))
	}
	lines = append(lines, cs.rivalryLines()...)
	lines = append(lines, "")

	for i, label := range careerMenuLabels {
		prefix := "  "
		if i == cs.cursor {
			prefix = "> "
		}
		lines = append(lines, prefix+label)
	}
	return lines
}

func (cs *CareerScreen) championLines(g *Game) []string {
	lines := make([]string, 0, len(cs.fed.Championships))
	for _, ch := range cs.fed.Championships {
		if ch.Champion == "" {
			lines = append(lines, fmt.Sprintf("%s: VACANT", ch.Name))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s%s", ch.Name, ch.Champion, statusMarkers(g, ch.Champion)))
	}
	return lines
}

func (cs *CareerScreen) rivalryLines() []string {
	rivals := cs.fed.ActiveRivals()
	if len(rivals) == 0 {
		return nil
	}
	lines := []string{"", "ACTIVE RIVALRIES:"}
	for _, pair := range rivals {
		lines = append(lines, fmt.Sprintf("  %s vs %s (intensity: %d)", pair[0], pair[1], cs.fed.RivalryScore(pair[0], pair[1])))
	}
	return lines
}
