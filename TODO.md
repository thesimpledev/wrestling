# Ring Wars - TODO

## Goal
Recreate a DOS-style wrestling match simulator like the one my brother used to have years ago. He had a program that would run card-based wrestling matches in a sensationalized format. We're building him a similar program where he can input his wrestler cards and have it simulate matches for him.

## Project Status: Feature Complete

## What's Implemented
- [x] Core match engine with event system
- [x] All wrestler card data models (offense/defense grids, ratings, finisher)
- [x] YAML card loader and saver
- [x] Dice rolling (1d6, 2d6)
- [x] Basic match loop (offense -> defense -> reversal/continue)
- [x] PIN resolution with fatigue system
- [x] Finisher resolution (including roll finishers)
- [x] All 8 game charts (Ropes, Turnbuckle, Out of Ring, Deathjump, Choice Situations, Feud Table, Pin Saves, Outside Interference)
- [x] Agility/Power move checks
- [x] Disqualification mechanics
- [x] Count-out mechanics
- [x] Add1 fatigue moves
- [x] Leaving the ring option
- [x] Referee down (from Deathjump chart)
- [x] Cage match (uses Cage rating, no DQ/countout, face-into-cage)
- [x] No-DQ match
- [x] Tag team matches (tag in/out, pin saves, double DQ)
- [x] Ebitengine UI with scrolling text commentary
- [x] 2x scaled chunky DOS-style font
- [x] Step-through (SPACE) and auto-play (A) with speed control (+/-)
- [x] Resizable window
- [x] Main menu (select match type, select wrestlers)
- [x] In-app card editor (create new / edit existing)
- [x] 8 wrestler cards (Briggs, Stormcloud, Fontaine, Colossus, Stallion, Rampage, Gravedigger, Harton)
- [x] Outside interference mechanic (rolls on interference chart during PIN/down-3)
- [x] Distraction mechanic (manager distracts ref before PIN roll)
- [x] Ringside ally interactions (ally attacks during Out of Ring)
- [x] Feud match mode (post-match feud table on doubles)
- [x] Injury tracking across matches (persists to JSON, +2 PIN penalty)
- [x] Better commentary text (randomized dramatic commentary)
- [x] Card validation in editor (validates all fields before save)

## What's Left
- [ ] Visual ring sprite / wrestler sprites (cosmetic)

## How to Run
```
go run .
```
Cards are loaded from `data/wrestlers/`. Use the in-game menu to select match type and wrestlers, or create/edit cards.

## Controls
- **UP/DOWN** - Navigate menus, scroll match text
- **ENTER/SPACE** - Confirm selection, advance match text
- **A** - Toggle auto-play during match
- **+/-** - Adjust auto-play speed
- **R** - Rematch (after match ends)
- **ESC** - Back / quit
- **Ctrl+S** - Save card (in editor)

## Match Types
- **Singles** - Standard 1v1 match
- **Tag Team** - 2v2 with tag in/out, pin saves
- **Cage** - Uses Cage rating, no DQ/countout, face-into-cage
- **No DQ** - No disqualifications, anything goes
- **Feud** - Singles match with post-match feud table (injuries possible)

## Ally System
After selecting wrestlers in non-tag matches, you can assign a ringside ally to each side. Allies enable:
- **Outside Interference** - When your wrestler rolls down-3 or PIN, ally can storm the ring (once per match)
- **Distraction** - Ally can distract referee before a PIN roll (once per match)
- **Ringside Attack** - Ally attacks opponent when thrown out of the ring

