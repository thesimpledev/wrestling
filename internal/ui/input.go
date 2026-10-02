package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input reports the keyboard state for the current game tick.
type Input interface {
	// JustPressed reports whether the key went down on this tick.
	JustPressed(key ebiten.Key) bool
	// Pressed reports whether the key is held down.
	Pressed(key ebiten.Key) bool
	// Chars returns the characters typed on this tick.
	Chars() []rune
	// AnyJustPressed reports whether any key went down on this tick.
	AnyJustPressed() bool
}

type ebitenInput struct{}

func (ebitenInput) JustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

func (ebitenInput) Pressed(key ebiten.Key) bool {
	return ebiten.IsKeyPressed(key)
}

func (ebitenInput) Chars() []rune {
	return ebiten.AppendInputChars(nil)
}

func (ebitenInput) AnyJustPressed() bool {
	return len(inpututil.AppendJustPressedKeys(nil)) > 0
}
