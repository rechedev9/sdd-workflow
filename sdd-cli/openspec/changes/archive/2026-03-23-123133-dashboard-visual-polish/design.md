# Design: dashboard-visual-polish

## Architecture

All changes are in `dashboard.js`. The rendering pipeline becomes:

```
Asset Load → Build offscreen bgCanvas
                    ↓
Game Loop (60fps):
  1. ctx.drawImage(bgCanvas, 0, 0)     ← static layer (1 call)
  2. drawConveyorAnim(dt)               ← conveyor scroll overlay
  3. drawStations(dt)                   ← stations + workers
  4. drawParticles(dt)                  ← ambient particles
  5. if(hudDirty) drawHUD(); hudDirty=false  ← conditional HUD
```

## D1: Offscreen Background Canvas

```javascript
var bgCanvas = null;

function buildBackground() {
  bgCanvas = document.createElement('canvas');
  bgCanvas.width = Math.ceil(LW * S);
  bgCanvas.height = Math.ceil(SCENE_END * S);
  var bgCtx = bgCanvas.getContext('2d');
  bgCtx.imageSmoothingEnabled = false;

  // Draw bg image
  if(bgOk) bgCtx.drawImage(bgImg, 0,0,bgImg.width,bgImg.height, 0,0, LW*S, BG_END*S);

  // Draw floor
  // ... fill + tile sprites

  // Draw static conveyor base
  // ... belt pattern without animation offset
}
```

Called on: `bgImg.onload`, `sprImg.onload`, `window.resize`.

## D2: HUD Dirty Flag

```javascript
var hudDirty = true;

// In ws.onmessage:
hudDirty = true;

// In gameLoop:
if(hudDirty) { drawHUD(); hudDirty = false; }
```

The HUD area is only cleared and redrawn when data changes. Between updates, the last drawn HUD persists on canvas.

## D3: Animation Accumulators

```javascript
var workerAnimTimer = 0;
var workerAnimFrame = 0;
var conveyorOffset = 0;

function updateAnimations(dt) {
  // Worker sprites: 4 fps
  workerAnimTimer += dt;
  if(workerAnimTimer >= 0.25) {
    workerAnimTimer -= 0.25;
    workerAnimFrame = (workerAnimFrame + 1) % 4;
  }

  // Conveyor: continuous scroll
  conveyorOffset = (conveyorOffset + dt * 8) % 8;

  // Stars: each star has its own phase, uses sin(time + phase)
}
```

## D4: Empty State Polish

```javascript
function drawCentered(str, y, color, size) {
  ctx.font = "bold " + Math.round(size * S) + "px monospace";
  var w = ctx.measureText(str).width;
  ctx.fillStyle = color;
  ctx.fillText(str, Math.floor((canvas.width - w) / 2), Math.floor(y * S));
}
```

Ellipsis animation: `"Waiting for quests" + ".".repeat(Math.floor(animTime * 2) % 4)`
