package engine

import "math/rand/v2"

// Roll1d6 returns a random number from 1-6.
func Roll1d6() int {
	return rand.IntN(6) + 1
}

// Roll2d6 returns the sum of two six-sided dice (2-12).
func Roll2d6() int {
	return Roll1d6() + Roll1d6()
}

// RollIsDoubles rolls 2d6 and returns the total and whether doubles were rolled.
func RollIsDoubles() (total int, doubles bool) {
	a := Roll1d6()
	b := Roll1d6()
	return a + b, a == b
}
