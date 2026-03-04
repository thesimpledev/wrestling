package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
)

type BookPhase int

const (
	BookViewCard BookPhase = iota
	BookEditMatch
	BookEditType
	BookEditSide1
	BookEditSide2
)

type CareerBookScreen struct {
	career *engine.CareerSave
	card   []engine.BookedMatch
	cursor int
	phase  BookPhase

	// Edit state
	editIdx      int
	editCursor   int
	roster       []*engine.WrestlerCard
}

func NewCareerBookScreen(career *engine.CareerSave, card []engine.BookedMatch, g *Game) *CareerBookScreen {
	return &CareerBookScreen{
		career: career,
		card:   card,
		roster: g.Roster,
	}
}

func (bs *CareerBookScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if bs.phase != BookViewCard {
			bs.phase = BookViewCard
			return nil
		}
		g.SetScreen(NewCareerScreen(bs.career))
		return nil
	}

	switch bs.phase {
	case BookViewCard:
		bs.updateViewCard(g)
	case BookEditType:
		bs.updateEditType(g)
	case BookEditSide1:
		bs.updateEditSide(g, 0)
	case BookEditSide2:
		bs.updateEditSide(g, 1)
	}

	return nil
}

func (bs *CareerBookScreen) updateViewCard(g *Game) {
	// Extra options: Watch All, Simulate All, Watch Main Event
	totalItems := len(bs.card) + 3
	bs.cursor = handleListInput(bs.cursor, totalItems)

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if bs.cursor < len(bs.card) {
			// Edit this match
			bs.editIdx = bs.cursor
			bs.editCursor = 0
			bs.phase = BookEditType
		} else {
			actionIdx := bs.cursor - len(bs.card)
			switch actionIdx {
			case 0: // Watch All
				g.SetScreen(NewCareerShowScreen(bs.career, bs.card, ShowModeWatch, g))
			case 1: // Simulate All
				g.SetScreen(NewCareerShowScreen(bs.career, bs.card, ShowModeSimulate, g))
			case 2: // Watch Main Event Only
				g.SetScreen(NewCareerShowScreen(bs.career, bs.card, ShowModeMainEvent, g))
			}
		}
	}
}

var editableTypes = []engine.MatchType{
	engine.MatchSingles,
	engine.MatchTag,
	engine.MatchCage,
	engine.MatchNoDQ,
}

var editableTypeNames = []string{
	"Singles",
	"Tag Team",
	"Cage",
	"No DQ",
}

func (bs *CareerBookScreen) updateEditType(g *Game) {
	bs.editCursor = handleListInput(bs.editCursor, len(editableTypes))
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		match := &bs.card[bs.editIdx]
		match.Type = editableTypes[bs.editCursor]
		// For tag, ensure 2 per side
		if match.Type == engine.MatchTag {
			if len(match.Side1) < 2 {
				match.Side1 = append(match.Side1, "")
			}
			if len(match.Side2) < 2 {
				match.Side2 = append(match.Side2, "")
			}
		} else {
			if len(match.Side1) > 1 {
				match.Side1 = match.Side1[:1]
			}
			if len(match.Side2) > 1 {
				match.Side2 = match.Side2[:1]
			}
		}
		bs.editCursor = 0
		bs.phase = BookEditSide1
	}
}

func (bs *CareerBookScreen) updateEditSide(g *Game, sideIdx int) {
	bs.editCursor = handleListInput(bs.editCursor, len(g.Roster))
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		match := &bs.card[bs.editIdx]
		name := g.Roster[bs.editCursor].Name
		if sideIdx == 0 {
			if len(match.Side1) > 0 {
				match.Side1[0] = name
			} else {
				match.Side1 = []string{name}
			}
			bs.editCursor = 0
			bs.phase = BookEditSide2
		} else {
			if len(match.Side2) > 0 {
				match.Side2[0] = name
			} else {
				match.Side2 = []string{name}
			}
			bs.phase = BookViewCard
		}
	}
}

func (bs *CareerBookScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	title := fmt.Sprintf("  Week %d - %s", bs.career.Week, bs.career.ShowName())
	DrawText(screen, title, Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	switch bs.phase {
	case BookViewCard:
		bs.drawCard(screen, g, y)
	case BookEditType:
		bs.drawEditType(screen, g, y)
	case BookEditSide1:
		bs.drawEditSide(screen, g, y, "WRESTLER 1 (Side 1)")
	case BookEditSide2:
		bs.drawEditSide(screen, g, y, "WRESTLER 2 (Side 2)")
	}
}

func (bs *CareerBookScreen) drawCard(screen *ebiten.Image, g *Game, y int) {
	DrawText(screen, "FIGHT CARD:", Margin, y)
	y += LineHeight * 2

	for i, m := range bs.card {
		prefix := "  "
		if i == bs.cursor {
			prefix = "> "
		}

		var line string
		if len(m.BREntrants) > 0 {
			line = fmt.Sprintf("[BATTLE ROYAL] %d-man Battle Royal", len(m.BREntrants))
		} else if m.IsTournament {
			titleTag := ""
			if m.IsTitle {
				titleTag = "TITLE "
			}
			line = fmt.Sprintf("[%sTOURNAMENT] %d-man Tournament", titleTag, m.TournSize)
		} else {
			typeStr := engine.MatchTypeString(m.Type)
			titleTag := ""
			if m.IsTitle {
				titleTag = "TITLE - "
			}
			s1 := "TBD"
			if len(m.Side1) > 0 && m.Side1[0] != "" {
				s1 = m.Side1[0]
				if g.Injuries.IsInjured(s1) {
					s1 += "*"
				}
			}
			s2 := "TBD"
			if len(m.Side2) > 0 && m.Side2[0] != "" {
				s2 = m.Side2[0]
				if g.Injuries.IsInjured(s2) {
					s2 += "*"
				}
			}
			// Mark champion
			champ := bs.career.WorldChampion()
			if s1 == champ || (len(m.Side1) > 0 && m.Side1[0] == champ) {
				s1 += " (c)"
			}
			if s2 == champ || (len(m.Side2) > 0 && m.Side2[0] == champ) {
				s2 += " (c)"
			}
			// Mark rivals
			isFeud := len(m.Side1) > 0 && len(m.Side2) > 0 && bs.career.IsRival(m.Side1[0], m.Side2[0])
			feudTag := ""
			if isFeud {
				feudTag = " [FEUD]"
			}
			line = fmt.Sprintf("[%s%s] %s vs %s%s", titleTag, typeStr, s1, s2, feudTag)
		}

		DrawText(screen, fmt.Sprintf("%s%d. %s", prefix, i+1, line), Margin, y)
		y += LineHeight
	}

	y += LineHeight

	// Action options
	actions := []string{"Watch All Matches", "Simulate All Matches", "Watch Main Event Only"}
	for i, label := range actions {
		idx := len(bs.card) + i
		prefix := "  "
		if idx == bs.cursor {
			prefix = "> "
		}
		DrawText(screen, prefix+label, Margin, y)
		y += LineHeight
	}

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Edit/Start  [ESC] Back", Margin, statusY)
}

func (bs *CareerBookScreen) drawEditType(screen *ebiten.Image, g *Game, y int) {
	DrawText(screen, fmt.Sprintf("EDIT MATCH %d — SELECT TYPE:", bs.editIdx+1), Margin, y)
	y += LineHeight * 2

	for i, name := range editableTypeNames {
		prefix := "  "
		if i == bs.editCursor {
			prefix = "> "
		}
		DrawText(screen, prefix+name, Margin, y)
		y += LineHeight
	}

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Cancel", Margin, statusY)
}

func (bs *CareerBookScreen) drawEditSide(screen *ebiten.Image, g *Game, y int, label string) {
	DrawText(screen, fmt.Sprintf("EDIT MATCH %d — SELECT %s:", bs.editIdx+1, label), Margin, y)
	y += LineHeight * 2

	for i, card := range g.Roster {
		prefix := "  "
		if i == bs.editCursor {
			prefix = "> "
		}
		name := card.Name
		if g.Injuries.IsInjured(card.Name) {
			name += fmt.Sprintf("  [INJURED %d]", g.Injuries.InjuryCards(card.Name))
		}
		DrawText(screen, prefix+name, Margin, y)
		y += LineHeight
	}

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Cancel", Margin, statusY)
}
