package ui

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"gopkg.in/yaml.v3"
	"wrestling/engine"
)

// Field types for the editor
type FieldType int

const (
	FieldString FieldType = iota
	FieldInt
	FieldRating // A, B, or C
	FieldMove
	FieldDefense
)

type EditorField struct {
	Label string
	Value string
	Type  FieldType
}

type CardEditorScreen struct {
	fields   []EditorField
	cursor   int
	scroll   int
	editing  bool
	message  string
	msgTimer int
}

func NewCardEditorScreen(card *engine.WrestlerCard) *CardEditorScreen {
	e := &CardEditorScreen{}
	if card != nil {
		e.loadCard(card)
	} else {
		e.loadDefaults()
	}
	return e
}

func (e *CardEditorScreen) loadDefaults() {
	e.fields = []EditorField{
		{Label: "Name", Value: "New Wrestler", Type: FieldString},
		{Label: "--- RATINGS ---", Value: "", Type: FieldString},
		{Label: "Ropes", Value: "B", Type: FieldRating},
		{Label: "Turnbuckle", Value: "B", Type: FieldRating},
		{Label: "Ring", Value: "B", Type: FieldRating},
		{Label: "Deathjump", Value: "B", Type: FieldRating},
		{Label: "PIN", Value: "5", Type: FieldInt},
		{Label: "PIN Adv", Value: "3", Type: FieldInt},
		{Label: "Cage", Value: "5", Type: FieldInt},
		{Label: "DQ", Value: "4", Type: FieldInt},
		{Label: "Agility", Value: "0", Type: FieldInt},
		{Label: "Power", Value: "0", Type: FieldInt},
		{Label: "Distractor", Value: "5", Type: FieldInt},
		{Label: "--- FINISHER ---", Value: "", Type: FieldString},
		{Label: "Finisher Name", Value: "FINISHING MOVE", Type: FieldString},
		{Label: "Finisher Rating", Value: "3", Type: FieldInt},
	}
	// Add offense moves (3 levels x 6 moves)
	for lvl := 1; lvl <= 3; lvl++ {
		e.fields = append(e.fields, EditorField{
			Label: fmt.Sprintf("--- OFFENSE LEVEL %d ---", lvl), Type: FieldString,
		})
		for slot := 1; slot <= 6; slot++ {
			e.fields = append(e.fields, EditorField{
				Label: fmt.Sprintf("L%d Move %d (name,power,deflvl)", lvl, slot),
				Value: fmt.Sprintf("Move %d,%d,%d", slot, 1, 1),
				Type:  FieldMove,
			})
		}
	}
	// Add defense outcomes (3 levels x 6 outcomes)
	for lvl := 1; lvl <= 3; lvl++ {
		e.fields = append(e.fields, EditorField{
			Label: fmt.Sprintf("--- DEFENSE LEVEL %d ---", lvl), Type: FieldString,
		})
		for slot := 1; slot <= 6; slot++ {
			e.fields = append(e.fields, EditorField{
				Label: fmt.Sprintf("L%d Def %d (type,power)", lvl, slot),
				Value: "dazed,1",
				Type:  FieldDefense,
			})
		}
	}
}

func (e *CardEditorScreen) loadCard(card *engine.WrestlerCard) {
	ratingStr := func(r engine.Rating) string {
		return r.String()
	}
	e.fields = []EditorField{
		{Label: "Name", Value: card.Name, Type: FieldString},
		{Label: "--- RATINGS ---", Value: "", Type: FieldString},
		{Label: "Ropes", Value: ratingStr(card.Ropes), Type: FieldRating},
		{Label: "Turnbuckle", Value: ratingStr(card.Turnbuckle), Type: FieldRating},
		{Label: "Ring", Value: ratingStr(card.Ring), Type: FieldRating},
		{Label: "Deathjump", Value: ratingStr(card.Deathjump), Type: FieldRating},
		{Label: "PIN", Value: fmt.Sprintf("%d", card.PIN), Type: FieldInt},
		{Label: "PIN Adv", Value: fmt.Sprintf("%d", card.PINAdv), Type: FieldInt},
		{Label: "Cage", Value: fmt.Sprintf("%d", card.Cage), Type: FieldInt},
		{Label: "DQ", Value: fmt.Sprintf("%d", card.DQ), Type: FieldInt},
		{Label: "Agility", Value: fmt.Sprintf("%d", card.Agility), Type: FieldInt},
		{Label: "Power", Value: fmt.Sprintf("%d", card.Power), Type: FieldInt},
		{Label: "Distractor", Value: fmt.Sprintf("%d", card.Distractor), Type: FieldInt},
		{Label: "--- FINISHER ---", Value: "", Type: FieldString},
		{Label: "Finisher Name", Value: card.Finisher.Name, Type: FieldString},
		{Label: "Finisher Rating", Value: fmt.Sprintf("%d", card.Finisher.Rating), Type: FieldInt},
	}
	for lvl := 0; lvl < 3; lvl++ {
		e.fields = append(e.fields, EditorField{
			Label: fmt.Sprintf("--- OFFENSE LEVEL %d ---", lvl+1), Type: FieldString,
		})
		for slot := 0; slot < 6; slot++ {
			mv := card.Offense[lvl][slot]
			e.fields = append(e.fields, EditorField{
				Label: fmt.Sprintf("L%d Move %d (name,power,deflvl)", lvl+1, slot+1),
				Value: fmt.Sprintf("%s,%d,%d", mv.Name, mv.Power, mv.DefLevel),
				Type:  FieldMove,
			})
		}
	}
	for lvl := 0; lvl < 3; lvl++ {
		e.fields = append(e.fields, EditorField{
			Label: fmt.Sprintf("--- DEFENSE LEVEL %d ---", lvl+1), Type: FieldString,
		})
		for slot := 0; slot < 6; slot++ {
			def := card.Defense[lvl][slot]
			e.fields = append(e.fields, EditorField{
				Label: fmt.Sprintf("L%d Def %d (type,power)", lvl+1, slot+1),
				Value: fmt.Sprintf("%s,%d", def.Type, def.Power),
				Type:  FieldDefense,
			})
		}
	}
}

func (e *CardEditorScreen) Update(g *Game) error {
	if e.msgTimer > 0 {
		e.msgTimer--
		if e.msgTimer == 0 {
			e.message = ""
		}
	}

	if e.editing {
		return e.updateEditing(g)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewMenuScreen())
		return nil
	}

	e.cursor = handleListInput(e.cursor, len(e.fields))
	// Skip separator lines
	for e.fields[e.cursor].Label[0] == '-' {
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			e.cursor--
			if e.cursor < 0 {
				e.cursor = len(e.fields) - 1
			}
		} else {
			e.cursor++
			if e.cursor >= len(e.fields) {
				e.cursor = 0
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		e.editing = true
	}

	// Save with Ctrl+S
	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		e.saveCard(g)
	}

	return nil
}

func (e *CardEditorScreen) updateEditing(g *Game) error {
	field := &e.fields[e.cursor]

	// Handle rating fields specially
	if field.Type == FieldRating {
		if inpututil.IsKeyJustPressed(ebiten.KeyA) {
			field.Value = "A"
			e.editing = false
		} else if inpututil.IsKeyJustPressed(ebiten.KeyB) {
			field.Value = "B"
			e.editing = false
		} else if inpututil.IsKeyJustPressed(ebiten.KeyC) {
			field.Value = "C"
			e.editing = false
		} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			e.editing = false
		}
		return nil
	}

	// General text input
	var chars []rune
	chars = ebiten.AppendInputChars(chars)
	for _, c := range chars {
		field.Value += string(c)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(field.Value) > 0 {
		field.Value = field.Value[:len(field.Value)-1]
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		e.editing = false
	}

	return nil
}

func (e *CardEditorScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)

	visibleLines := (g.screenH - Margin*2 - LineHeight*3) / LineHeight
	if visibleLines < 5 {
		visibleLines = 5
	}

	// Auto-scroll to keep cursor visible
	if e.cursor < e.scroll {
		e.scroll = e.cursor
	}
	if e.cursor >= e.scroll+visibleLines {
		e.scroll = e.cursor - visibleLines + 1
	}

	y := Margin
	DrawText(screen, "CARD EDITOR  [ENTER] Edit Field  [Ctrl+S] Save  [ESC] Back", Margin, y)
	y += LineHeight * 2

	endIdx := e.scroll + visibleLines
	if endIdx > len(e.fields) {
		endIdx = len(e.fields)
	}

	for i := e.scroll; i < endIdx; i++ {
		f := e.fields[i]
		prefix := "  "
		if i == e.cursor {
			prefix = "> "
		}

		if f.Label[0] == '-' {
			DrawText(screen, "  "+f.Label, Margin, y)
		} else {
			suffix := ""
			if i == e.cursor && e.editing {
				suffix = "_"
			}
			DrawText(screen, fmt.Sprintf("%s%-28s %s%s", prefix, f.Label+":", f.Value, suffix), Margin, y)
		}
		y += LineHeight
	}

	if e.message != "" {
		DrawText(screen, e.message, Margin, g.screenH-LineHeight*2-Margin)
	}
	DrawText(screen, fmt.Sprintf("Field %d/%d", e.cursor+1, len(e.fields)), Margin, g.screenH-LineHeight-Margin)
}

func (e *CardEditorScreen) validateCard() string {
	name := strings.TrimSpace(e.fieldValue("Name"))
	if name == "" {
		return "Name cannot be empty"
	}

	// Validate ratings
	for _, r := range []string{"Ropes", "Turnbuckle", "Ring", "Deathjump"} {
		v := strings.ToUpper(strings.TrimSpace(e.fieldValue(r)))
		if v != "A" && v != "B" && v != "C" {
			return fmt.Sprintf("%s must be A, B, or C (got %q)", r, v)
		}
	}

	// Validate numeric fields
	type intRange struct {
		label    string
		min, max int
	}
	checks := []intRange{
		{"PIN", 1, 12}, {"PIN Adv", 1, 12}, {"Cage", 1, 12},
		{"DQ", 1, 12}, {"Agility", -5, 5}, {"Power", -5, 5},
		{"Distractor", 1, 12}, {"Finisher Rating", 0, 8},
	}
	for _, c := range checks {
		v := e.fieldInt(c.label)
		if v < c.min || v > c.max {
			return fmt.Sprintf("%s must be %d to %d (got %d)", c.label, c.min, c.max, v)
		}
	}

	// Validate finisher name
	if strings.TrimSpace(e.fieldValue("Finisher Name")) == "" {
		return "Finisher Name cannot be empty"
	}

	// Validate offense moves
	for lvl := 1; lvl <= 3; lvl++ {
		for slot := 1; slot <= 6; slot++ {
			label := fmt.Sprintf("L%d Move %d (name,power,deflvl)", lvl, slot)
			val := e.fieldValue(label)
			parts := strings.SplitN(val, ",", 3)
			if len(parts) < 3 {
				return fmt.Sprintf("L%d Move %d: use format name,power,deflvl", lvl, slot)
			}
			moveName := strings.TrimSpace(parts[0])
			if moveName == "" {
				return fmt.Sprintf("L%d Move %d: name cannot be empty", lvl, slot)
			}
			var power, defLvl int
			fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &power)
			fmt.Sscanf(strings.TrimSpace(parts[2]), "%d", &defLvl)
			if power < 1 || power > 3 {
				return fmt.Sprintf("L%d Move %d: power must be 1-3 (got %d)", lvl, slot, power)
			}
			if defLvl < 1 || defLvl > 3 {
				return fmt.Sprintf("L%d Move %d: def_level must be 1-3 (got %d)", lvl, slot, defLvl)
			}
		}
	}

	// Validate defense outcomes
	validTypes := map[string]bool{
		"dazed": true, "hurt": true, "down": true, "reversal": true, "pin": true,
	}
	for lvl := 1; lvl <= 3; lvl++ {
		for slot := 1; slot <= 6; slot++ {
			label := fmt.Sprintf("L%d Def %d (type,power)", lvl, slot)
			val := e.fieldValue(label)
			parts := strings.SplitN(val, ",", 2)
			if len(parts) < 2 {
				return fmt.Sprintf("L%d Def %d: use format type,power", lvl, slot)
			}
			dtype := strings.ToLower(strings.TrimSpace(parts[0]))
			if !validTypes[dtype] {
				return fmt.Sprintf("L%d Def %d: type must be dazed/hurt/down/reversal/pin", lvl, slot)
			}
			var power int
			fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &power)
			if power < 1 || power > 3 {
				return fmt.Sprintf("L%d Def %d: power must be 1-3 (got %d)", lvl, slot, power)
			}
		}
	}

	return "" // No errors
}

func (e *CardEditorScreen) saveCard(g *Game) {
	// Validate before saving
	if errMsg := e.validateCard(); errMsg != "" {
		e.message = "Validation: " + errMsg
		e.msgTimer = 300
		return
	}

	// Build YAML structure from fields
	data := make(map[string]any)
	data["name"] = e.fields[0].Value

	data["ropes"] = e.fieldValue("Ropes")
	data["turnbuckle"] = e.fieldValue("Turnbuckle")
	data["ring"] = e.fieldValue("Ring")
	data["deathjump"] = e.fieldValue("Deathjump")
	data["pin"] = e.fieldInt("PIN")
	data["pin_adv"] = e.fieldInt("PIN Adv")
	data["cage"] = e.fieldInt("Cage")
	data["dq"] = e.fieldInt("DQ")
	data["agility"] = e.fieldInt("Agility")
	data["power"] = e.fieldInt("Power")
	data["distractor"] = e.fieldInt("Distractor")

	data["finisher"] = map[string]any{
		"name":   e.fieldValue("Finisher Name"),
		"rating": e.fieldInt("Finisher Rating"),
	}

	// Offense
	var offense [3][6]map[string]any
	for lvl := 0; lvl < 3; lvl++ {
		for slot := 0; slot < 6; slot++ {
			label := fmt.Sprintf("L%d Move %d (name,power,deflvl)", lvl+1, slot+1)
			val := e.fieldValue(label)
			parts := strings.SplitN(val, ",", 3)
			name := val
			power := 1
			defLvl := 1
			if len(parts) >= 3 {
				name = strings.TrimSpace(parts[0])
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &power)
				fmt.Sscanf(strings.TrimSpace(parts[2]), "%d", &defLvl)
			}
			offense[lvl][slot] = map[string]any{
				"name":      name,
				"power":     power,
				"def_level": defLvl,
			}
		}
	}
	data["offense"] = offense

	// Defense
	var defense [3][6]map[string]any
	for lvl := 0; lvl < 3; lvl++ {
		for slot := 0; slot < 6; slot++ {
			label := fmt.Sprintf("L%d Def %d (type,power)", lvl+1, slot+1)
			val := e.fieldValue(label)
			parts := strings.SplitN(val, ",", 2)
			dtype := "dazed"
			power := 1
			if len(parts) >= 2 {
				dtype = strings.TrimSpace(parts[0])
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &power)
			}
			defense[lvl][slot] = map[string]any{
				"type":  dtype,
				"power": power,
			}
		}
	}
	data["defense"] = defense

	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		e.message = "Error: " + err.Error()
		e.msgTimer = 180
		return
	}

	filename := strings.ToLower(strings.ReplaceAll(e.fields[0].Value, " ", "_")) + ".yaml"
	err = g.Store.SaveCardBytes(filename, yamlBytes)
	if err != nil {
		e.message = "Error saving: " + err.Error()
		e.msgTimer = 180
		return
	}

	e.message = "Saved " + filename
	e.msgTimer = 180

	// Reload roster
	reloadRoster(g)
}

func (e *CardEditorScreen) fieldValue(label string) string {
	for _, f := range e.fields {
		if f.Label == label {
			return f.Value
		}
	}
	return ""
}

func (e *CardEditorScreen) fieldInt(label string) int {
	val := e.fieldValue(label)
	var n int
	fmt.Sscanf(val, "%d", &n)
	return n
}
