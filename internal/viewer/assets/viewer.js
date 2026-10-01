// The viewer page `browserscale view` serves. It is the third StreamTransport
// adapter alongside the web panel and the widget's own docs: the widget core
// does the WebRTC and the coordinate maths, and everything session-shaped is
// reached by asking the CLI process over loopback. No API key is ever in this
// page — the CLI holds it and signs the calls itself, the DevTools pane's
// included.

import {
  getOrCreateSharedConnection,
  releaseSharedConnection,
  toViewportCoords,
  domModifiers,
  sendJSON,
} from "./widget.js";

const { sessionId, token, interactive } = window.__BROWSERSCALE_VIEW;

const stage = document.getElementById("stage");
const video = document.getElementById("video");
const dot = document.getElementById("dot");
const hint = document.getElementById("hint");
const overlay = document.getElementById("overlay");
const overlayTitle = document.getElementById("overlay-title");
const overlayDetail = document.getElementById("overlay-detail");

/** How long a live transport may stay silent before that counts as a failure. */
const MEDIA_TIMEOUT_MS = 10000;

function setStatus(status, title, detail) {
  dot.dataset.status = status;
  if (status === "connected") {
    overlay.hidden = true;
    return;
  }
  overlay.hidden = false;
  overlayTitle.textContent = title;
  overlayDetail.textContent = detail || "";
}

// --- Transport: every method is one call into the CLI process ---------------

async function call(path, body) {
  // The token travels as a custom header on purpose. A page on another origin
  // cannot set one without a CORS preflight, which this server does not answer,
  // so a random website cannot drive the session even though it can reach
  // 127.0.0.1.
  const res = await fetch(path, {
    method: "POST",
    headers: { "content-type": "application/json", "x-viewer-token": token },
    body: JSON.stringify(body || {}),
  });
  if (!res.ok) {
    throw new Error((await res.text()).trim() || res.statusText);
  }
  return res.json();
}

const transport = {
  getIceServers: async () => (await call("/api/ice")).iceServers,
  startStream: async (offerSdp) => {
    const r = await call("/api/start", { offerSdp });
    return { answerSdp: r.answerSdp, viewport: r.viewport };
  },
  stopStream: async () => { await call("/api/stop"); },
  insertText: async (text) => { await call("/api/insert-text", { text }); },
  getSelection: async () => (await call("/api/selection")).text,
};

// --- Connection ------------------------------------------------------------

const conn = getOrCreateSharedConnection(sessionId, transport);
conn.refs++;

let viewport = conn.viewport;
conn.subscribeViewport((v) => { viewport = v; fitWindow(v); });

let mediaTimer = 0;

// "connected" means a frame has been painted, not that a track object exists:
// ontrack fires on setRemoteDescription, before a single packet has moved, so
// reporting it there makes an unreachable relay look like a working stream with
// a black picture.
video.addEventListener("loadeddata", () => {
  if (mediaTimer) { clearTimeout(mediaTimer); mediaTimer = 0; }
  setStatus("connected");
});

conn.pc.addEventListener("track", (e) => {
  const rx = e.receiver;
  if (rx) {
    if ("jitterBufferTarget" in rx) rx.jitterBufferTarget = 0;
    if ("playoutDelayHint" in rx) rx.playoutDelayHint = 0;
  }
  const stream = e.streams[0] || null;
  if (stream) video.srcObject = stream;
});

conn.pc.addEventListener("connectionstatechange", () => {
  const state = conn.pc.connectionState;
  if (state === "failed") {
    setStatus("error", "Connection failed", "The relay could not be reached.");
  } else if (state === "closed" || state === "disconnected") {
    setStatus("disconnected", "Disconnected", "");
  } else if (state === "connected" && !mediaTimer) {
    // The transport is up, so whatever is still missing is media. Say so
    // instead of sitting on a black picture forever.
    mediaTimer = setTimeout(() => {
      mediaTimer = 0;
      if (video.readyState >= 2) return;
      setStatus("error", "Connected, but no video is arriving", "");
    }, MEDIA_TIMEOUT_MS);
  }
});

// --- Window shape ---------------------------------------------------------

// The window is shaped from in here rather than at launch: the size asked for on
// Chromium's command line is ignored whenever a browser is already running,
// because that instance is the one that opens the window.
const mayFit = new URLSearchParams(location.search).get("fit") === "1";
let fitted = false;

/** Give the window the session's own size, once, if this window is ours. */
function fitWindow(v) {
  if (!mayFit || fitted || !v || !v.w || !v.h) return;
  fitted = true;
  // The extra height covers our header and the window's title bar. The browser
  // clamps this to the screen, so a session larger than the display is fine.
  const pane = document.getElementById("devtools");
  window.resizeTo(v.w + (pane.hidden ? 0 : pane.offsetWidth), v.h + 50);
}

setStatus("connecting", "Connecting…", "");
conn.ready.then(
  () => {
    if (conn.stream) video.srcObject = conn.stream;
    fitWindow(conn.viewport);
    if (interactive) stage.focus();
  },
  (err) => setStatus("error", "Could not start the stream", String(err && err.message || err)),
);

// The tab closing is the ordinary way this viewer ends, and it is the one
// moment the CLI cannot observe for us, so release here rather than relying on
// the process noticing.
window.addEventListener("pagehide", () => {
  conn.refs = 0;
  releaseSharedConnection(sessionId);
});

// --- Input ----------------------------------------------------------------

if (!interactive) {
  hint.textContent = "Read-only";
} else {
  // Buttons held down on the video, so a release anywhere can be matched to
  // one, and the last point inside it for releases that land outside.
  let heldButtons = 0;
  let lastPoint = null;
  let pendingMove = null;
  let rafId = 0;

  const at = (e) => toViewportCoords(e, video, viewport);

  stage.addEventListener("contextmenu", (e) => e.preventDefault());

  video.addEventListener("mousemove", (e) => {
    if (!viewport) return;
    const pt = at(e);
    if (!pt) return;
    lastPoint = pt;
    pendingMove = { type: "mousemove", x: pt.x, y: pt.y, buttons: e.buttons, modifiers: domModifiers(e) };
    // One move per frame, latest wins: the unreliable channel is fine here
    // precisely because the next move supersedes this one.
    if (!rafId) {
      rafId = requestAnimationFrame(() => {
        rafId = 0;
        sendJSON(conn.pointerDc, pendingMove);
        pendingMove = null;
      });
    }
  });

  video.addEventListener("mousedown", (e) => {
    if (!viewport) return;
    const pt = at(e);
    if (!pt) return;
    lastPoint = pt;
    heldButtons = e.buttons;
    stage.focus();
    // Reliable channel: a click is a discrete action, and a dropped one simply
    // never happens. Only moves can afford to be lost.
    sendJSON(conn.inputDc, {
      type: "mousedown",
      x: pt.x, y: pt.y,
      button: e.button, buttons: e.buttons,
      detail: e.detail || 1,
      modifiers: domModifiers(e),
    });
  });

  // Releases are watched on the window, not the video. A press that starts on
  // the picture can end anywhere — over the header, past the window edge — and
  // if that release is never sent the remote button stays down for good, so
  // every later move reads as a drag.
  window.addEventListener("mouseup", (e) => {
    const bit = e.button === 2 ? 2 : e.button === 1 ? 4 : 1;
    if (!(heldButtons & bit)) return;
    heldButtons &= ~bit;
    const pt = (viewport && at(e)) || lastPoint;
    if (!pt) return;
    lastPoint = pt;
    sendJSON(conn.inputDc, {
      type: "mouseup",
      x: pt.x, y: pt.y,
      button: e.button, buttons: heldButtons,
      detail: e.detail || 1,
      modifiers: domModifiers(e),
    });
  });

  video.addEventListener("wheel", (e) => {
    if (!viewport) return;
    e.preventDefault();
    const pt = at(e);
    if (!pt) return;
    // Reliable: a dropped scroll leaves the page at the wrong offset with
    // nothing to correct it, unlike a dropped move.
    sendJSON(conn.inputDc, {
      type: "wheel",
      x: pt.x, y: pt.y,
      deltaX: Math.round(e.deltaX), deltaY: Math.round(e.deltaY),
      modifiers: domModifiers(e),
    });
  }, { passive: false });

  stage.addEventListener("keydown", (e) => {
    if (e.repeat) return;

    // Clipboard is bridged rather than forwarded: the remote page has no access
    // to this machine's clipboard, so paste reads here and inserts there.
    if ((e.ctrlKey || e.metaKey) && e.key === "v") {
      e.preventDefault();
      navigator.clipboard.readText()
        .then((text) => { if (text) return transport.insertText(text); })
        .catch(() => {});
      return;
    }
    if ((e.ctrlKey || e.metaKey) && e.key === "c") {
      e.preventDefault();
      transport.getSelection()
        .then((text) => { if (text) return navigator.clipboard.writeText(text); })
        .catch(() => {});
      return;
    }

    e.preventDefault();
    e.stopPropagation();
    sendJSON(conn.inputDc, { type: "keydown", key: e.key, code: e.code, modifiers: domModifiers(e) });
  });

  stage.addEventListener("keyup", (e) => {
    e.preventDefault();
    e.stopPropagation();
    if ((e.ctrlKey || e.metaKey) && (e.key === "v" || e.key === "c")) return;
    sendJSON(conn.inputDc, { type: "keyup", key: e.key, code: e.code, modifiers: domModifiers(e) });
  });
}

// --- DevTools ---------------------------------------------------------------

// The pane is browserscale-devtools' standalone bundle on the SDK's WebSocket
// transport. It loads on first open, so a plain `view` never pays for it.

const DT_OPEN_KEY = "bs.view.devtools.open";
const DT_WIDTH_KEY = "bs.view.devtools.width";
const DT_MIN_WIDTH = 340;

const dtPane = document.getElementById("devtools");
const dtRoot = document.getElementById("devtools-root");
const dtToggle = document.getElementById("devtools-toggle");
const dtResize = document.getElementById("devtools-resize");
const inspectBtn = document.getElementById("inspect");

const store = {
  get(key) {
    try { return localStorage.getItem(key); } catch { return null; }
  },
  set(key, value) {
    try { localStorage.setItem(key, value); } catch { /* storage unavailable */ }
  },
};

let devtools = null;
let dtTab = "elements";
let picking = false;

function loadDevTools() {
  if (devtools) return devtools;
  if (!document.querySelector('link[href="devtools.css"]')) {
    const css = document.createElement("link");
    css.rel = "stylesheet";
    css.href = "devtools.css";
    document.head.appendChild(css);
  }
  devtools = import("./devtools.js").then((mod) => {
    // No API key here: the CLI writes it into every call on its side.
    const client = mod.createWebSocketBrowser(`ws://${location.host}/ws?t=${encodeURIComponent(token)}`, sessionId, "");
    const view = mod.mount(dtRoot, {
      client,
      active: true,
      tab: dtTab,
      onTab: (t) => {
        dtTab = t;
        view.update({ tab: t });
      },
      onClose: () => setDevTools(false),
      closeLabel: "Close DevTools (Ctrl+.)",
    });
    return { mod, client, view };
  });
  devtools.catch((err) => {
    devtools = null;
    const p = document.createElement("p");
    p.id = "devtools-error";
    p.textContent = `DevTools could not load: ${(err && err.message) || err}`;
    dtRoot.replaceChildren(p);
  });
  return devtools;
}

function setDevTools(open) {
  dtPane.hidden = !open;
  dtToggle.setAttribute("aria-pressed", String(open));
  inspectBtn.hidden = !open;
  if (!open) setPicking(false);
  store.set(DT_OPEN_KEY, open ? "1" : "0");
  if (open) loadDevTools();
}

function setPicking(on) {
  picking = on;
  inspectBtn.setAttribute("aria-pressed", String(on));
  if (on) stage.dataset.picking = "";
  else delete stage.dataset.picking;
}

dtToggle.addEventListener("click", () => setDevTools(dtPane.hidden));
inspectBtn.addEventListener("click", () => setPicking(!picking));

// Picking takes the click before the input forwarding sees it: the capture
// listener on the stage runs ahead of the ones on the video.
stage.addEventListener("mousedown", async (e) => {
  if (!picking || e.button !== 0) return;
  e.preventDefault();
  e.stopPropagation();
  setPicking(false);
  const pt = viewport && toViewportCoords(e, video, viewport);
  if (!pt) return;
  const { mod, client, view } = await loadDevTools();
  const target = await mod.inspectAt(client, pt.x, pt.y);
  if (!target) return;
  dtTab = "elements";
  view.update({ inspect: target, tab: "elements" });
}, true);

// Shortcuts are read on the window in the capture phase, before the stage
// forwards keys to the page.
function shortcut(e) {
  if (!(e.ctrlKey || e.metaKey)) return null;
  if (e.key === ".") return "toggle";
  if (e.shiftKey && e.key.toLowerCase() === "c" && !dtPane.hidden) return "pick";
  return null;
}
window.addEventListener("keydown", (e) => {
  const action = shortcut(e);
  if (!action) return;
  e.preventDefault();
  e.stopPropagation();
  if (e.repeat) return;
  if (action === "toggle") setDevTools(dtPane.hidden);
  else setPicking(!picking);
}, true);
window.addEventListener("keyup", (e) => {
  if (shortcut(e)) e.stopPropagation();
}, true);

const savedWidth = Number(store.get(DT_WIDTH_KEY));
if (savedWidth >= DT_MIN_WIDTH) dtPane.style.width = `${savedWidth}px`;

dtResize.addEventListener("pointerdown", (e) => {
  e.preventDefault();
  dtResize.setPointerCapture(e.pointerId);
  dtResize.dataset.dragging = "";
  const right = dtPane.getBoundingClientRect().right;
  const move = (ev) => {
    const w = Math.max(DT_MIN_WIDTH, Math.min(right - ev.clientX, window.innerWidth * 0.75));
    dtPane.style.width = `${Math.round(w)}px`;
  };
  const up = () => {
    dtResize.removeEventListener("pointermove", move);
    dtResize.removeEventListener("pointerup", up);
    dtResize.removeEventListener("pointercancel", up);
    delete dtResize.dataset.dragging;
    store.set(DT_WIDTH_KEY, String(Math.round(dtPane.getBoundingClientRect().width)));
  };
  dtResize.addEventListener("pointermove", move);
  dtResize.addEventListener("pointerup", up);
  dtResize.addEventListener("pointercancel", up);
});

setDevTools(store.get(DT_OPEN_KEY) === "1");

// --- Fullscreen -----------------------------------------------------------

document.getElementById("fullscreen").addEventListener("click", async () => {
  try {
    if (document.fullscreenElement) await document.exitFullscreen();
    else await stage.requestFullscreen();
    if (interactive) stage.focus();
  } catch (err) {
    console.warn("Fullscreen request failed:", err);
  }
});
