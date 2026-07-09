package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type TextInput struct {
	Text      string
	MaxLength int
	blink     int
}

func NewTextInput(maxLen int) *TextInput {
	return &TextInput{MaxLength: maxLen}
}

func (t *TextInput) Update() {
	t.blink++

	// Append typed characters
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if ch >= 32 && ch < 127 && len(t.Text) < t.MaxLength {
			t.Text += string(ch)
		}
	}

	// Backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(t.Text) > 0 {
		t.Text = t.Text[:len(t.Text)-1]
	}
}

func (t *TextInput) DisplayText() string {
	cursor := "_"
	if (t.blink/30)%2 == 0 {
		cursor = " "
	}
	return t.Text + cursor
}

func (t *TextInput) Reset() {
	t.Text = ""
	t.blink = 0
}
