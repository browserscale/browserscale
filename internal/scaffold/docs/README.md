# browserscale docs (offline copy)

Bundled with this module so an agent can read and grep the SDK docs without
a network call. Guides show both Go and TS snippets; the API reference here
is **Go only**. Prefer opening the specific file below over guessing an API.

## Files

- `introduction.md` — Docs: browserscale documentation: rent a real cloud Chromium session and drive it from Go or TypeScript. Quickstart, core concepts, guides, and the full SDK reference.
- `quickstart.md` — Quickstart: Install the browserscale Go or TypeScript SDK, rent a cloud Chromium session, and run your first script in under five minutes.
- `concepts.md` — Core concepts: Sessions, locators, frames, the flat frame model, and the action / wait / read loop that every browserscale script is built on.
- `guides/locators.md` — Targeting elements: CSS, JavaScript, node-handle and coordinate locators for picking elements on the page in the browserscale Go and TypeScript SDKs.
- `guides/waiting.md` — Waiting: Explicit waits, timeouts, and the arm-before-trigger pattern every browserscale script needs to stay deterministic.
- `guides/loading.md` — Page loading: Navigate, commit vs. load events, and the right signal to wait for before driving a page with browserscale.
- `guides/interaction.md` — Interacting with the page: Click, fill, hover, scroll, drag, select and key events — every input gesture browserscale's real cloud Chromium exposes, with human-like mouse movement.
- `guides/reading.md` — Reading the page: Read text content, attributes, the DOM, screenshots and accessibility observations from a live browserscale session.
- `guides/dom-mirror.md` — Live DOM mirror: Hold a browserscale page's DOM as one live tree across every frame, updated incrementally, reporting changes only inside the part you expanded.
- `guides/evaluation.md` — Evaluating JavaScript: Run JavaScript inside the page or a specific frame from your Go or TypeScript script and return typed values back.
- `guides/frames.md` — Frames & iframes: browserscale's flat frame model: same-origin and OOPIF frames are uniformly addressable, with offsets exposed on every frame.
- `guides/network.md` — Network: Inspect, intercept, and modify network traffic in a browserscale session — armed before the triggering action, then awaited.
- `guides/capture.md` — Capturing traffic: Stream every request a browserscale session completes — cross-process iframes, workers and each redirect hop included — without ever pausing the page.
- `guides/cookies.md` — Cookies, storage & sessions: Manage cookies, localStorage, session lifecycle, and fingerprint persistence via rent + Fingerprint() in browserscale.
- `guides/captchas.md` — Captchas: Passive anti-bot challenges, interactive captchas, and the SolveCaptcha integration for the ones that need a human solver.
- `guides/shadow-canvas.md` — Shadow DOM & canvas: Read across browser security boundaries in a browserscale session: pierce closed shadow roots with __wrc.shadow and read tainted cross-origin canvas pixels with ReadCanvas.
- `guides/agentic-coding.md` — Agentic coding: Drive browserscale from an AI coding agent: connect the hosted MCP server at mcp.browserscale.cloud/mcp with a Bearer API key, then scaffold a runnable Go module with browserscale init and let your agent write the flow.
- `api-reference/go.md` — Go SDK Reference: Complete reference for the github.com/browserscale/browserscale-go module: context-first methods, Method / MethodWith pairs, error codes, and runnable examples.
