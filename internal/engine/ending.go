package engine

const methodDoubleDQ = "double dq"

// Draw reports whether the match ended with no winner.
func (r *MatchResult) Draw() bool {
	return r.Winner == ""
}

// endMatch records the result. endingRoll is the roll that decided the match;
// in a feud match doubles on that roll send the wrestlers to the Feud Table.
func (m *Match) endMatch(winner, loser *WrestlerState, method string, endingRoll twoDice) {
	m.over = true
	m.result = &MatchResult{
		WinningSide: m.sideOf(winner),
		Winner:      winner.Card.Name,
		Loser:       loser.Card.Name,
		Method:      method,
	}
	m.emit(matchEndEvent(winner.Card.Name, loser.Card.Name, method))

	if method == "dq" {
		m.feudAfterMatch(endingRoll, loser, winner)
		return
	}
	m.feudAfterMatch(endingRoll, winner, loser)
}

// endDoubleDQ ends the match with both sides disqualified and no winner.
func (m *Match) endDoubleDQ(endingRoll twoDice) {
	m.over = true
	m.result = &MatchResult{WinningSide: -1, Method: methodDoubleDQ}
	m.emit(newEvent(EventMatchEnd, "DOUBLE DISQUALIFICATION! Both sides are thrown out and there is no winner!"))
	m.feudAfterMatch(endingRoll, m.attacker(), m.defender())
}

func (m *Match) sideOf(ws *WrestlerState) int {
	for i, side := range m.Sides {
		for _, w := range side.Wrestlers {
			if w == ws {
				return i
			}
		}
	}
	return 0
}

func (m *Match) otherWrestler(ws *WrestlerState) *WrestlerState {
	return m.Sides[1-m.sideOf(ws)].Active()
}
