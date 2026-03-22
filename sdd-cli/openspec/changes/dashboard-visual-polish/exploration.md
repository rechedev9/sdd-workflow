# Exploration: dashboard-visual-polish

## Current State

### Rendering approach

The dashboard is a single-page SNES-style canvas app served from an embedded HTTP server on `:8811`. It renders a 480x270 logical resolution game scene plus a HUD overlay, all on one `<canvas>` element.

**Critical mismatch with design doc:** The `DASHBOARD-DESIGN.md` recommends drawing at native 480x270 and letting CSS `image-rendering: pixelated` scale up. The actual implementation (`dashboard.js`) does the opposite — it sets `canvas.width/height` to the full viewport dimensions and multiplies every coordinate by a scale factor `S` manually. This means every frame renders at native viewport resolution (e.g., 1920x1080 = 16x more pixels than 480x270). The CSS in `base.html` has no `image-rendering: pixelated` rule at all.

### Assets

- `sprites.png` — 256x256 sprite sheet (workers, stations, floor tile)
- `bg-scene.png` — 960x200 pre-rendered background
- Both loaded via `new Image()` with simple boolean flags (`sprOk`, `bgOk`)
- No offscreen canvas; no asset precomposition

### Data flow

1. Go `Hub` polls filesystem + SQLite every 1 second (`hub.go:82`)
2. Hub diffs via SHA-256 content hashing; only broadcasts changed data
3. JS receives `kpi`, `pipelines`, `errors` message types over WebSocket
4. Chart messages (`chart:heatmap`, `chart:tokens`, `chart:durations`, `chart:cache`, `chart:verify`) are sent but **not consumed** by `dashboard.js` — no chart rendering code exists in the JS

### Game loop

- `requestAnimationFrame` drives rendering at display refresh rate
- `animTime` accumulates delta-time in seconds
- `drawScene(animTime)` and `drawHUD()` called every frame
- Animation uses `t*3|0` (integer frame counter from elapsed time) — coarse 3 fps effective animation rate

---

## Performance Issues Found

### P1: Full-resolution rendering (16x pixel overhead)

**File:** `dashboard.js:285-309`

Canvas is sized to viewport (`canvas.width = vw; canvas.height = vh`) and every primitive multiplies coordinates by `S`. At 1920x1080 this processes ~2M pixels per frame vs. the intended ~130K at 480x270. The design doc explicitly warns against this approach (`DASHBOARD-DESIGN.md:33-37`).

### P2: Background image redrawn every frame

**File:** `dashboard.js:113-115`

`ctx.drawImage(bgImg, ...)` called every frame to draw the static background. This is a full-width stretch blit from 960x200 source to viewport-width destination — one of the most expensive single draw calls in the loop.

### P3: Excessive floor tile drawImage calls

**File:** `dashboard.js:131`

```js
for(var tx=0;tx<LW;tx+=14) spr(FTILE.x,FTILE.y,SP,SP, tx,FLOOR_Y+1,14,14);
```

At logical width 480, this is `Math.ceil(460/14) = 33` `drawImage` calls per frame for static floor tiles that never change. Combined with full-resolution rendering, each call is a scaled blit.

### P4: Conveyor belt redrawn from scratch every frame

**File:** `dashboard.js:136-138`

`~117` `fillRect` calls per frame (`(LW-20)/4 = 115` iterations) for the conveyor animation, which only changes pattern every few frames.

### P5: HUD redrawn completely every frame

**File:** `dashboard.js:182-263`

The entire HUD (panels, borders, text, progress bars, tables) is re-rendered every frame with dozens of `fillRect` + `fillText` calls, even though it only changes when WebSocket data arrives (1 Hz at most).

### P6: Stars re-evaluated every frame

**File:** `dashboard.js:122-125`

50 star visibility checks + potential `fillRect` calls per frame. Stars are static positions with a slow twinkle cycle — could be pre-rendered to an offscreen canvas.

### P7: Particle splice in hot loop

**File:** `dashboard.js:96`

`Array.splice(i, 1)` in reverse iteration. For up to 30 particles this is fine, but the pattern allocates and shifts. Minor concern.

---

## Visual Issues Found

### V1: No CSS `image-rendering: pixelated`

**File:** `templates/base.html:10-14`

The `<canvas>` style has no `image-rendering` property. Since the canvas renders at native resolution with manual scaling, subpixel rounding causes inconsistent pixel sizes. Text rendered via `fillText` with fractional-scaled font sizes looks blurry rather than crisp pixel-art.

### V2: Font sizing produces non-integer px values

**File:** `dashboard.js:76`

```js
ctx.font="bold "+Math.max(1,Math.round((sz||5)*S))+"px monospace";
```

At common `S` values (e.g., 4.0, 3.5556), `Math.round(5*3.5556)` = 18px — this works, but at `sz=4` it produces `Math.round(4*3.5556)` = 14px. The mix of 14px and 18px monospace text lacks the SNES pixel-font aesthetic. The design doc recommends drawing at 480x270 where font sizes would be literal pixel heights (4px, 5px, 6px, 7px, 8px).

### V3: Empty state messages are generic and hard to read

**File:** `dashboard.js:223`

```js
txt("No active quests",LW/2-35|0,tY+10,P.dim,5);
```

Hardcoded horizontal offset (`LW/2-35`) is a rough approximation of centering. The message "No active quests" in dim color `#335` against panel background `#10102a` has very low contrast (both are dark blue).

**File:** `dashboard.js:246`

```js
txt("No enemies encountered",LW/2-45|0,eY+10,"#443333",5);
```

"No enemies encountered" uses `#443333` (dark red-brown) — even lower contrast. No visual indicator (icon, decoration) to soften the empty state.

### V4: Animation is frame-count based, not delta-time

**File:** `dashboard.js:110`

```js
var anim = t*3|0;
```

While `t` is delta-time accumulated, truncating to integer and using modular arithmetic (`anim%20`, `anim%30`, etc.) creates animation that runs at exactly 3 "ticks" per second regardless of frame rate. This means:
- Worker sprite animation: `(anim/3+w|0)%4` = roughly 1 fps sprite cycling — very slow
- Star twinkle: `(anim+s.b)%80 < 55` = deterministic but choppy
- Conveyor: `(anim%4)*2` = 0.75 Hz belt movement

The animations work but feel sluggish. True delta-time interpolation would allow smoother transitions.

### V5: HUD text alignment is pixel-approximate

Throughout `drawHUD()`, column positions for the pipeline table are hardcoded magic numbers: `8, 120, 210, 295, 340, 430`. These don't adapt to different logical widths or content, and the "ST" column header at x=430 is near the right edge of the 480-wide canvas with only 50px margin.

---

## Opportunities

### O1: Switch to native-resolution canvas + CSS upscaling (high impact)

Revert to the design doc approach: set `canvas.width=480; canvas.height=270`, remove the `S` multiplier entirely, add `image-rendering: pixelated` to CSS. This:
- Reduces pixel fill by ~16x
- Eliminates all `Math.floor(x*S)` calculations
- Makes font sizes literal pixel heights (crisp pixel-font look)
- Aligns implementation with `DASHBOARD-DESIGN.md`

**Files:** `dashboard.js:285-309` (initCanvas), `dashboard.js:59-78` (spr, px, txt), `templates/base.html:10-14` (CSS)

### O2: Offscreen canvas for static background layer (medium impact)

Composite `bgImg` + stars + floor tiles + conveyor belt base into a single offscreen canvas once (and on resize). Main loop draws one `drawImage` call for the entire background layer instead of ~85+ draw calls.

**File:** `dashboard.js:109-138` (drawScene)

### O3: Dirty-flag HUD rendering (medium impact)

Only redraw the HUD when WebSocket data changes. Add a `hudDirty` flag set by `ws.onmessage`, cleared after draw. The game loop can skip `drawHUD()` entirely on most frames.

**Files:** `dashboard.js:320-327` (onmessage), `dashboard.js:270-283` (gameLoop)

### O4: Delta-time interpolation for animations (low-medium impact)

Replace `var anim = t*3|0` with proper per-entity timers:
- Worker sprites: smooth frame cycling at configurable fps (e.g., 6 fps)
- Star twinkle: sine-wave opacity instead of binary on/off
- Conveyor: continuous scroll offset

**File:** `dashboard.js:109-175` (drawScene)

### O5: Improve empty state visual feedback (low impact, high polish)

- Center text properly using `ctx.measureText()` instead of hardcoded offsets
- Add a subtle pulsing animation or decorative element (pixel-art shield, sword icon)
- Increase contrast: use `P.label` (`#668`) instead of `P.dim` (`#335`) for empty messages
- Add a "Waiting for quests..." with ellipsis animation

**File:** `dashboard.js:222-224, 245-247`

### O6: Pre-render text to offscreen canvases (low impact)

Static labels ("CHANGES", "TOKENS", "CACHE", "ERRORS", column headers) could be pre-rendered once. Reduces `fillText` calls which are expensive at scale.

**File:** `dashboard.js:182-263` (drawHUD)

---

## Risk Assessment

| Change | Risk | Difficulty | Notes |
|--------|------|-----------|-------|
| O1: Native resolution + CSS | **Medium** | Medium | Most impactful change. Requires removing S multiplier from every draw call. Risk: coordinate math errors, text readability at small pixel sizes. Mitigated by: the design doc already describes the target state. |
| O2: Offscreen background | **Low** | Low | Isolated to drawScene. Offscreen canvas created once, redrawn on resize. No interaction with HUD or WebSocket. |
| O3: Dirty-flag HUD | **Low** | Low | Simple boolean flag. Worst case: missed update on a frame, visible next frame (16ms later). |
| O4: Delta-time animation | **Low** | Low | Purely visual, no data coupling. Can be done incrementally per element. |
| O5: Empty state polish | **Very Low** | Very Low | Cosmetic text/color changes. No logic impact. |
| O6: Pre-render text | **Low** | Medium | `measureText` + offscreen canvas management adds complexity. Benefit is marginal if O1 is done (480x270 text is cheap). |

### What could break

- **O1** is the riskiest: if any draw call still uses `*S` after the refactor, it will draw at wrong position. Systematic search-and-replace of the `S` multiplier is required. The `spr()`, `px()`, `txt()` wrappers centralize most usage, which helps.
- **O1** could affect text readability: at 480x270, font size 4px is genuinely tiny. May need to increase some font sizes or test on target displays.
- **O2** interacts with resize: the offscreen canvas must be invalidated on `window.resize`.
- None of these changes affect Go code, WebSocket protocol, or data flow. All risk is confined to `dashboard.js` and `base.html`.

---

## Recommended Approach

**Priority order (do in sequence, each independently testable):**

1. **O1: Switch to 480x270 native canvas + CSS pixelated scaling**
   - Highest impact on both performance and visual crispness
   - Remove `S` from `spr()`, `px()`, `txt()`, `initCanvas()`
   - Add `image-rendering: pixelated` + vendor prefixes to `base.html`
   - Set `canvas.width=LW; canvas.height=LH` fixed
   - Adjust font sizes for readability at native resolution

2. **O2: Offscreen canvas for static background**
   - Create `bgCanvas` offscreen, render bg image + stars + floor tiles once
   - `drawScene()` becomes: `ctx.drawImage(bgCanvas, 0, 0)` + conveyor animation + workers
   - Invalidate on asset load completion and window resize

3. **O5: Empty state visual polish**
   - Quick win while touching drawHUD
   - Better contrast colors, centered text, subtle animation

4. **O4: Delta-time animation smoothing**
   - Smoother worker sprites, star twinkle, conveyor scroll
   - Replace `anim` integer with per-system accumulators

5. **O3: Dirty-flag HUD rendering**
   - Add `hudDirty` flag, skip HUD redraw when data unchanged
   - After O1, HUD rendering is cheap (480x270), so this becomes lower priority

6. **O6: Pre-render static text** — skip unless profiling shows text is still a bottleneck after O1

## Relevant Files

- `internal/dashboard/static/dashboard.js` — Canvas rendering, game loop, HUD, WebSocket client
- `internal/dashboard/templates/base.html` — HTML template with canvas element
- `internal/dashboard/static/sprites.png` — Character and station sprite sheet (256x256)
- `internal/dashboard/static/bg-scene.png` — Pre-rendered background scene (480x108)
- `internal/dashboard/hub.go` — WebSocket hub, poll loop, data broadcasting
- `internal/dashboard/server.go` — HTTP server, routing, embedded assets
- `internal/dashboard/DASHBOARD-DESIGN.md` — Design reference document
