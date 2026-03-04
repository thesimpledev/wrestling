package ui

import (
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"wrestling/engine"
	"wrestling/loader"
)

type BRPhase int

const (
	BRIntro BRPhase = iota
	BRShowingBracket
	BRRunningMatch
	BRMatchResult
	BRFinished
)

type BattleRoyalScreen struct {
	phase           BRPhase
	wrestlers       []*engine.WrestlerCard // shuffled entrants
	eliminated      []string               // names in elimination order
	champion        *engine.WrestlerCard
	championFatigue int // accumulated PIN carry-over
	nextIdx         int // next challenger index (starts at 1, champion is wrestlers[0])

	// Sub-match display
	match    *engine.Match
	events   []engine.Event
	shown    int
	lines    []string
	scroll   int
	autoPlay bool
	speed    int
	ticker   int
}

func NewBattleRoyalScreen(wrestlers []*engine.WrestlerCard, g *Game) *BattleRoyalScreen {
	// Shuffle the entrants
	shuffled := make([]*engine.WrestlerCard, len(wrestlers))
	copy(shuffled, wrestlers)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return &BattleRoyalScreen{
		phase:     BRIntro,
		wrestlers: shuffled,
		nextIdx:   1,
		speed:     30,
	}
}

func (b *BattleRoyalScreen) Update(g *Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewMenuScreen())
		return nil
	}

	switch b.phase {
	case BRIntro:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			b.champion = b.wrestlers[0]
			b.phase = BRShowingBracket
		}

	case BRShowingBracket:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			b.startNextMatch(g)
		}

	case BRRunningMatch:
		b.updateMatch(g)

	case BRMatchResult:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if b.nextIdx >= len(b.wrestlers) {
				b.phase = BRFinished
			} else {
				b.phase = BRShowingBracket
			}
		}

	case BRFinished:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.SetScreen(NewMenuScreen())
		}
	}

	return nil
}

func (b *BattleRoyalScreen) startNextMatch(g *Game) {
	challenger := b.wrestlers[b.nextIdx]

	match := engine.NewMatch(b.champion, challenger)
	match.Type = engine.MatchSingles
	match.InitForMatchType()
	match.ApplyInjuries(g.Injuries.IsInjured)

	// Apply fatigue carry-forward to champion
	if b.championFatigue > 0 {
		match.Sides[0].Wrestlers[0].CurrentPIN += b.championFatigue
	}

	b.match = match
	b.events = match.Run()
	b.shown = 0
	b.autoPlay = false
	b.ticker = 0
	b.scroll = 0
	b.lines = []string{
		"============================================================",
		fmt.Sprintf("  BATTLE ROYAL — Round %d of %d", b.nextIdx, len(b.wrestlers)-1),
		fmt.Sprintf("  %s  vs  %s", b.champion.Name, challenger.Name),
		"============================================================",
		"",
	}
	b.phase = BRRunningMatch
}

func (b *BattleRoyalScreen) updateMatch(g *Game) {
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		b.autoPlay = !b.autoPlay
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		if b.speed > 5 {
			b.speed -= 5
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		b.speed += 5
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		if b.scroll > 0 {
			b.scroll--
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		max := b.maxScroll(g)
		if b.scroll < max {
			b.scroll++
		}
	}

	advance := false
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		advance = true
	}
	if b.autoPlay {
		b.ticker++
		if b.ticker >= b.speed {
			b.ticker = 0
			advance = true
		}
	}

	if advance && b.shown < len(b.events) {
		e := b.events[b.shown]
		b.lines = append(b.lines, e.Text)
		b.shown++
		b.scrollToBottom(g)

		if b.shown >= len(b.events) {
			b.finishSubMatch(g)
		}
	}
}

func (b *BattleRoyalScreen) finishSubMatch(g *Game) {
	result := b.match.Result()

	// Record injuries from this sub-match
	if result != nil && result.InjuredWrestler != "" && result.InjuryCards > 0 {
		g.Injuries.RecordInjury(result.InjuredWrestler, result.InjuryCards)
	}

	if result != nil {
		b.eliminated = append(b.eliminated, result.Loser)

		// Determine new champion
		if result.Winner == b.champion.Name {
			// Champion retained — accumulate fatigue
			champState := b.match.Sides[0].Wrestlers[0]
			b.championFatigue = champState.CurrentPIN - b.champion.PINAdv
		} else {
			// New champion
			challenger := b.wrestlers[b.nextIdx]
			b.champion = challenger
			champState := b.match.Sides[1].Wrestlers[0]
			b.championFatigue = champState.CurrentPIN - challenger.PINAdv
		}

		b.lines = append(b.lines, "")
		b.lines = append(b.lines, "============================================================")
		b.lines = append(b.lines, fmt.Sprintf("  %s WINS by %s!", result.Winner, result.Method))
		b.lines = append(b.lines, fmt.Sprintf("  %s is ELIMINATED!", result.Loser))
		b.lines = append(b.lines, "============================================================")
	} else {
		// Draw — champion stays
		b.lines = append(b.lines, "")
		b.lines = append(b.lines, "  Match ended in a draw — champion retains!")
	}

	b.nextIdx++
	b.scrollToBottom(g)

	// Decrement injuries once for the entire battle royal
	if b.nextIdx >= len(b.wrestlers) {
		g.Injuries.DecrementAll()
		loader.SaveInjuries(g.Store, g.Injuries)
	}

	b.phase = BRMatchResult
}

func (b *BattleRoyalScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)

	switch b.phase {
	case BRIntro:
		b.drawIntro(screen, g)
	case BRShowingBracket:
		b.drawStandings(screen, g)
	case BRRunningMatch, BRMatchResult:
		b.drawMatch(screen, g)
	case BRFinished:
		b.drawFinished(screen, g)
	}
}

func (b *BattleRoyalScreen) drawIntro(screen *ebiten.Image, g *Game) {
	y := Margin
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                    BATTLE ROYAL", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	DrawText(screen, "Gauntlet-style elimination — two fight, loser out,", Margin, y)
	y += LineHeight
	DrawText(screen, "winner stays and faces next challenger!", Margin, y)
	y += LineHeight * 2

	DrawText(screen, "LINEUP:", Margin, y)
	y += LineHeight
	for i, w := range b.wrestlers {
		DrawText(screen, fmt.Sprintf("  %d. %s", i+1, w.Name), Margin, y)
		y += LineHeight
	}

	y += LineHeight
	DrawText(screen, fmt.Sprintf("First match: %s vs %s", b.wrestlers[0].Name, b.wrestlers[1].Name), Margin, y)

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[SPACE] Start  [ESC] Back", Margin, statusY)
}

func (b *BattleRoyalScreen) drawStandings(screen *ebiten.Image, g *Game) {
	y := Margin
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                  BATTLE ROYAL STANDINGS", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	DrawText(screen, fmt.Sprintf("CHAMPION: %s", b.champion.Name), Margin, y)
	if b.championFatigue > 0 {
		DrawText(screen, fmt.Sprintf("  (fatigue: +%d PIN)", b.championFatigue), Margin+30*CharWidth, y)
	}
	y += LineHeight * 2

	if len(b.eliminated) > 0 {
		DrawText(screen, "ELIMINATED:", Margin, y)
		y += LineHeight
		for i, name := range b.eliminated {
			DrawText(screen, fmt.Sprintf("  %d. %s", i+1, name), Margin, y)
			y += LineHeight
		}
		y += LineHeight
	}

	if b.nextIdx < len(b.wrestlers) {
		DrawText(screen, fmt.Sprintf("NEXT CHALLENGER: %s", b.wrestlers[b.nextIdx].Name), Margin, y)
		y += LineHeight

		remaining := len(b.wrestlers) - b.nextIdx
		DrawText(screen, fmt.Sprintf("Challengers remaining: %d", remaining), Margin, y)
	}

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[SPACE] Start Next Match  [ESC] Quit", Margin, statusY)
}

func (b *BattleRoyalScreen) drawMatch(screen *ebiten.Image, g *Game) {
	statusBarY := g.screenH - LineHeight - Margin

	startLine := b.scroll
	if startLine > len(b.lines)-1 {
		startLine = len(b.lines) - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	y := Margin
	for i := startLine; i < len(b.lines) && y < statusBarY; i++ {
		DrawText(screen, b.lines[i], Margin, y)
		y += LineHeight
	}

	var status string
	if b.phase == BRMatchResult {
		if b.nextIdx >= len(b.wrestlers) {
			status = "[SPACE] See Final Results  [ESC] Quit"
		} else {
			status = "[SPACE] Back to Standings  [ESC] Quit"
		}
	} else if b.autoPlay {
		status = "AUTO-PLAY ON  [A] Stop  [+/-] Speed  [ESC] Quit"
	} else {
		status = "[SPACE] Step  [A] Auto-play  [+/-] Speed  [ESC] Quit"
	}
	DrawText(screen, status, Margin, statusBarY)
}

func (b *BattleRoyalScreen) drawFinished(screen *ebiten.Image, g *Game) {
	y := Margin
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                  BATTLE ROYAL COMPLETE!", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	DrawText(screen, fmt.Sprintf("CHAMPION: %s", b.champion.Name), Margin, y)
	y += LineHeight * 2

	DrawText(screen, "ELIMINATION ORDER:", Margin, y)
	y += LineHeight
	for i, name := range b.eliminated {
		DrawText(screen, fmt.Sprintf("  %d. %s", i+1, name), Margin, y)
		y += LineHeight
	}

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[SPACE] Return to Menu  [ESC] Quit", Margin, statusY)
}

func (b *BattleRoyalScreen) visibleLines(g *Game) int {
	if g.screenH == 0 {
		return 20
	}
	return (g.screenH - Margin*2 - LineHeight) / LineHeight
}

func (b *BattleRoyalScreen) maxScroll(g *Game) int {
	max := len(b.lines) - b.visibleLines(g)
	if max < 0 {
		return 0
	}
	return max
}

func (b *BattleRoyalScreen) scrollToBottom(g *Game) {
	b.scroll = b.maxScroll(g)
}
