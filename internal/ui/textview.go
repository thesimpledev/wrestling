package ui

import "github.com/hajimehoshi/ebiten/v2"

// fittingLines returns the lines from start that fit between the top margin
// and the status bar, so that no line is drawn over the status bar.
func fittingLines(lines []string, start, screenH int) []string {
	if len(lines) == 0 {
		return nil
	}
	start = max(min(start, len(lines)-1), 0)
	rows := max((screenH-LineHeight-2*Margin)/LineHeight, 0)
	return lines[start:min(start+rows, len(lines))]
}

// drawScrolledLines draws the lines from start down the screen, stopping
// above the status bar.
func drawScrolledLines(screen *ebiten.Image, lines []string, start, screenH int) {
	for i, line := range fittingLines(lines, start, screenH) {
		DrawText(screen, line, Margin, Margin+i*LineHeight)
	}
}
