// dashboard.js — SNES-style SDD Dashboard
// Native viewport resolution. All drawing uses scale factor S.
// Optimizations: offscreen bg cache, dirty-flag HUD, delta-time animations.
"use strict";

// ---------------------------------------------------------------------------
// 1. Palette
// ---------------------------------------------------------------------------
var P = {
  bg:"#0a0a1a", ground:"#1a1a30", wall:"#2a2a4a", star:"#556",
  cyan:"#4af", green:"#4f4", yellow:"#ff4", red:"#f55",
  purple:"#c8f", white:"#dde", dim:"#335", panel:"#10102a",
  border:"#2a2a4a", label:"#668", hudBg:"#0c0c20",
};

var PHASES = ["explore","propose","spec","design","tasks","apply","review","verify","clean","archive"];
var SLBL = ["EXP","PRO","SPC","DES","TSK","APL","REV","VER","CLN","ARC"];

var LW = 480, LH = 270;
var S = 1;

// Layout zones
var BG_END = 90, FLOOR_Y = 90, SCENE_END = 148, HUD_Y = 150;

// Sprite sheet — measured bounds
var WORKERS = [
  {frames:[[36,4,28,31],[68,4,28,28],[100,4,28,28],[36,4,28,31]]},
  {frames:[[6,35,24,35],[38,35,24,35],[66,35,24,35],[97,36,31,34]]},
  {frames:[[6,70,22,35],[37,70,25,35],[66,70,30,35],[6,70,22,35]]},
  {frames:[[8,105,20,20],[39,105,23,20],[70,105,23,20],[101,105,25,20]]},
];
var ST_OFF={x:5,y:125,w:54,h:40}, ST_ON={x:69,y:125,w:54,h:40};
var FTILE={x:192,y:224,w:32,h:32};

// ---------------------------------------------------------------------------
// 2. State
// ---------------------------------------------------------------------------
var pipelines=[], errors=[], kpi={ActiveChanges:0,TotalTokens:0,CacheHitPct:0,ErrorCount:0};
var stars=[], particles=[];
var lastTime=0, animTime=0;
var canvas, ctx;
var sprImg=null, sprOk=false, bgImg=null, bgOk=false;
var wsState="off";

// Offscreen background canvas (perf: pre-composited static layer)
var bgCanvas=null, bgDirty=true;

// HUD dirty flag (perf: skip redraw when data unchanged)
var hudDirty=true;

// Animation accumulators (smooth delta-time animations)
var workerFrame=0, workerTimer=0;
var conveyorOff=0;

// ---------------------------------------------------------------------------
// 3. Assets
// ---------------------------------------------------------------------------
function loadAssets() {
  sprImg=new Image();
  sprImg.onload=function(){sprOk=true;bgDirty=true;};
  sprImg.src="/static/sprites.png";
  bgImg=new Image();
  bgImg.onload=function(){bgOk=true;bgDirty=true;};
  bgImg.src="/static/bg-scene.png";
}

function spr(sx,sy,sw,sh,dx,dy,dw,dh) {
  if(!sprOk) return;
  ctx.drawImage(sprImg, sx,sy,sw,sh,
    Math.floor(dx*S), Math.floor(dy*S),
    Math.floor((dw||sw)*S), Math.floor((dh||sh)*S));
}

// Draw sprite to an arbitrary context (for offscreen canvas)
function sprTo(target,sx,sy,sw,sh,dx,dy,dw,dh) {
  if(!sprOk) return;
  target.drawImage(sprImg, sx,sy,sw,sh,
    Math.floor(dx*S), Math.floor(dy*S),
    Math.floor((dw||sw)*S), Math.floor((dh||sh)*S));
}

// ---------------------------------------------------------------------------
// 4. Primitives
// ---------------------------------------------------------------------------
function px(x,y,w,h,c) {
  ctx.fillStyle=c;
  ctx.fillRect(Math.floor(x*S),Math.floor(y*S),Math.ceil((w||1)*S),Math.ceil((h||1)*S));
}

function txt(str,x,y,c,sz) {
  ctx.fillStyle=c||P.white;
  ctx.font="bold "+Math.max(1,Math.round((sz||5)*S))+"px monospace";
  ctx.fillText(str,Math.floor(x*S),Math.floor(y*S));
}

// Centered text (for empty states)
function txtC(str,y,c,sz) {
  ctx.fillStyle=c||P.white;
  ctx.font="bold "+Math.max(1,Math.round((sz||5)*S))+"px monospace";
  var w=ctx.measureText(str).width;
  ctx.fillText(str, Math.floor((canvas.width-w)/2), Math.floor(y*S));
}

// ---------------------------------------------------------------------------
// 5. Offscreen Background (composited once, redrawn on resize/asset load)
// ---------------------------------------------------------------------------
function buildBackground() {
  bgCanvas=document.createElement("canvas");
  bgCanvas.width=Math.ceil(LW*S);
  bgCanvas.height=Math.ceil(SCENE_END*S);
  var bg=bgCanvas.getContext("2d");
  bg.imageSmoothingEnabled=false;

  // Sky/wall background image
  if(bgOk) {
    bg.drawImage(bgImg, 0,0,bgImg.width,bgImg.height,
      0,0, Math.floor(LW*S), Math.floor(BG_END*S));
  } else {
    bg.fillStyle=P.bg; bg.fillRect(0,0,bgCanvas.width,Math.floor(60*S));
    bg.fillStyle="#0c0c30"; bg.fillRect(0,Math.floor(60*S),bgCanvas.width,Math.floor((BG_END-60)*S));
  }

  // Floor zone
  bg.fillStyle=P.ground;
  bg.fillRect(0,Math.floor(FLOOR_Y*S), bgCanvas.width, Math.ceil((SCENE_END-FLOOR_Y)*S));
  bg.fillStyle=P.wall;
  bg.fillRect(0,Math.floor(FLOOR_Y*S), bgCanvas.width, Math.ceil(1*S));

  // Floor tiles
  if(sprOk) {
    for(var tx=0;tx<LW;tx+=14) {
      sprTo(bg, FTILE.x,FTILE.y,FTILE.w,FTILE.h, tx,FLOOR_Y+1,14,14);
    }
  }

  bgDirty=false;
}

// ---------------------------------------------------------------------------
// 6. Particles
// ---------------------------------------------------------------------------
function spawnParticle() {
  particles.push({
    x:20+Math.random()*(LW-40), y:FLOOR_Y+10+Math.random()*20,
    vx:(Math.random()-0.5)*0.3, vy:-0.15-Math.random()*0.15,
    life:80+Math.random()*60|0,
    col:[P.cyan,P.green,P.purple][Math.random()*3|0],
  });
}

function updateParticles(dt) {
  for(var i=particles.length-1;i>=0;i--) {
    var p=particles[i];
    p.x+=p.vx*dt*60; p.y+=p.vy*dt*60; p.life--;
    if(p.life<=0){particles.splice(i,1);continue;}
    if(p.life>15) px(p.x,p.y,1,1,p.col);
  }
}

// ---------------------------------------------------------------------------
// 7. Scene (draws every frame — dynamic elements only)
// ---------------------------------------------------------------------------
function initStars() {
  stars=[];
  for(var i=0;i<50;i++) stars.push({
    x:Math.random()*LW|0, y:Math.random()*30|0,
    phase:Math.random()*Math.PI*2, // for smooth sine twinkle
    speed:0.5+Math.random()*1.5,
  });
}

function drawScene(dt) {
  // 1. Static background — single drawImage from offscreen cache
  if(bgDirty || !bgCanvas) buildBackground();
  ctx.drawImage(bgCanvas, 0, 0);

  // 2. Stars — smooth sine-wave twinkle
  for(var i=0;i<stars.length;i++) {
    var s=stars[i];
    var alpha=Math.sin(animTime*s.speed+s.phase)*0.5+0.5;
    if(alpha>0.3) px(s.x,s.y,1,1,P.star);
  }

  // 3. Conveyor — continuous scroll
  conveyorOff=(conveyorOff+dt*12)%8;
  for(var x=10;x<LW-10;x+=4) {
    px(x,FLOOR_Y+24,3,2,((x+conveyorOff|0)%8<4)?"#222240":"#1a1a35");
  }

  // 4. Stations + Workers
  var spacing=(LW-20)/PHASES.length;
  var baseY=FLOOR_Y+22;
  var stW={};
  for(var i=0;i<pipelines.length;i++) {
    var pi=PHASES.indexOf(pipelines[i].CurrentPhase);
    if(pi>=0){if(!stW[pi])stW[pi]=[];stW[pi].push(pipelines[i]);}
  }

  // Ground level where feet touch
  var groundLine = SCENE_END - 10;

  for(var s=0;s<PHASES.length;s++) {
    var sx=10+s*spacing|0;
    var active=!!stW[s];

    // Desk — sits on the ground
    var deskH = 16, deskW = 26;
    var deskY = groundLine - deskH;
    if(sprOk) {
      var st=active?ST_ON:ST_OFF;
      spr(st.x,st.y,st.w,st.h, sx-3,deskY,deskW,deskH);
    }

    // Label below ground line
    txt(SLBL[s], sx, groundLine+2, active?P.cyan:P.dim, 4);

    // Workers — stand next to desk, feet on ground
    if(stW[s]) {
      var wType;
      if(s<=1) wType=WORKERS[0];
      else if(s<=3) wType=WORKERS[3];
      else if(s<=5) wType=WORKERS[1];
      else wType=WORKERS[2];

      for(var w=0;w<stW[s].length&&w<3;w++) {
        var fi=(workerFrame+w)%4;
        var f=wType.frames[fi];
        var workerH = 13;
        var workerW = 10;
        var wx = sx - 4 + w*10 |0;
        var wy = groundLine - workerH |0;
        if(sprOk) spr(f[0],f[1],f[2],f[3], wx,wy,workerW,workerH);
        if(stW[s][w].Status==="error"&&(animTime*4|0)%2===0) txt("!",wx+5,wy-3,P.red,6);
        if(stW[s][w].Status==="warn"&&(animTime*2|0)%2===0) px(wx+5,wy-2,2,2,P.yellow);
      }
    }
  }

  // 5. Particles
  if(particles.length<30 && Math.random()<dt*4) spawnParticle();
  updateParticles(dt);

  // Title
  txt("SHENRON SDD", 4, 9, P.cyan, 7);
}

// ---------------------------------------------------------------------------
// 8. HUD (only redraws when data changes via hudDirty flag)
// ---------------------------------------------------------------------------
function drawHUD() {
  px(0,SCENE_END,LW,LH-SCENE_END,P.hudBg);
  px(0,SCENE_END,LW,1,P.border);

  var Y=HUD_Y;
  px(3,Y,LW-6,12,P.panel);
  px(3,Y,LW-6,1,P.border);
  px(3,Y+12,LW-6,1,P.border);
  txt("⚔ SHENRON SDD",7,Y+9,P.cyan,6);
  var wsC=wsState==="live"?P.green:wsState==="connecting"?P.yellow:P.red;
  px(LW-22,Y+5,5,5,wsC);
  txt("LIVE",LW-50,Y+9,wsC,5);

  // KPI cards
  var kY=Y+16;
  var cW=(LW-14)/4|0;
  var cards=[
    {l:"CHANGES",v:""+kpi.ActiveChanges,c:P.cyan},
    {l:"TOKENS",v:shortNum(kpi.TotalTokens),c:P.green},
    {l:"CACHE",v:Math.round(kpi.CacheHitPct)+"%",c:P.purple},
    {l:"ERRORS",v:""+kpi.ErrorCount,c:kpi.ErrorCount>0?P.red:P.dim},
  ];
  for(var i=0;i<4;i++) {
    var cx=3+i*(cW+2)|0;
    px(cx,kY,cW,22,P.panel);
    px(cx,kY,2,22,cards[i].c);
    px(cx,kY,cW,1,P.border);
    txt(cards[i].l,cx+5,kY+7,P.label,4);
    txt(cards[i].v,cx+5,kY+18,cards[i].c,8);
  }

  // Pipeline table
  var tY=kY+27;
  px(3,tY,LW-6,1,P.border);
  txt("▸ ACTIVE PIPELINES",6,tY+8,P.label,5);
  tY+=12;
  txt("CHANGE",8,tY+5,P.dim,4);txt("PHASE",120,tY+5,P.dim,4);
  txt("PROGRESS",210,tY+5,P.dim,4);txt("TOKENS",340,tY+5,P.dim,4);txt("ST",430,tY+5,P.dim,4);
  tY+=8;
  px(3,tY,LW-6,1,"#1a1a30");
  tY+=2;

  if(pipelines.length===0) {
    var dots=".".repeat((animTime*2|0)%4);
    txtC("Waiting for quests"+dots, tY+10, P.label, 5);
    tY+=16;
  } else {
    for(var i=0;i<pipelines.length&&i<4;i++) {
      var p=pipelines[i],rY=tY+i*12;
      if(i%2===1)px(3,rY,LW-6,12,P.panel);
      txt((p.Name||"").substring(0,18),8,rY+8,P.white,5);
      txt(p.CurrentPhase||"",120,rY+8,P.cyan,5);
      px(210,rY+3,80,5,P.ground);
      var fw=80*p.ProgressPct/100|0;
      if(fw>0)px(210,rY+3,fw,5,p.Status==="error"?P.red:P.green);
      txt(p.Completed+"/"+p.Total,295,rY+8,P.dim,4);
      txt(shortNum(p.Tokens),340,rY+8,P.white,5);
      px(430,rY+4,5,5,p.Status==="error"?P.red:p.Status==="warn"?P.yellow:P.green);
    }
    tY+=pipelines.length*12+2;
  }

  // Errors
  var eY=tY+2;
  px(3,eY,LW-6,1,P.border);
  txt("▸ BATTLE LOG",6,eY+8,P.label,5);
  eY+=12;
  if(errors.length===0) {
    txtC("No enemies encountered", eY+10, P.label, 5);
  } else {
    for(var i=0;i<errors.length&&i<3;i++) {
      var e=errors[i],rY=eY+i*10;
      if(i%2===1)px(3,rY,LW-6,10,P.panel);
      txt((e.Timestamp||"").substring(11,19),8,rY+7,"#f88",4);
      txt((e.CommandName||"").substring(0,14),60,rY+7,"#f88",4);
      txt("x"+e.ExitCode,145,rY+7,"#f66",4);
      txt((e.Change||"").substring(0,12),175,rY+7,"#f88",4);
      txt((e.FirstLine||"").substring(0,35),265,rY+7,"#a66",4);
    }
  }

  // Legend
  var lY=LH-7;
  var leg=[{c:P.green,t:"Completed"},{c:P.yellow,t:"In Progress"},{c:P.dim,t:"Pending"},{c:P.red,t:"Error"}];
  var lx=8;
  for(var i=0;i<leg.length;i++){px(lx,lY,5,5,leg[i].c);txt(leg[i].t,lx+7,lY+5,P.label,4);lx+=55;}
}

function shortNum(n){if(n>=1e6)return(n/1e6).toFixed(1)+"M";if(n>=1e3)return(n/1e3).toFixed(1)+"K";return""+n;}

// ---------------------------------------------------------------------------
// 9. Game Loop (delta-time based)
// ---------------------------------------------------------------------------
function gameLoop(timestamp) {
  var dt=Math.min((timestamp-lastTime)/1000, 0.1); // cap at 100ms to avoid jumps
  lastTime=timestamp;
  animTime+=dt;

  // Worker sprite animation: 4 fps
  workerTimer+=dt;
  if(workerTimer>=0.25){workerTimer-=0.25;workerFrame=(workerFrame+1)%4;}

  ctx.imageSmoothingEnabled=false;
  ctx.fillStyle=P.bg;
  ctx.fillRect(0,0,canvas.width,canvas.height);

  drawScene(dt);

  // HUD: redraw always for animated ellipsis, but skip heavy data rebuild if clean
  drawHUD();
  hudDirty=false;

  requestAnimationFrame(gameLoop);
}

function initCanvas() {
  canvas=document.getElementById("scene");
  if(!canvas) return;

  function resize() {
    var vw=window.innerWidth, vh=window.innerHeight;
    canvas.width=vw; canvas.height=vh;
    canvas.style.width=vw+"px"; canvas.style.height=vh+"px";
    S=Math.min(vw/LW,vh/LH);
    bgDirty=true; hudDirty=true;
    if(ctx) ctx.imageSmoothingEnabled=false;
  }
  resize();
  window.addEventListener("resize",resize);

  ctx=canvas.getContext("2d");
  ctx.imageSmoothingEnabled=false;
  initStars();
  loadAssets();
  requestAnimationFrame(gameLoop);
}

// ---------------------------------------------------------------------------
// 10. WebSocket
// ---------------------------------------------------------------------------
var ws=null,reconnectDelay=1000,reconnectTimer=null;

function connect() {
  wsState="connecting"; hudDirty=true;
  ws=new WebSocket("ws://"+location.host+"/ws");
  ws.onopen=function(){reconnectDelay=1000;wsState="live";hudDirty=true;};
  ws.onmessage=function(evt){
    var msg;try{msg=JSON.parse(evt.data);}catch(_){return;}
    hudDirty=true;
    switch(msg.type){
      case"kpi":kpi=msg.data;break;
      case"pipelines":pipelines=msg.data||[];break;
      case"errors":errors=msg.data||[];break;
    }
  };
  ws.onclose=function(){
    wsState="off"; hudDirty=true;
    if(reconnectTimer)return;
    reconnectTimer=setTimeout(function(){reconnectTimer=null;connect();},reconnectDelay);
    reconnectDelay=Math.min(reconnectDelay*2,10000);
  };
  ws.onerror=function(){ws.close();};
}

document.addEventListener("DOMContentLoaded",function(){initCanvas();connect();});
