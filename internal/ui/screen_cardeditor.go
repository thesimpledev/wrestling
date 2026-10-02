package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"wrestling/internal/engine"
	"wrestling/internal/loader"
)

// Field types for the editor
type FieldType int

const (
	FieldString FieldType = iota
	FieldInt
	FieldRating // A, B, or C
	FieldMove
	FieldDefense
	FieldHeading
)

type EditorField struct {
	Label string
	Value string
	Type  FieldType
}

const (
	finisherRollLabel = "Finisher Roll (min-max)"
	messageTicks      = 300
)

func moveLabel(level, slot int) string {
	return fmt.Sprintf("L%d Move %d (name,power,deflvl,extras)", level, slot)
}

func defenseLabel(level, slot int) string {
	return fmt.Sprintf("L%d Def %d (type,power,extras)", level, slot)
}

// CardEditorScreen edits one wrestler card as a list of text fields.
type CardEditorScreen struct {
	fields   []EditorField
	cursor   int
	scroll   int
	editing  bool
	message  string
	msgTimer int

	// base is the card as it was loaded or last saved, or nil for a new card.
	// It supplies anything the fields do not show.
	base *engine.WrestlerCard
}

func NewCardEditorScreen(card *engine.WrestlerCard) *CardEditorScreen {
	e := &CardEditorScreen{base: card}
	if card == nil {
		card = defaultCard()
	}
	e.fields = cardFields(card)
	return e
}

// defaultCard is the starting point for "Create New Card".
func defaultCard() *engine.WrestlerCard {
	card := &engine.WrestlerCard{
		Name:  "New Wrestler",
		Ropes: engine.RatingB, Turnbuckle: engine.RatingB, Ring: engine.RatingB, Deathjump: engine.RatingB,
		PIN: 5, PINAdv: 3, Cage: 5, DQ: 4, Distractor: 5,
		Finisher: engine.Finisher{Name: "FINISHING MOVE", Rating: 3},
	}
	for lvl := range card.Offense {
		for slot := range card.Offense[lvl] {
			card.Offense[lvl][slot] = engine.Move{Name: fmt.Sprintf("Move %d", slot+1), Power: 1, DefLevel: 1}
			card.Defense[lvl][slot] = engine.DefenseOutcome{Type: engine.DefDazed, Power: 1}
		}
	}
	return card
}

func cardFields(card *engine.WrestlerCard) []EditorField {
	number := strconv.Itoa
	fields := []EditorField{
		{Label: "Name", Value: card.Name, Type: FieldString},
		{Label: "--- RATINGS ---", Type: FieldHeading},
		{Label: "Ropes", Value: card.Ropes.String(), Type: FieldRating},
		{Label: "Turnbuckle", Value: card.Turnbuckle.String(), Type: FieldRating},
		{Label: "Ring", Value: card.Ring.String(), Type: FieldRating},
		{Label: "Deathjump", Value: card.Deathjump.String(), Type: FieldRating},
		{Label: "PIN", Value: number(card.PIN), Type: FieldInt},
		{Label: "PIN Adv", Value: number(card.PINAdv), Type: FieldInt},
		{Label: "Cage", Value: number(card.Cage), Type: FieldInt},
		{Label: "DQ", Value: number(card.DQ), Type: FieldInt},
		{Label: "Agility", Value: number(card.Agility), Type: FieldInt},
		{Label: "Power", Value: number(card.Power), Type: FieldInt},
		{Label: "Distractor", Value: number(card.Distractor), Type: FieldInt},
		{Label: "--- FINISHER ---", Type: FieldHeading},
		{Label: "Finisher Name", Value: card.Finisher.Name, Type: FieldString},
		{Label: "Finisher Rating", Value: number(card.Finisher.Rating), Type: FieldInt},
		{Label: finisherRollLabel, Value: formatFinisherRoll(card.Finisher), Type: FieldString},
	}
	for lvl := range card.Offense {
		fields = append(fields, EditorField{Label: fmt.Sprintf("--- OFFENSE LEVEL %d ---", lvl+1), Type: FieldHeading})
		for slot, move := range card.Offense[lvl] {
			fields = append(fields, EditorField{Label: moveLabel(lvl+1, slot+1), Value: formatMoveLine(move), Type: FieldMove})
		}
	}
	for lvl := range card.Defense {
		fields = append(fields, EditorField{Label: fmt.Sprintf("--- DEFENSE LEVEL %d ---", lvl+1), Type: FieldHeading})
		for slot, outcome := range card.Defense[lvl] {
			fields = append(fields, EditorField{Label: defenseLabel(lvl+1, slot+1), Value: formatDefenseLine(outcome), Type: FieldDefense})
		}
	}
	return fields
}

// ─── Input ──────────────────────────────────────────────────────────────────

func (e *CardEditorScreen) Update(g *Game) error {
	if e.msgTimer > 0 {
		e.msgTimer--
		if e.msgTimer == 0 {
			e.message = ""
		}
	}

	if e.editing {
		e.updateEditing(g)
		return nil
	}
	if g.in.JustPressed(ebiten.KeyEscape) {
		g.SetScreen(NewMenuScreen())
		return nil
	}

	e.moveCursor(g.in)
	if confirmPressed(g.in) {
		e.editing = true
	}
	if g.in.Pressed(ebiten.KeyControl) && g.in.JustPressed(ebiten.KeyS) {
		e.saveCard(g)
	}
	return nil
}

// moveCursor moves to the next or previous field, passing over headings.
func (e *CardEditorScreen) moveCursor(in Input) {
	step := 0
	if in.JustPressed(ebiten.KeyDown) {
		step = 1
	}
	if in.JustPressed(ebiten.KeyUp) {
		step = -1
	}
	if step == 0 {
		return
	}
	for moved := 0; moved < len(e.fields); moved++ {
		e.cursor = (e.cursor + step + len(e.fields)) % len(e.fields)
		if e.fields[e.cursor].Type != FieldHeading {
			return
		}
	}
}

func (e *CardEditorScreen) updateEditing(g *Game) {
	field := &e.fields[e.cursor]
	if field.Type == FieldRating {
		e.updateRating(g.in, field)
		return
	}

	for _, c := range g.in.Chars() {
		field.Value += string(c)
	}
	if g.in.JustPressed(ebiten.KeyBackspace) && len(field.Value) > 0 {
		field.Value = field.Value[:len(field.Value)-1]
	}
	if g.in.JustPressed(ebiten.KeyEnter) || g.in.JustPressed(ebiten.KeyEscape) {
		e.editing = false
	}
}

func (e *CardEditorScreen) updateRating(in Input, field *EditorField) {
	ratingKeys := map[ebiten.Key]string{ebiten.KeyA: "A", ebiten.KeyB: "B", ebiten.KeyC: "C"}
	for key, letter := range ratingKeys {
		if in.JustPressed(key) {
			field.Value = letter
			e.editing = false
		}
	}
	if in.JustPressed(ebiten.KeyEscape) {
		e.editing = false
	}
}

// ─── Fields to card ─────────────────────────────────────────────────────────

func (e *CardEditorScreen) fieldValue(label string) string {
	for _, f := range e.fields {
		if f.Label == label {
			return f.Value
		}
	}
	return ""
}

func (e *CardEditorScreen) setFieldValue(label, value string) {
	for i := range e.fields {
		if e.fields[i].Label == label {
			e.fields[i].Value = value
		}
	}
}

func (e *CardEditorScreen) fieldInt(label string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(e.fieldValue(label))) // not a number reads as 0, which the range checks reject
	return n
}

func (e *CardEditorScreen) setMessage(text string) {
	e.message = text
	e.msgTimer = messageTicks
}

type intField struct {
	label    string
	min, max int
	target   func(card *engine.WrestlerCard) *int
}

var intFields = []intField{
	{"PIN", 1, 12, func(c *engine.WrestlerCard) *int { return &c.PIN }},
	{"PIN Adv", 1, 12, func(c *engine.WrestlerCard) *int { return &c.PINAdv }},
	{"Cage", 1, 12, func(c *engine.WrestlerCard) *int { return &c.Cage }},
	{"DQ", 1, 12, func(c *engine.WrestlerCard) *int { return &c.DQ }},
	{"Agility", -5, 5, func(c *engine.WrestlerCard) *int { return &c.Agility }},
	{"Power", -5, 5, func(c *engine.WrestlerCard) *int { return &c.Power }},
	{"Distractor", 1, 12, func(c *engine.WrestlerCard) *int { return &c.Distractor }},
	{"Finisher Rating", 0, 8, func(c *engine.WrestlerCard) *int { return &c.Finisher.Rating }},
}

type ratingField struct {
	label  string
	target func(card *engine.WrestlerCard) *engine.Rating
}

var ratingFields = []ratingField{
	{"Ropes", func(c *engine.WrestlerCard) *engine.Rating { return &c.Ropes }},
	{"Turnbuckle", func(c *engine.WrestlerCard) *engine.Rating { return &c.Turnbuckle }},
	{"Ring", func(c *engine.WrestlerCard) *engine.Rating { return &c.Ring }},
	{"Deathjump", func(c *engine.WrestlerCard) *engine.Rating { return &c.Deathjump }},
}

var ratingsByLetter = map[string]engine.Rating{"A": engine.RatingA, "B": engine.RatingB, "C": engine.RatingC}

// buildCard turns the fields into a card. The error is the first field that
// does not hold a valid value.
func (e *CardEditorScreen) buildCard() (*engine.WrestlerCard, error) {
	card := &engine.WrestlerCard{Name: strings.TrimSpace(e.fieldValue("Name"))}
	if card.Name == "" {
		return nil, errors.New("Name cannot be empty")
	}
	card.Finisher.Name = strings.TrimSpace(e.fieldValue("Finisher Name"))

	steps := []func(*engine.WrestlerCard) error{
		e.readRatings, e.readNumbers, e.readFinisher, e.readOffense, e.readDefense,
	}
	for _, step := range steps {
		if err := step(card); err != nil {
			return nil, err
		}
	}
	return card, nil
}

func (e *CardEditorScreen) readRatings(card *engine.WrestlerCard) error {
	for _, f := range ratingFields {
		letter := strings.ToUpper(strings.TrimSpace(e.fieldValue(f.label)))
		rating, ok := ratingsByLetter[letter]
		if !ok {
			return fmt.Errorf("%s must be A, B, or C (got %q)", f.label, letter)
		}
		*f.target(card) = rating
	}
	return nil
}

func (e *CardEditorScreen) readNumbers(card *engine.WrestlerCard) error {
	for _, f := range intFields {
		v := e.fieldInt(f.label)
		if v < f.min || v > f.max {
			return fmt.Errorf("%s must be %d to %d (got %d)", f.label, f.min, f.max, v)
		}
		*f.target(card) = v
	}
	return nil
}

func (e *CardEditorScreen) readFinisher(card *engine.WrestlerCard) error {
	if card.Finisher.Name == "" {
		return errors.New("Finisher Name cannot be empty")
	}
	if err := applyFinisherRoll(&card.Finisher, e.fieldValue(finisherRollLabel)); err != nil {
		return fmt.Errorf("Finisher Roll: %w", err)
	}
	return nil
}

func (e *CardEditorScreen) readOffense(card *engine.WrestlerCard) error {
	for lvl := range card.Offense {
		for slot := range card.Offense[lvl] {
			move, err := parseMoveLine(e.fieldValue(moveLabel(lvl+1, slot+1)))
			if err != nil {
				return fmt.Errorf("L%d Move %d: %w", lvl+1, slot+1, err)
			}
			card.Offense[lvl][slot] = move
		}
	}
	return nil
}

func (e *CardEditorScreen) readDefense(card *engine.WrestlerCard) error {
	for lvl := range card.Defense {
		for slot := range card.Defense[lvl] {
			outcome, err := parseDefenseLine(e.fieldValue(defenseLabel(lvl+1, slot+1)))
			if err != nil {
				return fmt.Errorf("L%d Def %d: %w", lvl+1, slot+1, err)
			}
			if e.base != nil && outcome.Type == engine.DefPIN {
				outcome.PINThreshold = e.base.Defense[lvl][slot].PINThreshold
			}
			card.Defense[lvl][slot] = outcome
		}
	}
	return nil
}

func cardFileName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_")) + ".yaml"
}

func (e *CardEditorScreen) saveCard(g *Game) {
	card, err := e.buildCard()
	if err != nil {
		e.setMessage("Validation: " + err.Error())
		return
	}
	data, err := loader.MarshalCard(card)
	if err != nil {
		e.setMessage("Error: " + err.Error())
		return
	}
	filename := cardFileName(card.Name)
	if err := g.Store.SaveCardBytes(filename, data); err != nil {
		e.setMessage("Error saving: " + err.Error())
		return
	}

	if e.base != nil && e.base.Name != card.Name {
		e.setMessage(fmt.Sprintf("Saved as new card %s (original %s kept)", filename, e.base.Name))
	} else {
		e.setMessage("Saved " + filename)
	}
	e.base = card
	reloadRoster(g)
}

// ─── Drawing ────────────────────────────────────────────────────────────────

const editorListTop = Margin + LineHeight*2

func editorMessageY(screenH int) int {
	return screenH - LineHeight*2 - Margin
}

func editorVisibleLines(screenH int) int {
	return (editorMessageY(screenH) - editorListTop) / LineHeight
}

func (e *CardEditorScreen) Draw(screen *ebiten.Image, g *Game) {
	screen.Fill(Background)

	visibleLines := editorVisibleLines(g.screenH)
	if visibleLines < 5 {
		visibleLines = 5
	}
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
		DrawText(screen, e.fieldLine(i), Margin, y)
		y += LineHeight
	}

	if e.message != "" {
		DrawText(screen, e.message, Margin, editorMessageY(g.screenH))
	}
	DrawText(screen, fmt.Sprintf("Field %d/%d", e.cursor+1, len(e.fields)), Margin, g.screenH-LineHeight-Margin)
}

func (e *CardEditorScreen) fieldLine(i int) string {
	f := e.fields[i]
	if f.Type == FieldHeading {
		return "  " + f.Label
	}
	prefix := "  "
	if i == e.cursor {
		prefix = "> "
	}
	suffix := ""
	if i == e.cursor && e.editing {
		suffix = "_"
	}
	return fmt.Sprintf("%s%-40s %s%s", prefix, f.Label+":", f.Value, suffix)
}
