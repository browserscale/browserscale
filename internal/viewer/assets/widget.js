// src/input.ts
function sendJSON(dc, msg) {
  if (!msg) return;
  if (!dc || dc.readyState !== "open") return;
  dc.send(JSON.stringify(msg));
}
function domModifiers(e) {
  return (e.altKey ? 1 : 0) | (e.ctrlKey ? 2 : 0) | (e.metaKey ? 4 : 0) | (e.shiftKey ? 8 : 0);
}

// src/geometry.ts
function toViewportCoords(e, video, viewportDims) {
  if (!video.videoWidth || !video.videoHeight) return null;
  if (!viewportDims.w || !viewportDims.h) return null;
  const rect = video.getBoundingClientRect();
  const scale = Math.min(rect.width / video.videoWidth, rect.height / video.videoHeight);
  const renderedW = video.videoWidth * scale;
  const renderedH = video.videoHeight * scale;
  const offsetX = (rect.width - renderedW) / 2;
  const offsetY = (rect.height - renderedH) / 2;
  const relX = e.clientX - rect.left - offsetX;
  const relY = e.clientY - rect.top - offsetY;
  if (relX < 0 || relY < 0 || relX > renderedW || relY > renderedH) return null;
  return {
    x: Math.round(relX / renderedW * viewportDims.w),
    y: Math.round(relY / renderedH * viewportDims.h)
  };
}

// src/connection.ts
var sharedConnections = /* @__PURE__ */ new Map();
function getOrCreateSharedConnection(sessionId, transport) {
  const existing = sharedConnections.get(sessionId);
  if (existing) {
    if (existing.stopTimer !== null) {
      window.clearTimeout(existing.stopTimer);
      existing.stopTimer = null;
    }
    return existing;
  }
  const pc = new RTCPeerConnection();
  const inputDc = pc.createDataChannel("input", { ordered: true });
  const pointerDc = pc.createDataChannel("pointer", {
    ordered: false,
    maxRetransmits: 0
  });
  pointerDc.binaryType = "arraybuffer";
  inputDc.binaryType = "arraybuffer";
  const viewportListeners = /* @__PURE__ */ new Set();
  const entry = {
    pc,
    inputDc,
    pointerDc,
    ready: Promise.resolve(),
    stream: null,
    refs: 0,
    stopTimer: null,
    transport,
    viewport: null,
    subscribeViewport(listener) {
      viewportListeners.add(listener);
      return () => {
        viewportListeners.delete(listener);
      };
    }
  };
  sharedConnections.set(sessionId, entry);
  const applyViewport = (viewport) => {
    if (!viewport || viewport.w <= 0 || viewport.h <= 0) return;
    if (entry.viewport?.w === viewport.w && entry.viewport?.h === viewport.h) return;
    entry.viewport = viewport;
    for (const listener of viewportListeners) listener(viewport);
  };
  inputDc.addEventListener("message", (e) => {
    if (typeof e.data !== "string") return;
    let msg;
    try {
      msg = JSON.parse(e.data);
    } catch {
      return;
    }
    const m = msg;
    if (m?.type !== "viewport") return;
    if (!Number.isFinite(m.width) || !Number.isFinite(m.height)) return;
    applyViewport({ w: Math.round(m.width), h: Math.round(m.height) });
  });
  pc.addEventListener("track", (e) => {
    entry.stream = e.streams[0] ?? null;
  });
  entry.ready = (async () => {
    const iceServers = await transport.getIceServers();
    pc.setConfiguration({
      iceServers,
      iceTransportPolicy: "relay"
    });
    const transceiver = pc.addTransceiver("video", { direction: "recvonly" });
    try {
      const caps = RTCRtpReceiver.getCapabilities("video");
      if (caps) {
        const h264 = caps.codecs.find((c) => c.mimeType.toLowerCase() === "video/h264");
        if (h264) transceiver.setCodecPreferences([h264]);
      }
    } catch (e) {
      console.warn("Codec preferences:", e);
    }
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    await waitForIceGatheringComplete(pc);
    const localDescription = pc.localDescription;
    if (!localDescription) {
      throw new Error("Failed to create local WebRTC offer");
    }
    const answer = await transport.startStream(localDescription.sdp);
    applyViewport(answer.viewport);
    await pc.setRemoteDescription({ type: "answer", sdp: answer.answerSdp });
  })().catch((error) => {
    sharedConnections.delete(sessionId);
    pc.close();
    throw error;
  });
  return entry;
}
function releaseSharedConnection(sessionId) {
  const entry = sharedConnections.get(sessionId);
  if (!entry) return;
  entry.refs = Math.max(0, entry.refs - 1);
  if (entry.refs > 0 || entry.stopTimer !== null) return;
  entry.stopTimer = window.setTimeout(() => {
    if (entry.refs > 0) return;
    entry.pc.close();
    sharedConnections.delete(sessionId);
    void entry.transport.stopStream?.().catch(() => {
    });
  }, 750);
}
function waitForIceGatheringComplete(pc) {
  if (pc.iceGatheringState === "complete") {
    return Promise.resolve();
  }
  return new Promise((resolve) => {
    const timeout = window.setTimeout(() => {
      pc.removeEventListener("icegatheringstatechange", done);
      resolve();
    }, 5e3);
    const done = () => {
      if (pc.iceGatheringState === "complete") {
        window.clearTimeout(timeout);
        pc.removeEventListener("icegatheringstatechange", done);
        resolve();
      }
    };
    pc.addEventListener("icegatheringstatechange", done);
  });
}

export { domModifiers, getOrCreateSharedConnection, releaseSharedConnection, sendJSON, toViewportCoords, waitForIceGatheringComplete };
//# sourceMappingURL=index.js.map
//# sourceMappingURL=index.js.map