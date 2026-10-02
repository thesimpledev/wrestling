package ui

import "github.com/hajimehoshi/ebiten/v2"

const (
	listMoreAbove = "    ^ more"
	listMoreBelow = "    v more"

	// Below this many rows there is no room to spend two on scroll hints.
	minRowsForScrollHints = 5
)

// listWindow returns the range of items to show so that the cursor stays in
// view when a list has more items than rows.
func listWindow(cursor, total, rows int) (start, end int) {
	if rows < 1 {
		rows = 1
	}
	if total <= rows {
		return 0, total
	}
	start = cursor - rows/2
	if start < 0 {
		start = 0
	}
	if start > total-rows {
		start = total - rows
	}
	return start, start + rows
}

// windowedLines returns the lines to draw for a list that may be longer than
// the rows available: the item under the cursor is marked, and the first or
// last row becomes a hint when there is more to scroll to.
func windowedLines(items []string, cursor, rows int) []string {
	start, end := listWindow(cursor, len(items), rows)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		lines = append(lines, prefix+items[i])
	}
	if rows < minRowsForScrollHints || len(lines) == 0 {
		return lines
	}
	if start > 0 {
		lines[0] = listMoreAbove
	}
	if end < len(items) {
		lines[len(lines)-1] = listMoreBelow
	}
	return lines
}

// drawList draws a scrolling list from y down to the status bar.
func drawList(screen *ebiten.Image, g *Game, y int, items []string, cursor int) {
	rows := (g.screenH - LineHeight - Margin - y) / LineHeight
	for _, line := range windowedLines(items, cursor, rows) {
		DrawText(screen, line, Margin, y)
		y += LineHeight
	}
}
