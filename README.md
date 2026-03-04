# Ring Wars

A DOS-style wrestling match simulator built with Go and Ebitengine. Input wrestler cards and watch simulated matches play out with dramatic text commentary.

## Requirements

- Go 1.21 or later
- Linux, macOS, or Windows

## Installation

```
git clone <repo-url>
cd wrestling
go mod tidy
```

## Running

```
go run .
```

The game window opens at 1280x720 and can be resized freely.

## Main Menu

From the main menu, select one of the following options:

- **Singles Match** - Standard 1v1 match
- **Tag Team Match** - 2v2 tag team match
- **Cage Match** - Steel cage match (no DQ, no count-out, uses Cage rating)
- **No DQ Match** - No disqualification match
- **Create New Card** - Open the card editor with default values
- **Edit Existing Card** - Select a wrestler card to edit

After selecting a match type, you'll choose wrestlers from your roster. For tag team matches, you select 4 wrestlers (2 per team).

## Controls

### Menu Navigation

| Key | Action |
|-----|--------|
| UP / DOWN | Navigate menu options |
| ENTER / SPACE | Confirm selection |
| ESC | Go back / Quit |

### During a Match

| Key | Action |
|-----|--------|
| SPACE / ENTER | Advance to next event (step-through mode) |
| A | Toggle auto-play on/off |
| + | Speed up auto-play |
| - | Slow down auto-play |
| UP / DOWN | Scroll through match text |
| R | Rematch (after match ends) |
| ESC | Return to main menu |

### Card Editor

| Key | Action |
|-----|--------|
| UP / DOWN | Navigate fields |
| ENTER / SPACE | Edit selected field |
| A / B / C | Set rating fields (when editing a rating) |
| BACKSPACE | Delete character (when editing text) |
| ENTER / ESC | Finish editing a field |
| Ctrl+S | Save card to file |
| ESC | Return to main menu (when not editing a field) |

## How Matches Work

The simulator follows card-based wrestling rules:

### Basic Flow

1. Both wrestlers roll 1d6 for initiative. Higher roll goes on offense first.
2. The offensive wrestler rolls 2d6 to determine which move they execute from their offense grid.
3. The move's defense level determines which row of the defender's defense grid is used.
4. The defender rolls 1d6 to determine the defense outcome.
5. Defense outcomes include: **Dazed** (attacker stays on offense), **Hurt** (attacker stays, damage accumulates), **Down** (attacker stays, big damage), **Reversal** (defender takes over offense), or **PIN** (pin attempt triggered).

### Offense Grid

Each wrestler has 3 offense levels with 6 moves each (rolled on 2d6 mapped to slots 1-6). The offense level increases as the match progresses:
- **Level 1** (rolls 2-7): Basic moves
- **Level 2** (rolls 8-9): Intermediate moves
- **Level 3** (rolls 10-12): Advanced/signature moves

Each move has a name, a power value, and specifies which defense level the opponent must roll on.

### Defense Grid

Each wrestler has 3 defense levels with 6 outcomes each. The defense level is determined by the attacker's move. Outcomes determine what happens next.

### Special Move Tags

Moves can have special properties:

- **AG (Agility)** - Requires an agility check. The attacker rolls 1d6 and adds their Agility rating. On 5+, the move hits. On failure, the opponent takes offense.
- **PW (Power)** - Requires a power check. The attacker rolls 1d6 and adds their Power rating. On 5+, the move hits. On failure, the opponent takes offense.
- **DQ** - Illegal move. Roll against the wrestler's DQ rating to see if the referee catches it. If caught, the wrestler is disqualified.
- **Add1** - Adds 1 to the opponent's fatigue (PIN value increases by 1).
- **DIS (Distraction)** - Uses the distraction mechanic.
- **CH (Chart)** - Triggers a chart lookup (Ropes, Turnbuckle, Out of Ring, or Deathjump).

### PIN Attempts

When a PIN is triggered:
1. The attacker rolls 2d6.
2. If the roll is greater than or equal to the defender's current PIN value, the pin succeeds and the match is over.
3. If the roll is less, the defender kicks out.
4. After each failed pin, the defender's PIN value increases by 1 (fatigue), making future pins easier.
5. The defender's PIN Adv value is used when the attacker has advantage (e.g., after a finisher).

### Finishers

Each wrestler has a finishing move with a rating. When a finisher is triggered:
1. The attacker rolls 2d6 and adds the finisher rating.
2. If the total is 12 or higher, the finisher connects and triggers a PIN attempt using the opponent's advantaged PIN rating.
3. If the total is less than 12, the finisher misses and the opponent takes over on offense.

Some finishers are "roll finishers" — the finisher rating replaces the roll entirely (always hits if rating is 12+).

### Charts

When a move sends a wrestler to a chart, the outcome is determined by the wrestler's rating (A, B, or C) for that chart and a 2d6 roll:

- **Into the Ropes** - Outcomes range from continuing offense to agility/power checks, DQ situations, and PIN attempts.
- **Into the Turnbuckle** - Similar to ropes but with different outcome distributions.
- **Out of the Ring** - Includes count-out risk. Both wrestlers may end up brawling outside.
- **Deathjump** - High-risk moves. Can result in big damage, referee knockdowns, or the attacker hurting themselves.

### Special Match Types

**Cage Match**
- Uses each wrestler's Cage rating instead of standard PIN
- No disqualifications
- No count-outs
- "Face into cage" moves deal extra damage

**No DQ Match**
- All DQ-tagged moves are legal
- No disqualifications can occur
- Count-outs still apply

**Tag Team Match**
- Each team has 2 wrestlers
- The active wrestler can tag their partner in during their offense turn
- On defense, wrestlers can attempt to tag out (roll 1d6, succeed on 4 or less)
- Each team gets up to 2 pin save attempts per match
- Pin saves can result in the save succeeding, failing, a double DQ, the saving wrestler getting hurt, or the pinned wrestler's partner being thrown out

### Referee Down

Certain chart outcomes can knock the referee down. While the referee is down:
- PIN attempts automatically fail (no ref to count)
- DQ calls cannot be made
- The referee recovers after 2 turns

## Wrestler Cards

Wrestler data is stored as YAML files in the `data/wrestlers/` directory. The game loads all `.yaml` files from this directory at startup.

### Card Format

```yaml
name: Wrestler Name
ropes: B          # Rating: A, B, or C
turnbuckle: A
ring: B
deathjump: C
pin: 9            # Base PIN number (2d6 must meet or exceed)
pin_adv: 6        # Advantaged PIN (used after finisher)
cage: 7           # Cage match PIN rating
dq: 4             # DQ threshold (1d6, caught if roll >= this)
agility: 2        # Agility modifier (-3 to +5)
power: 0          # Power modifier (-3 to +5)
distractor: 5     # Distraction rating

finisher:
  name: FINISHING MOVE NAME    # ALL CAPS name
  rating: 4                     # Added to 2d6 roll (12+ = hit)

offense:
  - - {name: Arm Drag, power: 1, def_level: 1}
    - {name: Dropkick, power: 2, def_level: 1}
    - {name: Body Slam, power: 1, def_level: 2}
    - {name: Suplex, power: 2, def_level: 2}
    - {name: Clothesline, power: 2, def_level: 1}
    - {name: Hip Toss, power: 1, def_level: 1}
  - - {name: DDT, power: 3, def_level: 2}
    - {name: Piledriver, power: 3, def_level: 2}
    - {name: Ropes, power: 0, def_level: 1, tags: [ch], chart: ropes}
    - {name: Flying Elbow, power: 2, def_level: 2, tags: [ag]}
    - {name: Turnbuckle Smash, power: 2, def_level: 2, tags: [ch], chart: turnbuckle}
    - {name: Backbreaker, power: 3, def_level: 3}
  - - {name: FINISHING MOVE NAME, power: 4, def_level: 3}
    - {name: Super Slam, power: 3, def_level: 3, tags: [pw]}
    - {name: Low Blow, power: 2, def_level: 2, tags: [dq]}
    - {name: Deathjump, power: 0, def_level: 1, tags: [ch], chart: deathjump}
    - {name: Top Rope Move, power: 3, def_level: 3, tags: [ag]}
    - {name: Power Bomb, power: 4, def_level: 3, tags: [pw]}

defense:
  - - {type: dazed, power: 1}
    - {type: dazed, power: 1}
    - {type: hurt, power: 2}
    - {type: reversal, power: 0}
    - {type: reversal, power: 0}
    - {type: down, power: 3}
  - - {type: dazed, power: 1}
    - {type: hurt, power: 2}
    - {type: hurt, power: 2}
    - {type: down, power: 3}
    - {type: reversal, power: 0}
    - {type: pin, power: 0}
  - - {type: hurt, power: 2}
    - {type: hurt, power: 2}
    - {type: down, power: 3}
    - {type: down, power: 3}
    - {type: pin, power: 0}
    - {type: pin, power: 0}
```

### Move Tags

| Tag | Meaning |
|-----|---------|
| `ag` | Agility check required |
| `pw` | Power check required |
| `dq` | Illegal move (DQ check) |
| `add1` | Adds 1 fatigue to opponent |
| `dis` | Distraction move |
| `ch` | Chart move (requires `chart` field) |

### Chart Types

| Value | Chart |
|-------|-------|
| `ropes` | Into the Ropes |
| `turnbuckle` | Into the Turnbuckle |
| `outofring` | Out of the Ring |
| `deathjump` | Deathjump |

### Defense Types

| Type | Effect |
|------|--------|
| `dazed` | Attacker stays on offense |
| `hurt` | Attacker stays, extra damage |
| `down` | Attacker stays, major damage |
| `reversal` | Defender takes over offense |
| `pin` | PIN attempt triggered |

## Creating Cards

You can create wrestler cards two ways:

1. **In-Game Editor** - Select "Create New Card" from the main menu. Fill in all fields and press Ctrl+S to save. The card is saved to `data/wrestlers/` as a YAML file named after the wrestler.

2. **Manual YAML** - Create a `.yaml` file in `data/wrestlers/` following the card format above. The game loads all cards from this directory on startup.

### Tips for Card Design

- **PIN 8-10** is typical. Lower = harder to pin. Higher = easier.
- **PIN Adv** should be 2-4 lower than PIN.
- **Agility** ranges from -3 (slow) to +5 (very agile). High-flyers should have 3-5.
- **Power** ranges from -3 (weak) to +5 (very strong). Powerhouses should have 3-5.
- **DQ 3-5** is typical. Lower = more reckless. Higher = more cautious.
- **Finisher Rating 2-5** is typical. Higher = more devastating finisher.
- More reversals in defense = harder to keep on offense against.
- More PINs in defense = more pin attempts triggered.

## Included Wrestlers

The game comes with 8 example wrestler cards:

- **Butcher Briggs** - Brawling powerhouse (Power 4, Agility -1)
- **Rico Stormcloud** - High-flying technician (Agility 4, Power 0)
- **Rex Fontaine** - Dirty technical wrestler (DQ 5, well-rounded)
- **Armand the Colossus** - Unstoppable giant (Power 5, PIN 7, Agility -3)
- **Buck Stallion** - All-American powerhouse (Power 5, Agility -2)
- **Ricky Rampage** - High-flying brawler (Agility 2, Power 1)
- **The Gravedigger** - Supernatural powerhouse (Power 4, PIN 6)
- **Blake Harton** - Technical excellence (Agility 2, DQ 5)
