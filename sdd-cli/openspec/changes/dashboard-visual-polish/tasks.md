# Apply: dashboard-visual-polish

## Completed Tasks

- [x] Create `buildBackground()` function with offscreen canvas
- [x] Composite bg-scene.png + floor zone + floor tiles into offscreen canvas
- [x] Call `buildBackground()` on asset load and window resize via `bgDirty` flag
- [x] Replace drawScene background calls with single `ctx.drawImage(bgCanvas, 0, 0)`
- [x] Add `hudDirty` boolean state variable, initialized `true`
- [x] Set `hudDirty = true` in all WebSocket message handlers
- [x] Add `workerAnimTimer` and `workerAnimFrame` accumulators
- [x] Worker sprites cycle at 4 fps using delta-time accumulator
- [x] Conveyor belt uses continuous `conveyorOff` scrolled by dt
- [x] Star twinkle uses `sin(animTime + star.phase)` for smooth fade
- [x] Particle movement uses dt multiplier for frame-rate independence
- [x] Create `txtC()` helper using `measureText()` for centering
- [x] Replace hardcoded offsets in empty pipeline message
- [x] Replace hardcoded offsets in empty error message
- [x] Use `P.label` (#668) color for empty state text
- [x] Add animated ellipsis on "Waiting for quests..."
- [x] Cap delta-time at 100ms to prevent animation jumps on tab switch
