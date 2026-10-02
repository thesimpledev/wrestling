# Ring Wars

A DOS-style wrestling match simulator built with Go and Ebitengine. Input wrestler cards and watch simulated matches play out with dramatic text commentary.

Compatible with Filsinger Games card systems including Champions of the Galaxy and Legends of Wrestling. Ring Wars is not affiliated with or endorsed by Filsinger Games.

## Requirements

- Go 1.25 or later
- Linux, macOS, or Windows

## Project Layout

- `main.go`, `main_wasm.go`, `embed.go`: desktop and WebAssembly entry points and the embedded default cards
- `cmd/serve/`: small local web server for trying the web build
- `internal/engine/`: match, chart and career rules
- `internal/loader/`: card, injury and career loading
- `internal/storage/`: desktop and browser storage backends
- `internal/ui/`: Ebitengine screens
- `data/`: wrestler cards and injury tables
- `dist/web/`: page that hosts the WebAssembly build
- `vendor/`: vendored dependencies

## Installation

```
git clone <repo-url>
cd wrestling
go mod tidy
```

## Running

### Desktop

```
go run .
```

The game window opens at 1280x720 and can be resized freely.

### Web

The same code builds to WebAssembly and runs in a browser:

```
just serve
```

This builds `dist/web/wrestling.wasm` and serves the page at http://127.0.0.1:8080/. Without `just`, run `GOOS=js GOARCH=wasm go build -o dist/web/wrestling.wasm .`, copy `wasm_exec.js` from `$(go env GOROOT)/lib/wasm/` into `dist/web/`, then run `go run ./cmd/serve dist/web`.

Every push to `master` deploys the web build to GitHub Pages through the `Deploy to GitHub Pages` workflow.

### Where things are saved

| | Desktop | Web |
|---|---|---|
| Wrestler cards | `data/wrestlers/*.yaml` | browser storage (the 8 bundled cards are built in) |
| Injuries and suspensions | `data/injuries.json` | browser storage |
| Federations | `data/career.json` | browser storage |
| Settings | `data/settings.json` | browser storage |

If a saved card cannot be read, it is skipped and a red notice names the file; the game still starts. The same notice appears for a card that uses an instruction the game does not know, and for federation, injury or settings data that cannot be read.

## Main Menu

- **Federation** - Run a federation week by week (see Federation Mode)
- **Singles Match** - Standard 1v1 match
- **Tag Team Match** - 2v2 tag team match
- **Cage Match** - Steel cage match
- **No DQ Match** - No disqualifications and no count-outs
- **Feud Match** - Singles match that can go to the Feud Table after the bell
- **Battle Royal** - Gauntlet: the winner stays and faces the next entrant
- **Tournament** - Single-elimination bracket of 4, 8 or 16
- **Create New Card** - Open the card editor with default values
- **Edit Existing Card** - Select a wrestler card to edit
- **Settings** - Switch the optional rules on and off

### Match setup

After picking a match type you pick the wrestlers. A wrestler already in the match cannot be picked again, and long rosters scroll. ESC goes back one step.

- **Singles, Cage, No DQ, Feud**: two wrestlers, then an optional ringside ally for each.
- **Tag Team**: four wrestlers (two per team), then for each team whether they are regular tag partners. Regular partners tag out more easily and can save each other from pins.
- **Battle Royal**: tick three or more wrestlers with SPACE, then ENTER.

Injured wrestlers are marked `[INJURED n]` and suspended wrestlers `[SUSPENDED n]`, where n is the number of fight cards left.

## Controls

### Menu Navigation

| Key | Action |
|-----|--------|
| UP / DOWN | Navigate menu options |
| ENTER / SPACE | Confirm selection |
| ESC | Go back one step. On the main menu it quits the desktop version and does nothing in the browser |
| F1 | Switch between large and small text |

### During a Match

| Key | Action |
|-----|--------|
| SPACE / ENTER | Advance to next event (step-through mode) |
| A | Toggle auto-play on/off |
| + | Speed up auto-play |
| - | Slow down auto-play |
| UP / DOWN | Scroll through match text |
| R | Rematch with the same wrestlers, teams and allies (after the match ends) |
| ESC | Return to main menu |

### Card Editor

| Key | Action |
|-----|--------|
| UP / DOWN | Navigate fields |
| ENTER / SPACE | Edit selected field |
| A / B / C | Set rating fields (when editing a rating) |
| BACKSPACE | Delete character (when editing text) |
| ENTER / ESC | Finish editing a field |
| Ctrl+S | Save card |
| ESC | Return to main menu (when not editing a field) |

## How Matches Work

The simulator plays both wrestlers by the Filsinger Games rules, including the advanced rules for fatigue, agility and power, choices, finishers, charts, allies and feuds.

### Basic Flow

1. Both wrestlers roll one die for initiative. The higher roll starts on offense at Level 1 (the first wrestler on a tie).
2. The wrestler on offense rolls one die on his current offense level. The result is a move.
3. A normal move has a number (1, 2 or 3). The opponent rolls one die on that level of his defense.
4. The defense result decides what happens next:
   - **Dazed, Hurt, Down** followed by a number: the wrestler on offense stays there and rolls next on the level of that number.
   - **Reversal** followed by a number: the defender takes over offense on that level.
   - **PIN**: the defender is in a pinning predicament.

So the offense level goes up and down with the number on each defense result. It is not tied to the dice.

### PIN Attempts and Fatigue

The pinned wrestler rolls two dice against his PIN rating. A roll equal to or less than the rating is a pin and the match is over. A higher roll is a kick-out: his PIN rating goes up by 1 (fatigue) and the wrestler on offense keeps going on Level 3.

The simulator uses the advanced fatigue rules, so each wrestler starts on the PIN rating in parentheses on his card (`pin_adv`). In a cage match he starts on his Cage rating. An injured wrestler starts 2 higher.

### Finishers

A finisher is a move in CAPITAL LETTERS on Level 3 offense. When it is rolled, the opponent goes straight to a PIN roll with the finisher rating added to his PIN rating, so a +4 finisher against a PIN rating of 5 pins on 9 or less. A capital-letter move rolled anywhere but Level 3 is an ordinary move.

A roll finisher has a range on one die (for example 2-6) in place of a fixed rating. The wrestler rolls one die: inside the range, the roll is the rating; outside it, the finisher misses and the opponent takes over on Level 3.

### Agility and Power

Agility and Power run from -5 (excellent) to +5 (poor). Lower is better.

- **(ag)** - The move works only if the attacker's Agility is the same as or better than the opponent's. If not, the opponent takes over on Level 2.
- **(pw)** - The same, using Power.

### Other Move Instructions

- **(dis)** - Illegal move. The attacker rolls two dice against his Disqualification rating and is disqualified on that number or less. A card can print its own number with the instruction (`dis 7`).
- **(add1)** - Adds 1 to the opponent's PIN rating.
- **(tag)** on a move - Only used in tag matches. In a singles match it is rolled again.
- **(singles)** - Only used in singles matches. In a tag match it is rolled again.
- **Choice A to H** - The wrestler picks one of two moves from the Choice chart. A roll move works on two dice at or under its number plus the opponent's Agility or Power rating; then the opponent rolls defense on the level of the move's number. If it fails, the opponent takes over on Level 2. The simulator picks the option more likely to work.

### Defense Instructions

- **(lv)** on a Down 3 - The wrestler may leave the ring and roll the Out of the Ring chart on his own rating. The simulator leaves when his Ring rating is A or B.
- **(tag)** on a defense result - In a tag match the wrestler tries to tag out (see Tag Team Match).

### Charts

A chart move sends the opponent to a chart. He rolls two dice on the column for his own rating (A, B or C) on that chart.

- **Into the Ropes** and **Into the Turnbuckle** - Results range from a pin attempt for the wrestler sent there to Level 3 offense for the thrower, with power and rating comparisons in between.
- **Out of the Ring** - Brawls, disqualification rolls and count-outs. A count-out is two dice against the PIN rating of the wrestler outside.
- **Deathjump** - High risk. Can end in a pin attempt for either wrestler, a struggle decided by Agility, or the referee going down.

### Referee Down

When the referee goes down he misses the next moves (two dice are rolled for how many). While he is down pins are not counted, though they still add fatigue, and nobody can be disqualified or counted out.

### Special Match Types

**Cage Match**
- Wrestlers start on their Cage rating in place of their PIN rating
- No disqualifications and no count-outs
- Nobody leaves the ring: every Out of the Ring result becomes "face into cage 3" and the opponent rolls Level 3 defense, and (lv) is ignored

**No DQ Match**
- No disqualifications and no count-outs
- Illegal moves and outside interference go unpunished

**Tag Team Match**
- Each team has 2 wrestlers
- The team on offense tags in the fresh partner once the man in the ring has built up 3 points of fatigue
- A defense result marked (tag) lets the defender try to tag out: two dice, 6 or less for regular partners, 4 or less for other teams. The partner comes in on Level 1 offense
- Only regular partners get pin saves, two per match, rolled on the Pin Saves chart. Any pin that is broken up adds 1 to the pinned wrestler's PIN rating
- A pin save can end in a double disqualification, which is a match with no winner
- The win and the loss go to the two wrestlers in the ring at the end

**Ringside Allies**
- When a wrestler's PIN rating gets dangerous his ally can storm the ring once per match (Outside Interference chart), which risks a disqualification
- The ally can also try once to distract the referee during a pin, rolled on the ally's own Distractor rating
- When the opponent goes out of the ring, the ally may attack him there

**Feud Match**
- If the roll that ends the match is doubles, the feud carries on after the bell and the Feud Table is rolled
- "You" on the table is the winner, or the wrestler who was disqualified
- Results that need an ally who is not at ringside are rolled again
- The table can injure a wrestler for one or two fight cards, or injure the opponent and suspend the attacker and his ally for a die roll of fight cards each

### Injuries and Suspensions

An injured wrestler can still wrestle with 2 added to his PIN rating. Injuries and suspensions count down by one for every fight card: one exhibition match, one whole Battle Royal, one whole tournament, or one whole federation show. A new injury does not lose a card on the card where it happened. Suspensions keep a wrestler off federation shows; in exhibition matches they are shown but not enforced.

### Battle Royal and Tournament

A Battle Royal is a gauntlet: two wrestlers fight, the loser is eliminated and the winner faces the next entrant, carrying his fatigue with him. A Tournament is a single-elimination bracket you can fill by hand or automatically.

## Settings

The rulebook leaves three decisions to the player. The simulator plays both sides, so Settings on the main menu switches each one on or off. All three start on.

- **Skip risky (c) chart moves** - A chart move marked (c) may be declined. The simulator declines when the opponent is rated A on that chart, and rolls again one level lower.
- **Skip risky dis moves** - The simulator declines a dis move when the disqualification number in play is 6 or more, and rolls again one level lower.
- **Use the number printed with dis** - When a card prints a number with a dis move, roll against it in place of the wrestler's Disqualification rating.

## Federation Mode

A federation has its own roster (at least 4 wrestlers), one or more championship belts (the first is the main title), a weekly show and a PPV every few weeks.

- **Creating one**: name, roster, belts, weekly show name, PPV names. Type a name and press ENTER to add it; BACKSPACE on an empty box removes the last one; TAB moves on. Adding no PPV names uses the default list.
- **Next Show** books a card for the week. Suspended wrestlers are left off it. Weekly shows are rivalry matches and filler. A PPV books a match for every title first, and the main title goes on last: the champion's defense, or a tournament when the title is vacant.
- **Editing the card**: pick a match to change its type and wrestlers. A tag match asks for two wrestlers per side and whether each team are regular partners. A wrestler who is suspended, already in the match, or booked elsewhere on the card cannot be picked. A title match stays a title match only while it is one on one with the champion in it. Battle royals and tournaments cannot be edited.
- **Running the show**: watch every match, simulate them all, or watch only the main event. ESC during a show asks whether to cancel the show (nothing from it is recorded) or skip to the results (the rest is simulated).
- **Titles**: a champion must have a title match by the end of each PPV. If a PPV passes without one, the title is vacated. The winner of a PPV Battle Royal earns the next shot at the main title.
- **Rivalries** build as wrestlers face each other and cool off over time. Rivals are booked against each other, with feud rules, and heated rivalries become No DQ or cage matches.
- **Standings & Records** and **Match & Title History** show records, streaks, every match with its type, and each title's changes.
- **Federation Settings** changes the names and how often PPVs run (every 2 weeks or more). Changes apply only when you pick Save & Return; ESC leaves without saving.

## Wrestler Cards

On the desktop, wrestler cards are YAML files in `data/wrestlers/`, loaded at startup. In the browser, the bundled cards are built in and the cards you create or edit are kept in browser storage.

### Card Format

```yaml
name: "Wrestler Name"

offense:
  # Level 1
  - - { name: "Arm Drag", power: 1, def_level: 1 }
    - { name: "Wrist Lock", power: 1, def_level: 1 }
    - { name: "Snapmare", power: 1, def_level: 1 }
    - { name: "Into the Ropes", power: 2, def_level: 1, tags: ["chart", "c"], chart: "ropes" }
    - { name: "Shin Breaker", power: 2, def_level: 2 }
    - { name: "Flying Elbow", power: 2, def_level: 2, tags: ["ag"] }
  # Level 2
  - - { name: "Knee Crusher", power: 2, def_level: 2 }
    - { name: "Suplex", power: 2, def_level: 2 }
    - { name: "Into the Turnbuckle", power: 2, def_level: 2, tags: ["chart"], chart: "turnbuckle" }
    - { name: "Choice", power: 2, def_level: 2, tags: ["ch"], choice: "E" }
    - { name: "Low Blow", power: 3, def_level: 3, tags: ["dis"], dis_number: 7 }
    - { name: "Belly to Back Suplex", power: 3, def_level: 3, tags: ["add1"] }
  # Level 3
  - - { name: "Knee Drop", power: 3, def_level: 3 }
    - { name: "Power Slam", power: 3, def_level: 3, tags: ["pw"] }
    - { name: "Out of the Ring", power: 3, def_level: 3, tags: ["chart"], chart: "ring" }
    - { name: "Deathjump", power: 3, def_level: 3, tags: ["chart"], chart: "deathjump" }
    - { name: "Double Team", power: 3, def_level: 3, tags: ["tag"] }
    - { name: "FINISHING MOVE", power: 3, def_level: 3 }

defense:
  # Level 1
  - - { type: "reversal", power: 1 }
    - { type: "dazed", power: 1 }
    - { type: "reversal", power: 1 }
    - { type: "dazed", power: 1 }
    - { type: "reversal", power: 2 }
    - { type: "dazed", power: 1, tags: ["tag"] }
  # Level 2
  - - { type: "dazed", power: 2 }
    - { type: "reversal", power: 1 }
    - { type: "hurt", power: 2 }
    - { type: "reversal", power: 2 }
    - { type: "hurt", power: 2 }
    - { type: "down", power: 3 }
  # Level 3
  - - { type: "hurt", power: 3 }
    - { type: "reversal", power: 2 }
    - { type: "down", power: 3, tags: ["lv"] }
    - { type: "down", power: 3 }
    - { type: "pin", power: 0 }
    - { type: "reversal", power: 3 }

ropes: "B"          # Chart ratings: A, B, or C
turnbuckle: "B"
ring: "B"
deathjump: "C"

pin: 5              # PIN rating for the basic rules (kept on the card, not used by the simulator)
pin_adv: 2          # PIN rating in parentheses: where the wrestler starts under the fatigue rules
cage: 5             # Starting PIN rating in a cage match
dq: 5               # Disqualification rating: disqualified on two dice at or under this
agility: -1         # -5 (excellent) to +5 (poor)
power: 0            # -5 (excellent) to +5 (poor)
distractor: 6       # Used when this wrestler is someone's ringside ally (default 5)

finisher:
  name: "FINISHING MOVE"   # Must match the capital-letter move on Level 3
  rating: 4                # Added to the opponent's PIN rating
  # For a roll finisher, in place of a rating:
  # is_roll: true
  # roll_min: 2
  # roll_max: 6
```

Every card needs all 3 levels of offense and defense with 6 entries each. `power` on a move is the number printed after it, and `def_level` is the defense level the opponent rolls on, which on a card is the same number.

### Move Instructions

| `tags` value | Meaning |
|-----|---------|
| `chart` | Chart move. Needs a `chart` field |
| `c` | With `chart`: the move is optional, the (c) on a card |
| `ch` | Choice. Needs a `choice` field with a letter A to H |
| `ag` | Works only with equal or better Agility |
| `pw` | Works only with equal or better Power |
| `dis` | Illegal move. An optional `dis_number` field is the number printed with it |
| `add1` | Adds 1 to the opponent's PIN rating |
| `tag` | Tag matches only |
| `singles` | Singles matches only |

### Chart Names

| `chart` value | Chart |
|-------|-------|
| `ropes` | Into the Ropes |
| `turnbuckle` | Into the Turnbuckle |
| `ring` | Out of the Ring |
| `deathjump` | Deathjump |

### Defense Results

| `type` value | Effect |
|------|--------|
| `dazed` | Attacker stays on offense, on the level of `power` |
| `hurt` | Attacker stays on offense, on the level of `power` |
| `down` | Attacker stays on offense, on the level of `power`. A Down 3 can carry `tags: ["lv"]` |
| `reversal` | Defender takes over offense, on the level of `power` |
| `pin` | PIN attempt. `power` is not used |

Any defense result can carry `tags: ["tag"]` for the tag-out instruction.

## Creating Cards

You can create wrestler cards two ways:

1. **In-Game Editor** - Select "Create New Card" from the main menu, fill in the fields and press Ctrl+S. The card is saved under a file name made from the wrestler's name and joins the roster straight away. Saving a card under a new name saves a new card and keeps the original; the editor says so.

2. **Manual YAML** (desktop) - Create a `.yaml` file in `data/wrestlers/` following the card format above. The game loads all cards from this directory on startup.

### Editor Lines

Each move and each defense result is one line of text. The instructions go after the numbers, separated by spaces:

| Line | Meaning |
|------|---------|
| `Arm Drag,1,1` | name, number, defense level |
| `Flying Elbow,2,2,ag` | an (ag) move |
| `Big Slam,3,3,pw add1` | two instructions |
| `Into the Ropes,2,1,ropes` | chart move: the chart name |
| `Into the Ropes,2,1,ropes c` | optional chart move, (c) |
| `Choice,2,2,ch E` | Choice E |
| `Low Blow,3,3,dis` | illegal move on the wrestler's own rating |
| `Low Blow,3,3,dis 7` | illegal move with its own number |
| `Double Team,3,3,tag` | tag matches only |
| `down,3,lv` | defense result with the option to leave the ring |
| `dazed,1,tag` | defense result with a tag-out attempt |
| `pin,0` | PIN |

The **Finisher Roll (min-max)** field is blank for a normal finisher. For a roll finisher, enter the range on one die, for example `2-6`.

### Tips for Card Design

- **PIN in parentheses (`pin_adv`) of 2 to 4** is typical. Lower is harder to pin. It rises by 1 with every kick-out.
- **Agility and Power** run from -5 (excellent) to +5 (poor). High-flyers want Agility around -3 to -5, powerhouses want Power around -3 to -5.
- **Disqualification 3 to 5** is typical. A higher number is disqualified more often.
- **Finisher rating +2 to +5** is typical. Higher is more devastating.
- More reversals in defense make a wrestler harder to keep on offense against.
- More PINs in defense mean more pin attempts against him.

## Included Wrestlers

The game comes with 8 example wrestler cards:

- **Butcher Briggs** - Brawling powerhouse (Power -4, Agility +1)
- **Rico Stormcloud** - High-flying technician (Agility -4, Power 0)
- **Rex Fontaine** - Dirty technical wrestler (Disqualification 5, well-rounded)
- **Armand the Colossus** - Unstoppable giant (Power -5, Agility +3)
- **Buck Stallion** - All-American powerhouse (Power -5, Agility +2)
- **Ricky Rampage** - High-flying brawler (Agility -2, Power -1)
- **The Gravedigger** - Supernatural powerhouse (Power -4, Agility +1)
- **Blake Harton** - Technical excellence (Agility -2, Disqualification 5)

## Not Implemented

- The Superstar Pro Wrestling rules. Ring Wars follows the Filsinger Games rulebook only.
- Six-man and eight-man tag matches.
- Ringside allies in federation matches.
