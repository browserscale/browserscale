# browserscale docs (offline copy)

Bundled with this module so an agent can read and grep the SDK docs without
a network call. Guides show both Go and TS snippets; the API reference here
is **Go only**. Prefer opening the specific file below over guessing an API.

## Files

- `introduction.md` — Docs: browserscale docs: cloud Chromium with waits, clicks, frames and network handled in the engine, driven from Go, TypeScript, the CLI or MCP. Guides and SDK reference.
- `quickstart.md` — Quickstart: Install the browserscale Go or TypeScript SDK, rent a cloud Chromium session, and run your first script in under five minutes.
- `concepts.md` — Core concepts: Sessions, locators, frames, the flat frame model, and the action / wait / read loop that every browserscale script is built on.
- `guides/locators.md` — Targeting elements: CSS, JavaScript, node-handle and coordinate locators for picking elements on the page in the browserscale Go and TypeScript SDKs.
- `guides/waiting.md` — Waiting: Wait after every navigation and page change, with a timeout per step. Race success, error and challenge states, get the matched element and frame, and a per-condition diagnosis on timeout.
- `guides/loading.md` — Page loading: Navigate, commit vs. load events, and the right signal to wait for before driving a page with browserscale.
- `guides/interaction.md` — Interacting with the page: Click, fill, hover, scroll, drag and select. Actions find their target, wait for it to settle, check the pixel before pressing and name whatever blocked them.
- `guides/reactions.md` — Reactions: Handle cookie banners, modals and interstitials once: a reaction is carried out by the browser in every frame, while your flow keeps running.
- `guides/reading.md` — Reading the page: Read text content, attributes, the DOM, screenshots and accessibility observations from a live browserscale session.
- `guides/dom-mirror.md` — Live DOM mirror: Hold a browserscale page's DOM as one live tree across every frame, updated incrementally, reporting changes only inside the part you expanded.
- `guides/evaluation.md` — Evaluating JavaScript: Run JavaScript inside the page or a specific frame from your Go or TypeScript script and return typed values back.
- `guides/frames.md` — Frames & iframes: Every frame, same-origin or cross-origin and at any depth, is one frameId. Search all of them in one call, including frames that appear while you wait.
- `guides/network.md` — Network: Inspect, intercept, and modify network traffic in a browserscale session — armed before the triggering action, then awaited.
- `guides/capture.md` — Capturing traffic: Stream every request a browserscale session completes — cross-process iframes, workers and each redirect hop included — without ever pausing the page.
- `guides/cookies.md` — Cookies, storage & sessions: Bring a user back across runs: cookies and storage as data, a signed-in login as one portable object, a pinned machine identity and reattaching to live sessions.
- `guides/captchas.md` — Captchas: Passive checks pass on their own; interactive challenges are solved in the live session by browserscale's own solver. When to call SolveCaptcha, and when not to.
- `guides/shadow-canvas.md` — Shadow DOM & canvas: Read across browser security boundaries in a browserscale session: pierce closed shadow roots and read pixels from tainted cross-origin canvases.
- `guides/agentic-coding.md` — Agentic coding: Let a coding agent build your automation: explore with the hosted MCP server, scaffold a runnable Go module with browserscale init, let the agent write the flow.
- `api-reference/go.md` — Go SDK Reference: Complete reference for the github.com/browserscale/browserscale-go module: context-first methods, Method / MethodWith pairs, error codes, and runnable examples.
