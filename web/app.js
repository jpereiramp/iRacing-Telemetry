const cfg = {
  maxMph: 220,
  maxRpm: 10000,
  trendMaxSamples: 120,
  gauge: {
    minAngle: (-100 * Math.PI) / 180,
    maxAngle: (100 * Math.PI) / 180,
  },
};

const el = {
  speedValue: document.getElementById("speedValue"),
  speedKph: document.getElementById("speedKph"),
  sourceState: document.getElementById("sourceState"),
  rpmValue: document.getElementById("rpmValue"),
  rpmBar: document.getElementById("rpmBar"),
  gearValue: document.getElementById("gearValue"),
  throttlePct: document.getElementById("throttlePct"),
  brakePct: document.getElementById("brakePct"),
  clutchPct: document.getElementById("clutchPct"),
  throttleBar: document.getElementById("throttleBar"),
  brakeBar: document.getElementById("brakeBar"),
  clutchBar: document.getElementById("clutchBar"),
  gaugeCanvas: document.getElementById("gauge"),
  trendCanvas: document.getElementById("trend"),
};

const ctx = {
  gauge: el.gaugeCanvas.getContext("2d"),
  trend: el.trendCanvas.getContext("2d"),
};

const state = {
  isLive: false,
  currentMph: 0,
  targetMph: 0,
  rpm: 0,
  gear: 0,
  throttle: 0,
  brake: 0,
  clutch: 0,
  trend: [],
};

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

function drawGauge(mph) {
  const w = el.gaugeCanvas.width;
  const h = el.gaugeCanvas.height;
  ctx.gauge.clearRect(0, 0, w, h);

  const cx = w / 2;
  const cy = h * 0.9;
  const radius = Math.min(w * 0.4, h * 0.82);
  const value = Number.isFinite(mph) ? mph : 0;
  const progress = clamp(value / cfg.maxMph, 0, 1);
  const minAngle = cfg.gauge.minAngle;
  const maxAngle = cfg.gauge.maxAngle;
  const angle = minAngle + (maxAngle - minAngle) * progress;

  ctx.gauge.lineCap = "round";

  // Tick marks only (no arc track).
  for (let i = 0; i <= 12; i++) {
    const tick = minAngle + ((maxAngle - minAngle) * i) / 12;
    const r1 = radius - 24;
    const r2 = radius - 9;
    ctx.gauge.beginPath();
    ctx.gauge.lineWidth = 2;
    ctx.gauge.strokeStyle = "rgba(255,255,255,0.30)";
    ctx.gauge.moveTo(cx + Math.sin(tick) * r1, cy - Math.cos(tick) * r1);
    ctx.gauge.lineTo(cx + Math.sin(tick) * r2, cy - Math.cos(tick) * r2);
    ctx.gauge.stroke();
  }

  // Needle points to current speed; geometry is kept simple for readability.
  ctx.gauge.save();
  ctx.gauge.translate(cx, cy);
  ctx.gauge.rotate(angle);
  ctx.gauge.beginPath();
  ctx.gauge.fillStyle = "#4cd4ff";
  ctx.gauge.moveTo(-5, 8);
  ctx.gauge.lineTo(5, 8);
  ctx.gauge.lineTo(0, -radius + 30);
  ctx.gauge.closePath();
  ctx.gauge.fill();
  ctx.gauge.restore();

  ctx.gauge.beginPath();
  ctx.gauge.fillStyle = "#eaf0ff";
  ctx.gauge.arc(cx, cy, 8, 0, Math.PI * 2);
  ctx.gauge.fill();
}

function drawTrend() {
  const w = el.trendCanvas.width;
  const h = el.trendCanvas.height;
  ctx.trend.clearRect(0, 0, w, h);
  if (state.trend.length < 2) return;

  const max = Math.max(10, ...state.trend);
  const min = Math.max(0, Math.min(...state.trend) - 5);
  const span = Math.max(1, max - min);

  ctx.trend.beginPath();
  state.trend.forEach((v, i) => {
    const x = (i / (state.trend.length - 1)) * w;
    const y = h - ((v - min) / span) * (h - 8) - 4;
    if (i === 0) ctx.trend.moveTo(x, y);
    else ctx.trend.lineTo(x, y);
  });
  ctx.trend.lineWidth = 2.2;
  ctx.trend.strokeStyle = "#4cd4ff";
  ctx.trend.stroke();
}

function updateUI(sample) {
  state.isLive = sample.source === "live";
  state.targetMph = state.isLive ? clamp(sample.speedMph || 0, 0, cfg.maxMph) : 0;
  state.rpm = state.isLive ? clamp(sample.rpm || 0, 0, cfg.maxRpm) : 0;
  state.gear = state.isLive ? Number(sample.gear || 0) : 0;
  state.throttle = state.isLive ? clamp(sample.throttle || 0, 0, 1) : 0;
  state.brake = state.isLive ? clamp(sample.brake || 0, 0, 1) : 0;
  state.clutch = state.isLive ? clamp(sample.clutch || 0, 0, 1) : 0;

  el.sourceState.textContent = state.isLive ? "live" : "waiting for iRacing";
  el.sourceState.className = state.isLive ? "state live" : "state";
  el.speedKph.textContent = state.isLive ? `${(sample.speedKph || 0).toFixed(1)} km/h` : "-- km/h";
  el.rpmValue.textContent = state.isLive ? state.rpm.toFixed(0) : "--";
  el.rpmBar.style.width = `${(state.rpm / cfg.maxRpm) * 100}%`;
  el.gearValue.textContent = state.isLive ? gearToLabel(state.gear) : "N";
  setPedal(el.throttlePct, el.throttleBar, state.throttle);
  setPedal(el.brakePct, el.brakeBar, state.brake);
  setPedal(el.clutchPct, el.clutchBar, state.clutch);

  if (!state.isLive) {
    state.trend = [];
  }
}

function animate() {
  state.currentMph += (state.targetMph - state.currentMph) * 0.22;
  el.speedValue.textContent = state.isLive ? state.currentMph.toFixed(1) : "--";

  if (state.isLive) {
    state.trend.push(state.currentMph);
    if (state.trend.length > cfg.trendMaxSamples) state.trend.shift();
  }
  drawGauge(state.currentMph);
  drawTrend();
  requestAnimationFrame(animate);
}

function gearToLabel(value) {
  if (value < 0) return "R";
  if (value === 0) return "N";
  return String(value);
}

function setPedal(labelEl, barEl, value) {
  const pct = Math.round(value * 100);
  labelEl.textContent = `${pct}%`;
  barEl.style.width = `${pct}%`;
}

function connectStream() {
  const stream = new EventSource("/api/stream");
  stream.addEventListener("telemetry", (evt) => {
    try {
      updateUI(JSON.parse(evt.data));
    } catch (_) {}
  });
  stream.onerror = () => {
    el.sourceState.textContent = "reconnecting...";
    el.sourceState.className = "state";
    state.isLive = false;
    state.targetMph = 0;
    state.trend = [];
  };
}

connectStream();
animate();
