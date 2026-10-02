package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
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
	match.Rules = g.Rules

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
	match.Rules = g.Rules

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
	g.EndFightCard([]*engine.MatchResult{ms.match.Result()})
}

// matchResultBanner is the line shown under a finished match.
func matchResultBanner(result *engine.MatchResult) string {
	switch {
	case result == nil:
		return "  MATCH ENDED IN A DRAW"
	case result.Draw():
		return "  NO WINNER: DOUBLE DISQUALIFICATION"
	default:
		return "  WINNER: " + result.Winner + " by " + result.Method
	}
}

func (ms *MatchScreen) Update(g *Game) error {
	if g.in.JustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewMenuScreen())
		return nil
	}

	ms.updatePlaybackControls(g)
	if ms.shouldAdvance(g) && ms.shown < len(ms.events) {
		ms.showNextEvent(g)
	}

	if ms.state == MatchFinished && g.in.JustPressed(ebiten.KeyR) {
		next := ms.rematch(g)
		next.RunMatch(g)
		g.SetScreen(next)
	}
	return nil
}

const (
	fastestAutoPlay = 5
	autoPlayStep    = 5
)

func (ms *MatchScreen) updatePlaybackControls(g *Game) {
	if g.in.JustPressed(ebiten.KeyA) {
		ms.autoPlay = !ms.autoPlay
	}
	faster := g.in.JustPressed(ebiten.KeyEqual) || g.in.JustPressed(ebiten.KeyNumpadAdd)
	if faster && ms.speed > fastestAutoPlay {
		ms.speed -= autoPlayStep
	}
	if g.in.JustPressed(ebiten.KeyMinus) || g.in.JustPressed(ebiten.KeyNumpadSubtract) {
		ms.speed += autoPlayStep
	}
	if g.in.Pressed(ebiten.KeyUp) && ms.scroll > 0 {
		ms.scroll--
	}
	if g.in.Pressed(ebiten.KeyDown) && ms.scroll < ms.maxScroll(g) {
		ms.scroll++
	}
}

// shouldAdvance reports whether the next line of the match should be shown
// on this tick: on a key press, or when the auto-play timer comes round.
func (ms *MatchScreen) shouldAdvance(g *Game) bool {
	if confirmPressed(g.in) {
		return true
	}
	if !ms.autoPlay || ms.state != MatchRunning {
		return false
	}
	ms.ticker++
	if ms.ticker < ms.speed {
		return false
	}
	ms.ticker = 0
	return true
}

func (ms *MatchScreen) showNextEvent(g *Game) {
	ms.lines = append(ms.lines, ms.events[ms.shown].Text)
	ms.shown++

	if ms.shown >= len(ms.events) {
		ms.state = MatchFinished
		ms.lines = append(ms.lines,
			"",
			"============================================================",
			matchResultBanner(ms.match.Result()),
			"============================================================",
			"",
			"Press [R] for rematch, [ESC] for menu",
		)
	}
	ms.scrollToBottom(g)
}

// rematch builds a fresh match with the same wrestlers, teams, allies and
// match type as the one just finished.
func (ms *MatchScreen) rematch(g *Game) *MatchScreen {
	old := ms.match
	first, second := old.Sides[0], old.Sides[1]

	if len(first.Wrestlers) > 1 && len(second.Wrestlers) > 1 {
		match := engine.NewTagMatch(
			first.Wrestlers[0].Card, first.Wrestlers[1].Card,
			second.Wrestlers[0].Card, second.Wrestlers[1].Card,
		)
		match.Sides[0].RegularPartners = first.RegularPartners
		match.Sides[1].RegularPartners = second.RegularPartners
		return NewTagMatchScreen(match, g)
	}

	next := NewMatchScreen(first.Wrestlers[0].Card, second.Wrestlers[0].Card, old.Type, g)
	next.match.Sides[0].Ally = first.Ally
	next.match.Sides[1].Ally = second.Ally
	next.match.IsFeud = old.IsFeud
	return next
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
