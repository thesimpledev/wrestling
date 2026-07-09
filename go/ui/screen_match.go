package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
	"wrestling/loader"
)

type MatchState int

const (
	MatchRunning MatchState = iota
	MatchFinished
)

type MatchScreen struct {
	match    *engine.Match
	events   []engine.Event
	shown    int
	state    MatchState
	autoPlay bool
	speed    int
	ticker   int
	lines    []string
	scroll   int
}

func NewMatchScreen(card1, card2 *engine.WrestlerCard, matchType engine.MatchType, g *Game) *MatchScreen {
	match := engine.NewMatch(card1, card2)
	match.Type = matchType
	match.InitForMatchType()

	return &MatchScreen{
		match: match,
		speed: 30,
		lines: []string{
			"============================================================",
			"                     RING WARS MATCH",
			"============================================================",
			"",
			"[SPACE] Step  [A] Auto-play  [+/-] Speed",
			"",
		},
	}
}

func NewTagMatchScreen(match *engine.Match, g *Game) *MatchScreen {
	match.InitForMatchType()

	return &MatchScreen{
		match: match,
		speed: 30,
		lines: []string{
			"============================================================",
			"                 RING WARS TAG TEAM MATCH",
			"============================================================",
			"",
			"[SPACE] Step  [A] Auto-play  [+/-] Speed",
			"",
		},
	}
}

// RunMatch applies injuries, runs the match, and saves results.
// Call after setting allies and feud flag on the match.
func (ms *MatchScreen) RunMatch(g *Game) {
	ms.match.ApplyInjuries(g.Injuries.IsInjured)
	ms.events = ms.match.Run()
	ms.saveInjuries(g)
}

// saveInjuries records any injuries from this match and decrements existing injuries.
func (ms *MatchScreen) saveInjuries(g *Game) {
	if result := ms.match.Result(); result != nil {
		if result.InjuredWrestler != "" && result.InjuryCards > 0 {
			g.Injuries.RecordInjury(result.InjuredWrestler, result.InjuryCards)
		}
	}
	// Decrement existing injuries (one match = one fight card)
	g.Injuries.DecrementAll()
	loader.SaveInjuries(g.Store, g.Injuries)
}

func (ms *MatchScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if ms.state == MatchFinished {
			g.SetScreen(NewMenuScreen())
			return nil
		}
		// During match, ESC goes back to menu
		g.SetScreen(NewMenuScreen())
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		ms.autoPlay = !ms.autoPlay
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		if ms.speed > 5 {
			ms.speed -= 5
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		ms.speed += 5
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		if ms.scroll > 0 {
			ms.scroll--
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		max := ms.maxScroll(g)
		if ms.scroll < max {
			ms.scroll++
		}
	}

	advance := false
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		advance = true
	}
	if ms.autoPlay && ms.state == MatchRunning {
		ms.ticker++
		if ms.ticker >= ms.speed {
			ms.ticker = 0
			advance = true
		}
	}

	if advance && ms.shown < len(ms.events) {
		e := ms.events[ms.shown]
		ms.lines = append(ms.lines, e.Text)
		ms.shown++
		ms.scrollToBottom(g)

		if ms.shown >= len(ms.events) {
			ms.state = MatchFinished
			ms.lines = append(ms.lines, "")
			ms.lines = append(ms.lines, "============================================================")
			if result := ms.match.Result(); result != nil {
				ms.lines = append(ms.lines, "  WINNER: "+result.Winner+" by "+result.Method)
			} else {
				ms.lines = append(ms.lines, "  MATCH ENDED IN A DRAW")
			}
			ms.lines = append(ms.lines, "============================================================")
			ms.lines = append(ms.lines, "")
			ms.lines = append(ms.lines, "Press [R] for rematch, [ESC] for menu")
			ms.scrollToBottom(g)
		}
	}

	// Rematch
	if ms.state == MatchFinished && inpututil.IsKeyJustPressed(ebiten.KeyR) {
		card1 := ms.match.Sides[0].Active().Card
		card2 := ms.match.Sides[1].Active().Card
		newMs := NewMatchScreen(card1, card2, ms.match.Type, g)
		// Preserve allies and feud flag
		newMs.match.Sides[0].Ally = ms.match.Sides[0].Ally
		newMs.match.Sides[1].Ally = ms.match.Sides[1].Ally
		newMs.match.IsFeud = ms.match.IsFeud
		newMs.RunMatch(g)
		g.SetScreen(newMs)
	}

	return nil
}

func (ms *MatchScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)

	statusBarY := g.screenH - LineHeight - Margin

	startLine := ms.scroll
	if startLine > len(ms.lines)-1 {
		startLine = len(ms.lines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	y := Margin
	for i := startLine; i < len(ms.lines) && y < statusBarY; i++ {
		DrawText(screen, ms.lines[i], Margin, y)
		y += LineHeight
	}

	// Status bar
	status := "[SPACE] Step  [A] Auto-play  [+/-] Speed  [ESC] Menu"
	if ms.autoPlay {
		status = "AUTO-PLAY ON  [A] Stop  [+/-] Speed  [ESC] Menu"
	}
	if ms.state == MatchFinished {
		status = "[R] Rematch  [ESC] Menu  [UP/DOWN] Scroll"
	}
	DrawText(screen, status, Margin, g.screenH-LineHeight-Margin)
}

func (ms *MatchScreen) visibleLines(g *Game) int {
	if g.screenH == 0 {
		return 20
	}
	return (g.screenH - Margin*2 - LineHeight) / LineHeight
}

func (ms *MatchScreen) maxScroll(g *Game) int {
	max := len(ms.lines) - ms.visibleLines(g)
	if max < 0 {
		return 0
	}
	return max
}

func (ms *MatchScreen) scrollToBottom(g *Game) {
	ms.scroll = ms.maxScroll(g)
}
