package engine

import (
	"strings"
	"testing"
)

type scriptedDice struct {
	rolls []int
	next  int
}

func (d *scriptedDice) Roll() int {
	if d.next >= len(d.rolls) {
		panic("scripted dice exhausted")
	}
	r := d.rolls[d.next]
	d.next++
	return r
}

func (d *scriptedDice) remaining() int {
	return len(d.rolls) - d.next
}

func script(rolls ...int) *scriptedDice {
	return &scriptedDice{rolls: rolls}
}

func plainMove(name string, power, defLevel int) Move {
	return Move{Name: name, Power: power, DefLevel: defLevel}
}

func allMoves(m Move) [3][6]Move {
	var grid [3][6]Move
	for lvl := range grid {
		for slot := range grid[lvl] {
			grid[lvl][slot] = m
		}
	}
	return grid
}

func allDefense(d DefenseOutcome) [3][6]DefenseOutcome {
	var grid [3][6]DefenseOutcome
	for lvl := range grid {
		for slot := range grid[lvl] {
			grid[lvl][slot] = d
		}
	}
	return grid
}

func testCard(name string) *WrestlerCard {
	return &WrestlerCard{
		Name:       name,
		Offense:    allMoves(plainMove("Jab", 1, 1)),
		Defense:    allDefense(DefenseOutcome{Type: DefDazed, Power: 1}),
		Ropes:      RatingB,
		Turnbuckle: RatingB,
		Ring:       RatingB,
		Deathjump:  RatingB,
		PIN:        5,
		PINAdv:     3,
		Cage:       4,
		DQ:         4,
		Distractor: 5,
		Finisher:   Finisher{Name: "TEST FINISH", Rating: 2},
	}
}

func singlesMatch(a, b *WrestlerCard, d Dice) *Match {
	m := NewMatch(a, b)
	m.SetDice(d)
	return m
}

func eventTexts(m *Match) string {
	lines := make([]string, 0, len(m.Events))
	for _, e := range m.Events {
		lines = append(lines, e.Text)
	}
	return strings.Join(lines, "\n")
}

func wantOffense(t *testing.T, m *Match, side, level int) {
	t.Helper()
	if m.onOffense != side || m.offLevel != level-1 {
		t.Fatalf("offense: got side %d level %d, want side %d level %d\n%s",
			m.onOffense, m.offLevel+1, side, level, eventTexts(m))
	}
}

func wantLog(t *testing.T, m *Match, fragment string) {
	t.Helper()
	if !strings.Contains(eventTexts(m), fragment) {
		t.Fatalf("log does not contain %q\n%s", fragment, eventTexts(m))
	}
}

func wantNoLog(t *testing.T, m *Match, fragment string) {
	t.Helper()
	if strings.Contains(eventTexts(m), fragment) {
		t.Fatalf("log unexpectedly contains %q\n%s", fragment, eventTexts(m))
	}
}
