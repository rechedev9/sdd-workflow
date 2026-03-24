---
id: 0005
title: Dashboard: offscreen canvas + HUD dirty flag
status: implemented
change: dashboard-visual-polish
date: 2026-03-23
supersedes: ~
superseded-by: ~
---

## Decision

The dashboard canvas rendering uses two optimizations: (1) static background layers are pre-rendered into an offscreen canvas and composited with a single `drawImage` call per frame, and (2) the HUD is only redrawn when WebSocket data changes, tracked via a `hudDirty` boolean flag.

```
Game Loop (60fps):
  1. ctx.drawImage(bgCanvas, 0, 0)          ← static layer, 1 draw call
  2. drawConveyorAnim(dt)                    ← animated overlay
  3. drawStations(dt)                        ← stations + workers
  4. drawParticles(dt)                       ← particles
  5. if(hudDirty) { drawHUD(); hudDirty=false; }  ← conditional
```

## Context

The original rendering pipeline drew the full background (bg image + floor tiles + conveyor base) and the full HUD on every frame at 60fps. This is CPU-wasteful — the background is static (changes only on resize or asset load) and the HUD changes only on WebSocket messages, not every frame.

CSS `image-rendering: pixelated` was the first option considered for scaling quality, but it doesn't work reliably on the target environment (Chrome/Windows). The offscreen canvas approach solves the scaling and performance problems in one step.

## Alternatives Considered

- **CSS pixelated scaling** — rejected because it does not work on the target Chrome/Windows environment.
- **WebGL/WebGPU** — rejected as over-engineering for a developer dashboard; maintenance cost not justified.
- **requestAnimationFrame throttling (lower fps)** — rejected because it degrades animation smoothness for conveyor/star animations.
- **Redraw full canvas only on data change** — rejected because it would stutter the conveyor and star animations that must run continuously.

## Consequences

**Positive:**
- Static background cost reduced from O(tiles) draw calls per frame to 1 `drawImage`.
- HUD redraws reduced from 60/s to data-change rate (typically <1/s in normal operation).
- Offscreen canvas automatically rebuilt on `window.resize` and asset load.

**Negative:**
- Additional canvas element held in memory (~`LW*S × SCENE_END*S` pixels).
- `buildBackground()` must be called on both `bgImg.onload` and `sprImg.onload` to handle async asset loading order.
- `hudDirty = true` must be set in every WebSocket message handler — easy to forget when adding new message types.

## References

- Change: `openspec/changes/archive/2026-03-23-123133-dashboard-visual-polish/`
- File: `internal/dashboard/static/dashboard.js`
- Source design: `openspec/changes/archive/2026-03-23-123133-dashboard-visual-polish/design.md`
