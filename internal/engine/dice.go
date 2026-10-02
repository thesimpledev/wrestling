package engine

import "math/rand/v2"

// Dice is the source of six-sided die rolls for a match.
type Dice interface {
	// Roll returns a value from 1 to 6.
	Roll() int
}

type randomDice struct{}

func (randomDice) Roll() int {
	return rand.IntN(6) + 1 // #nosec G404 -- game dice, not security-sensitive
}

// twoDice is the result of rolling two dice together.
type twoDice struct {
	first  int
	second int
}

func (r twoDice) total() int {
	return r.first + r.second
}

func (r twoDice) doubles() bool {
	return r.first == r.second
}

// SetDice replaces the dice a match rolls, which lets tests script every roll.
func (m *Match) SetDice(d Dice) {
	if d == nil {
		panic("engine: SetDice called with nil dice")
	}
	m.dice = d
}

func (m *Match) rollOne() int {
	return m.dice.Roll()
}

func (m *Match) rollTwo() twoDice {
	return twoDice{first: m.dice.Roll(), second: m.dice.Roll()}
}
