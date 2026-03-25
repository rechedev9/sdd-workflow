# Proposal: dashboard-visual-polish

## Intent

Optimize the SNES-style dashboard rendering performance and visual polish. Reduce unnecessary draw calls per frame by caching static layers in offscreen canvases, add dirty-flag HUD rendering so the HUD only redraws when WebSocket data changes, smooth animations with proper delta-time accumulators, and improve empty state readability.

## Scope

**In scope:**
- Offscreen canvas for static background (bg image + floor tiles → 1 drawImage per frame)
- Dirty-flag HUD rendering (skip redraw when data unchanged)
- Smoother sprite/star/conveyor animations via delta-time accumulators
- Empty state text centering and contrast improvement
- HUD column alignment cleanup

**Out of scope:**
- CSS pixelated scaling (doesn't work on target Chrome/Windows)
- WebGL/WebGPU migration
- New sprite assets
- Go backend changes

**Files modified:** `internal/dashboard/static/dashboard.js` only

## Relevant Files

- `internal/dashboard/static/dashboard.js` — Canvas rendering, game loop, HUD
- `internal/dashboard/templates/base.html` — HTML canvas element
- `internal/dashboard/static/sprites.png` — Sprite sheet
- `internal/dashboard/static/bg-scene.png` — Background scene
