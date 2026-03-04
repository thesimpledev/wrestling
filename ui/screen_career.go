package ui

import (
	"fmt"

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
	CareerOptQuit
)

var careerMenuLabels = []string{
	"Next Show",
	"Standings & Records",
	"Match & Title History",
	"Quit Career Mode",
}

type CareerScreen struct {
	career *engine.CareerSave
	cursor int
}

func NewCareerScreen(career *engine.CareerSave) *CareerScreen {
	return &CareerScreen{career: career}
}

func (cs *CareerScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewMenuScreen())
		return nil
	}

	cs.cursor = handleListInput(cs.cursor, len(careerMenuLabels))

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		switch CareerMenuOption(cs.cursor) {
		case CareerOptNextShow:
			card := cs.career.AutoBook(g.Roster)
			g.SetScreen(NewCareerBookScreen(cs.career, card, g))
		case CareerOptStandings:
			g.SetScreen(NewCareerStandingsScreen(cs.career))
		case CareerOptHistory:
			g.SetScreen(NewCareerHistoryScreen(cs.career))
		case CareerOptQuit:
			loader.SaveCareer(g.Store, cs.career)
			g.SetScreen(NewMenuScreen())
		}
	}

	return nil
}

func (cs *CareerScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                    CAREER MODE", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	// Week & show info
	DrawText(screen, fmt.Sprintf("Week %d", cs.career.Week), Margin, y)
	y += LineHeight

	showType := "Weekly Show"
	if cs.career.IsPPV() {
		showType = fmt.Sprintf("PPV: %s", cs.career.CurrentPPVName())
	}
	DrawText(screen, fmt.Sprintf("Next: %s", showType), Margin, y)
	y += LineHeight * 2

	// Champion
	champ := cs.career.WorldChampion()
	if champ == "" {
		DrawText(screen, "World Heavyweight Championship: VACANT", Margin, y)
	} else {
		injured := ""
		if g.Injuries.IsInjured(champ) {
			injured = fmt.Sprintf(" [INJURED %d]", g.Injuries.InjuryCards(champ))
		}
		DrawText(screen, fmt.Sprintf("World Heavyweight Champion: %s%s", champ, injured), Margin, y)
	}
	y += LineHeight

	// Title shot earned
	if cs.career.TitleShotEarned != "" {
		DrawText(screen, fmt.Sprintf("#1 Contender: %s (earned title shot)", cs.career.TitleShotEarned), Margin, y)
		y += LineHeight
	}
	y += LineHeight

	// Active rivalries
	rivals := cs.career.ActiveRivals()
	if len(rivals) > 0 {
		DrawText(screen, "ACTIVE RIVALRIES:", Margin, y)
		y += LineHeight
		for _, pair := range rivals {
			score := cs.career.RivalryScore(pair[0], pair[1])
			DrawText(screen, fmt.Sprintf("  %s vs %s (intensity: %d)", pair[0], pair[1], score), Margin, y)
			y += LineHeight
		}
		y += LineHeight
	}

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
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Main Menu", Margin, statusY)
}
