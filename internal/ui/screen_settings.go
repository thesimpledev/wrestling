package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
	"wrestling/internal/loader"
)

type settingsItem struct {
	label  string
	detail string
	value  func(rules *engine.Rules) *bool
}

var settingsItems = []settingsItem{
	{
		label:  "Skip risky (c) chart moves",
		detail: "A chart move marked (c) is passed up when the opponent is rated A on that chart.",
		value:  func(r *engine.Rules) *bool { return &r.ChartChoice },
	},
	{
		label:  "Skip risky dis moves",
		detail: "A dis move is passed up when its disqualification number is 6 or higher.",
		value:  func(r *engine.Rules) *bool { return &r.AvoidDisMoves },
	},
	{
		label:  "Use the number printed with dis",
		detail: "A move such as dis 7 rolls against 7 in place of the wrestler's DQ rating.",
		value:  func(r *engine.Rules) *bool { return &r.CustomDisNumbers },
	},
}

// SettingsScreen switches the optional rules on and off. Each change is
// saved straight away.
type SettingsScreen struct {
	cursor int
}

func NewSettingsScreen() *SettingsScreen {
	return &SettingsScreen{}
}

func (s *SettingsScreen) Update(g *Game) error {
	if g.in.JustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewMenuScreen())
		return nil
	}
	s.cursor = handleListInput(g.in, s.cursor, len(settingsItems))
	if g.in.JustPressed(ebiten.KeyEnter) || g.in.JustPressed(ebiten.KeySpace) {
		value := settingsItems[s.cursor].value(&g.Rules)
		*value = !*value
		g.SaveRules()
	}
	return nil
}

func settingLine(item settingsItem, rules *engine.Rules, selected bool) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	state := "[OFF]"
	if *item.value(rules) {
		state = "[ON ]"
	}
	return prefix + state + " " + item.label
}

func (s *SettingsScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)
	y := Margin

	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight
	DrawText(screen, "                      SETTINGS", Margin, y)
	y += LineHeight
	DrawText(screen, "============================================================", Margin, y)
	y += LineHeight * 2

	DrawText(screen, "OPTIONAL RULES:", Margin, y)
	y += LineHeight * 2
	for i, item := range settingsItems {
		DrawText(screen, settingLine(item, &g.Rules, i == s.cursor), Margin, y)
		y += LineHeight
	}
	y += LineHeight
	DrawText(screen, settingsItems[s.cursor].detail, Margin, y)

	statusY := g.screenH - LineHeight - Margin
	DrawText(screen, "[UP/DOWN] Select  [ENTER] Switch on/off  [ESC] Back", Margin, statusY)
}

// SaveRules persists the optional-rule settings and shows a banner if it fails.
func (g *Game) SaveRules() {
	if err := loader.SaveRules(g.Store, g.Rules); err != nil {
		g.SetNotice("SAVE FAILED (settings): " + err.Error())
	}
}
