package engine

import "testing"

func TestInitiativeUsesInjectedDice(t *testing.T) {
	cases := []struct {
		name     string
		rolls    []int
		wantSide int
	}{
		{"first wrestler higher", []int{6, 1}, 0},
		{"second wrestler higher", []int{2, 5}, 1},
		{"tie goes to first wrestler", []int{3, 3}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dice := script(tc.rolls...)
			m := singlesMatch(testCard("A"), testCard("B"), dice)
			m.rollForInitiative()
			wantOffense(t, m, tc.wantSide, 1)
			if dice.remaining() != 0 {
				t.Fatalf("dice left unrolled: %d", dice.remaining())
			}
		})
	}
}

func TestRandomDiceStayInRange(t *testing.T) {
	d := randomDice{}
	for i := 0; i < 1000; i++ {
		if r := d.Roll(); r < 1 || r > 6 {
			t.Fatalf("roll %d out of range", r)
		}
	}
}

func TestRollTwoReturnsBothDice(t *testing.T) {
	m := singlesMatch(testCard("A"), testCard("B"), script(4, 4, 2, 5))
	if r := m.rollTwo(); r.total() != 8 || !r.doubles() {
		t.Fatalf("got %+v, want total 8 doubles", r)
	}
	if r := m.rollTwo(); r.total() != 7 || r.doubles() {
		t.Fatalf("got %+v, want total 7 not doubles", r)
	}
}
