package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
)

type MenuPhase int

const (
	PhaseSelectMatchType MenuPhase = iota
	PhaseSelectWrestler1
	PhaseSelectWrestler2
	PhaseSelectWrestler3 // Tag team only
	PhaseSelectWrestler4 // Tag team only
	PhaseSelectAlly1     // Optional ally for side 1
	PhaseSelectAlly2     // Optional ally for side 2
	PhaseReady
)

type MenuScreen struct {
	phase     MenuPhase
	cursor    int
	selected  [4]int // Up to 4 wrestler indices
	allies    [2]int // Ally indices for each side (-1 = no ally)
	matchType engine.MatchType
	isFeud    bool
}

func NewMenuScreen() *MenuScreen {
	return &MenuScreen{
		selected: [4]int{-1, -1, -1, -1},
		allies:   [2]int{-1, -1},
	}
}

var menuOptions = []string{
	"Singles Match",
	"Tag Team Match",
	"Cage Match",
	"No DQ Match",
	"Feud Match",
	"Create New Card",
	"Edit Existing Card",
}
var matchTypes = []engine.MatchType{
	engine.MatchSingles,
	engine.MatchTag,
	engine.MatchCage,
	engine.MatchNoDQ,
	engine.MatchSingles, // Feud uses singles type with IsFeud flag
}

const (
	menuSingles  = 0
	menuTag      = 1
	menuCage     = 2
	menuNoDQ     = 3
	menuFeud     = 4
	menuNewCard  = 5
	menuEditCard = 6
)

func (m *MenuScreen) isSelected(idx int) bool {
	for _, s := range m.selected {
		if s == idx {
			return true
		}
	}
	return false
}


func (m *MenuScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if m.phase == PhaseSelectMatchType {
			return ebiten.Termination
		}
		// Navigate back through phases, clearing the selection we're returning to
		switch m.phase {
		case PhaseSelectAlly2:
			m.allies[1] = -1
			m.phase = PhaseSelectAlly1
		case PhaseSelectAlly1:
			m.allies[0] = -1
			if m.matchType == engine.MatchTag {
				m.phase = PhaseSelectWrestler4
				m.selected[3] = -1
			} else {
				m.phase = PhaseSelectWrestler2
				m.selected[1] = -1
			}
		case PhaseSelectWrestler4:
			m.selected[3] = -1
			m.phase = PhaseSelectWrestler3
			m.selected[2] = -1
		case PhaseSelectWrestler3:
			m.selected[2] = -1
			m.phase = PhaseSelectWrestler2
			m.selected[1] = -1
		case PhaseSelectWrestler2:
			m.selected[1] = -1
			m.phase = PhaseSelectWrestler1
			m.selected[0] = -1
		default:
			m.phase--
		}
		m.cursor = 0
		return nil
	}

	switch m.phase {
	case PhaseSelectMatchType:
		m.cursor = handleListInput(m.cursor, len(menuOptions))
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			switch m.cursor {
			case menuNewCard:
				g.SetScreen(NewCardEditorScreen(nil))
				return nil
			case menuEditCard:
				m.phase = PhaseSelectWrestler1
				m.matchType = engine.MatchType(-1) // Sentinel for "edit mode"
				m.cursor = 0
				m.selected = [4]int{-1, -1, -1, -1}
				m.allies = [2]int{-1, -1}
			case menuFeud:
				m.matchType = engine.MatchSingles
				m.isFeud = true
				m.phase = PhaseSelectWrestler1
				m.cursor = 0
				m.selected = [4]int{-1, -1, -1, -1}
				m.allies = [2]int{-1, -1}
			default:
				m.matchType = matchTypes[m.cursor]
				m.isFeud = false
				m.phase = PhaseSelectWrestler1
				m.cursor = 0
				m.selected = [4]int{-1, -1, -1, -1}
				m.allies = [2]int{-1, -1}
			}
		}

	case PhaseSelectWrestler1:
		m.cursor = handleListInput(m.cursor, len(g.Roster))
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			if m.matchType == engine.MatchType(-1) {
				g.SetScreen(NewCardEditorScreen(g.Roster[m.cursor]))
				return nil
			}
			m.selected[0] = m.cursor
			m.phase = PhaseSelectWrestler2
			m.cursor = 0
		}

	case PhaseSelectWrestler2:
		m.cursor = handleListInput(m.cursor, len(g.Roster))
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			m.selected[1] = m.cursor
			if m.matchType == engine.MatchTag {
				m.phase = PhaseSelectWrestler3
			} else {
				// For non-tag matches, go to ally selection
				m.phase = PhaseSelectAlly1
			}
			m.cursor = 0
		}

	case PhaseSelectWrestler3:
		m.cursor = handleListInput(m.cursor, len(g.Roster))
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			m.selected[2] = m.cursor
			m.phase = PhaseSelectWrestler4
			m.cursor = 0
		}

	case PhaseSelectWrestler4:
		m.cursor = handleListInput(m.cursor, len(g.Roster))
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			m.selected[3] = m.cursor
			m.phase = PhaseReady
			m.cursor = 0
		}

	case PhaseSelectAlly1:
		// +1 for "No Ally" option at top
		m.cursor = handleListInput(m.cursor, len(g.Roster)+1)
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			if m.cursor == 0 {
				m.allies[0] = -1 // No ally
			} else {
				m.allies[0] = m.cursor - 1
			}
			m.phase = PhaseSelectAlly2
			m.cursor = 0
		}

	case PhaseSelectAlly2:
		m.cursor = handleListInput(m.cursor, len(g.Roster)+1)
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			if m.cursor == 0 {
				m.allies[1] = -1
			} else {
				m.allies[1] = m.cursor - 1
			}
			m.phase = PhaseReady
			m.cursor = 0
		}

	case PhaseReady:
		if m.matchType == engine.MatchTag {
			match := engine.NewTagMatch(
				g.Roster[m.selected[0]], g.Roster[m.selected[1]],
				g.Roster[m.selected[2]], g.Roster[m.selected[3]],
			)
			ms := NewTagMatchScreen(match, g)
			ms.RunMatch(g)
			g.SetScreen(ms)
		} else {
			card1 := g.Roster[m.selected[0]]
			card2 := g.Roster[m.selected[1]]
			ms := NewMatchScreen(card1, card2, m.matchType, g)
			// Set allies before running
			if m.allies[0] >= 0 && m.allies[0] < len(g.Roster) {
				ms.match.Sides[0].Ally = g.Roster[m.allies[0]]
			}
			if m.allies[1] >= 0 && m.allies[1] < len(g.Roster) {
				ms.match.Sides[1].Ally = g.Roster[m.allies[1]]
			}
			if m.isFeud {
				ms.match.IsFeud = true
			}
			ms.RunMatch(g)
			g.SetScreen(ms)
		}
	}

	return nil
}

func (m *MenuScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                      RING WARS", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	switch m.phase {
	case PhaseSelectMatchType:
		DrawText(screen, "MAIN MENU:", Margin, y)
		y += LineHeight * 2
		for i, name := range menuOptions {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			DrawText(screen, prefix+name, Margin, y)
			y += LineHeight
		}

	case PhaseSelectWrestler1:
		if m.matchType == engine.MatchType(-1) {
			DrawText(screen, "SELECT CARD TO EDIT:", Margin, y)
			y += LineHeight * 2
			m.drawRosterList(screen, g, y)
			break
		}
		label := "SELECT WRESTLER 1:"
		if m.matchType == engine.MatchTag {
			label = "SELECT TEAM 1 - WRESTLER A:"
		}
		DrawText(screen, label, Margin, y)
		y += LineHeight * 2
		m.drawRosterList(screen, g, y)

	case PhaseSelectWrestler2:
		m.drawSelectedSoFar(screen, g, &y)
		label := "SELECT WRESTLER 2:"
		if m.matchType == engine.MatchTag {
			label = "SELECT TEAM 1 - WRESTLER B:"
		}
		DrawText(screen, label, Margin, y)
		y += LineHeight * 2
		m.drawRosterList(screen, g, y)

	case PhaseSelectWrestler3:
		m.drawSelectedSoFar(screen, g, &y)
		DrawText(screen, "SELECT TEAM 2 - WRESTLER A:", Margin, y)
		y += LineHeight * 2
		m.drawRosterList(screen, g, y)

	case PhaseSelectWrestler4:
		m.drawSelectedSoFar(screen, g, &y)
		DrawText(screen, "SELECT TEAM 2 - WRESTLER B:", Margin, y)
		y += LineHeight * 2
		m.drawRosterList(screen, g, y)

	case PhaseSelectAlly1:
		m.drawSelectedSoFar(screen, g, &y)
		DrawText(screen, "SELECT RINGSIDE ALLY FOR WRESTLER 1 (optional):", Margin, y)
		y += LineHeight * 2
		m.drawAllyList(screen, g, y)

	case PhaseSelectAlly2:
		m.drawSelectedSoFar(screen, g, &y)
		if m.allies[0] >= 0 {
			DrawText(screen, fmt.Sprintf("Ally 1: %s", g.Roster[m.allies[0]].Name), Margin, y)
		} else {
			DrawText(screen, "Ally 1: None", Margin, y)
		}
		y += LineHeight * 2
		DrawText(screen, "SELECT RINGSIDE ALLY FOR WRESTLER 2 (optional):", Margin, y)
		y += LineHeight * 2
		m.drawAllyList(screen, g, y)
	}

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Confirm  [ESC] Back", Margin, statusY)
}

func (m *MenuScreen) drawSelectedSoFar(screen *ebiten.Image, g *Game, y *int) {
	matchLabel := "Singles"
	switch {
	case m.isFeud:
		matchLabel = "Feud"
	case m.matchType == engine.MatchTag:
		matchLabel = "Tag Team"
	case m.matchType == engine.MatchCage:
		matchLabel = "Cage"
	case m.matchType == engine.MatchNoDQ:
		matchLabel = "No DQ"
	}
	DrawText(screen, fmt.Sprintf("Match Type: %s", matchLabel), Margin, *y)
	*y += LineHeight

	for i, idx := range m.selected {
		if idx < 0 {
			break
		}
		labels := []string{"Wrestler 1", "Wrestler 2", "Wrestler 3", "Wrestler 4"}
		if m.matchType == engine.MatchTag {
			labels = []string{"Team 1A", "Team 1B", "Team 2A", "Team 2B"}
		}
		name := g.Roster[idx].Name
		if g.Injuries.IsInjured(name) {
			name += fmt.Sprintf(" [INJURED %d cards]", g.Injuries.InjuryCards(name))
		}
		DrawText(screen, fmt.Sprintf("%s: %s", labels[i], name), Margin, *y)
		*y += LineHeight
	}
	*y += LineHeight
}

func (m *MenuScreen) drawRosterList(screen *ebiten.Image, g *Game, y int) {
	for i, card := range g.Roster {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		name := card.Name
		if m.isSelected(i) {
			name += "  (already selected)"
		}
		if g.Injuries.IsInjured(card.Name) {
			name += fmt.Sprintf("  [INJURED %d]", g.Injuries.InjuryCards(card.Name))
		}
		DrawText(screen, prefix+name, Margin, y)
		y += LineHeight
	}
}

func (m *MenuScreen) drawAllyList(screen *ebiten.Image, g *Game, y int) {
	// First option: No Ally
	prefix := "  "
	if m.cursor == 0 {
		prefix = "> "
	}
	DrawText(screen, prefix+"No Ally", Margin, y)
	y += LineHeight

	// Show full roster with "(in match)" for participants
	for i, card := range g.Roster {
		prefix := "  "
		if i+1 == m.cursor {
			prefix = "> "
		}
		name := card.Name
		if m.isSelected(i) {
			name += "  (in match)"
		}
		DrawText(screen, prefix+name, Margin, y)
		y += LineHeight
	}
}

func handleListInput(cursor, length int) int {
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		cursor++
		if cursor >= length {
			cursor = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		cursor--
		if cursor < 0 {
			cursor = length - 1
		}
	}
	return cursor
}
