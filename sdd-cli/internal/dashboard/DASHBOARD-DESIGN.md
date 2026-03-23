# SNES-style Dashboard — Design & Implementation Reference

## Architecture

### Rendering Pipeline (MDN recommended approach)

The correct way to render crisp pixel art in a browser:

1. **Canvas at LOW internal resolution** (e.g., 480x270)
2. **CSS scales it up** to fill the viewport
3. `image-rendering: pixelated` on the canvas element prevents blur
4. **Never scale via drawImage or ctx.scale** — let CSS handle upscaling

```html
<canvas id="scene" width="480" height="270"></canvas>
```

```css
canvas {
  width: 100vw;
  height: 100vh;
  image-rendering: -moz-crisp-edges;
  image-rendering: -webkit-crisp-edges;
  image-rendering: pixelated;
  image-rendering: crisp-edges;
}
```

**Critical rule:** All `drawImage()` calls use **integer** x, y, width, height. Fractional coordinates cause blurring regardless of CSS settings.

### Why NOT render at full resolution

Our previous approach (rendering at 1920x1080 with manual S multiplier) was wrong because:
- Canvas operations at 1920x1080 are ~16x more pixels than 480x270
- Background image `drawImage` from 960x200 → 1920x1080 uses bilinear by default
- `imageSmoothingEnabled = false` on canvas context only affects `drawImage`, not fill/stroke
- The correct approach is: draw at game resolution, CSS scales to screen

### devicePixelRatio Caveat

When CSS pixels don't align with device pixels (non-integer devicePixelRatio), some pixels render larger than others. This is unavoidable — accept it. Integer DPR (1x, 2x, 3x) gives perfect results.

## Game Loop

```javascript
var lastTime = 0;

function gameLoop(timestamp) {
  var dt = (timestamp - lastTime) / 1000; // seconds
  lastTime = timestamp;

  update(dt);   // game logic
  render();     // draw everything

  requestAnimationFrame(gameLoop);
}
```

Animation should be delta-time based, not frame-count based, to work across refresh rates.

## Sprite Sheet Rendering

```javascript
// Extract frame from sprite sheet
function drawSprite(sheet, frameX, frameY, frameW, frameH, destX, destY, destW, destH) {
  ctx.drawImage(sheet,
    frameX, frameY, frameW, frameH,  // source rect (in sheet)
    Math.floor(destX), Math.floor(destY), // dest position (INTEGER!)
    Math.floor(destW), Math.floor(destH)  // dest size (INTEGER!)
  );
}
```

**Integer coordinates are mandatory** for crisp rendering.

## Sprite Sheet Layout (sprites.png — 256x256)

| Row | Y | Content | Frame Size | Frames |
|-----|---|---------|-----------|--------|
| 0 | 0 | Blue Mage (scholar) | 32x32 | 4 (idle, work1, work2, work3) |
| 1 | 32 | Red Knight (builder) | 32x32 | 4 |
| 2 | 64 | Green Healer (inspector) | 32x32 | 4 |
| 3 | 96 | Gold Scribe (writer) | 32x32 | 4 |
| 4 | 128 | Workstations | 32x32 | off(0,128), on(64,128) |
| 5-6 | 160 | Dragon (unused currently) | 64x64 | 2 |
| 7 | 224 | Floor tile | 32x32 | 1 at (192,224) |

## Worker-Phase Assignment

| Phases | Worker Type | Sprite | Rationale |
|--------|------------|--------|-----------|
| explore, propose | Blue Mage | Row 0 | Scholars investigate and propose |
| spec, design | Gold Scribe | Row 3 | Writers create specifications |
| tasks, apply | Red Knight | Row 1 | Builders implement code |
| review, verify, clean, archive | Green Healer | Row 2 | Inspectors verify quality |

## Background Scene (bg-scene.png — 960x200)

Pre-rendered background covering sky + lab wall area. Drawn at canvas internal resolution, NOT stretched. The floor zone below is drawn programmatically in dark color (#1a1a30) so stations and workers pop visually.

## Color Palette (max 24 colors)

| Name | Hex | Usage |
|------|-----|-------|
| bg | #0a0a1a | Body/canvas clear |
| sky1 | #0a0a2e | Upper sky |
| sky2 | #0e1030 | Lower sky |
| hudBg | #0c0c20 | HUD background |
| panel | #10102a | Card/table backgrounds |
| ground | #1a1a30 | Floor area |
| border | #2a2a4a | Panel borders |
| label | #668 | Dimmed label text |
| dim | #335 | Inactive/empty text |
| white | #dde | Primary text |
| cyan | #4af | Primary accent, active labels |
| green | #4f4 | Success, tokens |
| yellow | #ff4 | Warning, in-progress |
| red | #f55 | Error |
| purple | #c8f | Cache/secondary metric |
| star | #556 | Star twinkle |

## Layout (480x270 logical resolution)

```
┌─────────────────────────────────── 480px ──────────────────────────────────┐
│ bg-scene.png (sky + wall)                                    0..108       │
│ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─             │
│ Floor zone (dark, stations + workers here)                   108..148     │
│ ═══════════════════════════════════════════════════════════   148 (border) │
│ HUD header bar (title + LIVE indicator)                      150..162     │
│ KPI cards (4x: Changes, Tokens, Cache, Errors)              164..186     │
│ Pipeline table header + rows                                 188..230     │
│ Error table header + rows                                    232..260     │
│ Legend bar                                                   262..270     │
└───────────────────────────────────────────────────────────────────────────┘
```

## References

- [MDN: Crisp pixel art look](https://developer.mozilla.org/en-US/docs/Games/Techniques/Crisp_pixel_art_look)
- [Belén Albeza: Retro crisp pixel art in HTML5 games](https://www.belenalbeza.com/articles/retro-crisp-pixel-art-in-html-5-games/)
- [Scaling sprites for pixel art with HTML5 canvas](https://chewett.co.uk/blog/1177/scaling-sprites-for-pixel-art-with-html5-canvas/)
- [Spicy Yoghurt: Sprite animations tutorial](https://spicyyoghurt.com/tutorials/html5-javascript-game-development/images-and-sprite-animations)
- [Spicy Yoghurt: Game loop with requestAnimationFrame](https://spicyyoghurt.com/tutorials/html5-javascript-game-development/create-a-proper-game-loop-with-requestanimationframe)
- [Game Dev Without An Engine: 2025/2026 Renaissance](https://www.sitepoint.com/game-dev-without-an-engine-the-2025-2026-renaissance/)
- [NES.css: NES-style CSS framework](https://github.com/nostalgic-css/NES.css/)
- [snes.css: SNES-themed CSS framework](https://github.com/devMiguelCarrero/snes.css/)
