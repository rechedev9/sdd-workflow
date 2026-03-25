# Specification: dashboard-visual-polish

## Overview

Performance and visual polish improvements for the SNES-style canvas dashboard. All changes in `dashboard.js`, no Go backend modifications.

## S1: Offscreen Background Canvas

Cache the static background layer (bg image + floor tiles) in an offscreen canvas. Render once on asset load and on window resize. Main loop draws a single `drawImage` call for the entire background.

### Requirements
- Create `OffscreenCanvas` (or fallback `document.createElement('canvas')`) at `LW*S × BG_END*S` dimensions
- Composite: bg-scene.png + floor tiles + conveyor base pattern
- Invalidate on: window resize (S changes), asset load completion
- Main drawScene: replace ~50 individual draw calls with `ctx.drawImage(bgCanvas, 0, 0)`

## S2: Dirty-flag HUD Rendering

Only redraw HUD when data changes. The hub pushes data at 1 Hz but the canvas redraws at 60 fps.

### Requirements
- Add `hudDirty` boolean, initialized `true`
- Set `hudDirty = true` in every WebSocket message handler
- `drawHUD()` only runs when `hudDirty === true`, then sets it `false`
- On first frame, HUD must draw (initial state)
- HUD canvas area cleared only when dirty (preserve last drawn state)

## S3: Animation Smoothing

Replace integer-truncated animation counter with proper delta-time accumulators.

### Requirements
- Worker sprite cycling: 4 fps (frame advances every 250ms)
- Star twinkle: smooth sine-wave opacity, period 2-4 seconds per star
- Conveyor belt: continuous scroll at 8 pixels/sec
- Particle system: position updates use `dt` directly (already partially done)

## S4: Empty State Polish

Improve readability of "No active quests" and "No enemies encountered" messages.

### Requirements
- Center text using `ctx.measureText().width` instead of hardcoded offsets
- Use `P.label` (#668) for empty messages instead of `P.dim` (#335) or `#443333`
- Add animated ellipsis: "Waiting for quests..." cycles dots every 500ms
