package ui

import "github.com/hajimehoshi/ebiten/v2"

// adjustSpeed applies the playback speed keys. The speed is the number of
// ticks between auto-play steps, so + lowers it (never below 5) and - raises it.
func adjustSpeed(in Input, speed int) int {
	if (in.JustPressed(ebiten.KeyEqual) || in.JustPressed(ebiten.KeyNumpadAdd)) && speed > 5 {
		speed -= 5
	}
	if in.JustPressed(ebiten.KeyMinus) || in.JustPressed(ebiten.KeyNumpadSubtract) {
		speed += 5
	}
	return speed
}

// scrollWithKeys moves the scroll position one line while up or down is
// held, keeping it between 0 and maxScroll.
func scrollWithKeys(in Input, scroll, maxScroll int) int {
	if in.Pressed(ebiten.KeyUp) && scroll > 0 {
		scroll--
	}
	if in.Pressed(ebiten.KeyDown) && scroll < maxScroll {
		scroll++
	}
	return scroll
}

// stepDue reports whether the next match event should be shown, either
// because a step key was pressed or because the auto-play timer ran out.
// It returns the updated timer.
func stepDue(in Input, autoPlay bool, ticker, speed int) (bool, int) {
	due := in.JustPressed(ebiten.KeySpace) || in.JustPressed(ebiten.KeyEnter)
	if autoPlay {
		ticker++
		if ticker >= speed {
			ticker = 0
			due = true
		}
	}
	return due, ticker
}
