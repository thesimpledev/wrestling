# Future Updates

## Potential Port: Go/Ebitengine → Pure JavaScript (Web)

Considering migrating the project from Go + Ebitengine to a pure JavaScript/TypeScript web app so the game is playable online with no install.

### Why it fits

- The game is **text-driven**. There's no real-time rendering, no sprites, no physics — just scrolling text and menus. Ebitengine is overkill.
- **Instant shareability**: deploy to any static host (GitHub Pages, Netlify, Vercel) and anyone with a browser can play. No binaries, no platform builds.
- **Lower friction for iteration**: hot reload, browser devtools, easier UI work (HTML/CSS beats manual text layout).
- **Career persistence** maps naturally to `localStorage` or IndexedDB instead of `career.json`.

### What ports cleanly

Most of the engine is pure functions over data, which translates to TS with minimal friction:

- `engine/match.go` — match resolution logic
- `engine/charts.go` — probability/outcome charts
- `engine/wrestler.go` — wrestler model
- YAML card data in `data/wrestlers/` — convert to JSON (or keep YAML with a loader)
- Federation system (named feds, rosters, belts, show schedules)
- W/L records, championships, rivalries

### What gets rewritten

- **UI layer** (`ui/game.go`): replaced with HTML/CSS + a small render loop or a framework (vanilla TS, Svelte, or React).
- **Event/UI split**: the engine-emits-events / UI-consumes-events pattern survives — just swap Ebitengine's loop for DOM events or a tiny pub/sub.
- **Input handling**: keyboard events instead of Ebitengine key polling. The ESC/Space/Enter flows all map 1:1.
- **Persistence**: `career.json` → `localStorage` (or IndexedDB if it grows).

### Tradeoffs

| Pro | Con |
| --- | --- |
| Zero-install, shareable URL | Lose the native desktop feel |
| Easier UI iteration | Rewrite the UI layer |
| Browser devtools | Lose Go's type system (mitigated by TS) |
| Trivial deploy | Need to re-home save data |
| Potential for multiplayer/online features later | Save files no longer portable as plain files |

### Possible stack

- **TypeScript** for type safety comparable to Go
- **Vite** for dev server + build
- **Svelte or vanilla TS** for UI (keeping it lightweight matches the current text-first feel)
- **localStorage** for career saves, with optional JSON export/import for portability

### Open questions

- Keep YAML for card data (with a browser YAML parser) or convert to JSON?
- Single-player only, or lean into the web platform and add async multiplayer / shared leaderboards down the line?
- Do we want to preserve existing `career.json` files via import, or start fresh?

### Status

**Under consideration.** No work started. Current Ebitengine build continues to be the active version until a decision is made.
