<!--
  url: https://browserscale.cloud/docs
  title: Docs
  description: browserscale documentation: real cloud Chromium with waits, clicks, frames and network handled inside the engine. Drive it from Go, TypeScript, the CLI or MCP. Quickstart, guides and the full SDK reference.
-->

# Introduction

browserscale is browser automation that doesn't guess. You rent a real
Chromium browser in the cloud and drive it from **Go**, **TypeScript**, the
**CLI** or any **MCP** client, and the hard parts of automation are no
longer approximated from outside the browser. They happen inside the
engine: a wait is reported by the page the instant it is true, a click
checks the exact pixel it is about to press, and a cross-origin iframe is
just another frame in one tree.

Each session is an isolated browser context with its own cookies,
storage, proxy and fingerprint, ready in under 250 ms, running on real
consumer GPUs. Run one, or thousands side by side, without shipping a
browser binary or operating a browser farm.

> **TL;DR**
>
> - A **real browser** in the cloud, driven over one session API from Go, TypeScript, the CLI or MCP.
> - Waits, clicks, frames and network interception are handled **inside the browser engine**, so they are exact instead of polled and invisible to the page.
> - Every failure comes back as a **stable error code with typed detail**: what timed out, what blocked the click, how far each condition got.
> - Stealth, proxies, identities, captchas, live streaming and scale are part of the platform, not add-ons you bolt on.

## What you get

### Acting on the page

- **Clicks that check before they press.** The element is scrolled into
  view through nested scrollers and up the frame chain, held until it stops
  moving, approached on a human pointer path, and the pixel under the
  pointer is verified to belong to it before the button goes down. If
  something covers it, the click re-aims or steps out of a hover overlay's
  way. If it is still blocked, the click refuses and names the element in
  the way instead of landing on the wrong thing.
- **Input the way hardware sends it.** Pointer and key events take the
  path a real mouse and keyboard take. Typing follows the session region's
  keyboard layout, with per-character timing that varies like a hand.
- **Failures you can act on.** Every command answers with success or a
  stable error code (`not_found`, `occluded_after_evade`, `timeout`, …)
  plus typed detail, so your code, or a model, can repair the failure
  instead of retrying blindly.

→ [Interaction](/docs/guides/interaction) · [Targeting elements](/docs/guides/locators)

### Waiting and reacting

- **Waits the page reports.** Each document tells the wait the moment a
  condition holds, usually within a frame. An idle wait costs nothing, and
  more conditions or more frames cost a registration, not another polling
  loop. By default a match means *visible and holding still*, not merely
  present in the DOM.
- **Races as a first-class shape.** Pass several outcomes (success, error
  toast, captcha, login wall) and get back which one happened first,
  together with the frame and a node handle the next action can use
  directly.
- **Timeouts that explain themselves.** A `WaitError` says, per condition,
  how far it got: not found, found but hidden, found but covered (with the
  covering element), or not yet steady.
- **Reactions.** Arm a one-shot handler for the cookie banner or popup that
  may or may not appear. The browser fires it on its own, across every frame
  and navigation, between your calls, and then retires it.

→ [Waiting](/docs/guides/waiting) · [Loading pages](/docs/guides/loading) · [Go `AddReaction`](/docs/api-reference/go#AddReaction)

### Frames without bookkeeping

- **One flat frame tree.** The main document, same-origin iframes and
  cross-origin, out-of-process iframes are all just a `frameId`: no
  per-frame sessions, no isolated worlds, no depth limit. A frame created
  while a wait is running is covered the moment it exists.

→ [Frames & iframes](/docs/guides/frames) · [Shadow DOM & canvas](/docs/guides/shadow-canvas)

### Stealth on real hardware

- **Control lives below the page.** Commands are carried out by the browser
  itself. Nothing is injected into the page, there is no DevTools
  handshake, and page JavaScript has nothing to observe.
- **Real consumer GPUs.** Canvas, WebGL, audio and codec readbacks are
  genuinely rendered. There is no spoofing layer and no hash database for a
  deeper check to unmask.
- **A shipped Chrome, not a build of one.** Sessions carry the state and
  wire behavior of a consumer browser, consistent with the region they exit
  from and reproducible from run to run.
- **Captchas without third-party solvers.** Interactive challenges are
  completed in the live session by browserscale's own solver. The
  provider's own JavaScript issues the token; nothing is synthesized or
  bought from an external API.

→ [Captchas](/docs/guides/captchas)

### Network

- **Interception armed before anything loads.** It sits in the browser's
  network stack, so every frame, cross-process iframe, worker and service
  worker passes through it. There is no attach race, and nothing slips by.
- **Capture that never pauses the page.** Stream every finished request
  with the headers and cookies that actually went on the wire, each
  redirect hop as its own exchange, with bodies copied off to the side.
- **Catch one call and change it.** Wait for a request or response; block,
  mock or rewrite it; or answer a whole navigation yourself.
- **Pay for static assets once.** Heavy JS, CSS and images can be served
  from a server-side cache outside the proxy, so repeat runs pay neither
  the download nor the proxy bandwidth.

→ [Network](/docs/guides/network) · [Capturing traffic](/docs/guides/capture)

### Identity and state

- **A login as one portable object.** Export a signed-in persona,
  device-bound sessions included, and bring it up signed in inside a fresh
  context.
- **Cookies and storage as data.** Read and write the whole cookie jar,
  partitioned cookies included, and local storage per origin, with no page
  open.
- **A machine you can come back as.** A country sets language, locale,
  timezone and keyboard together; cores, memory and renderer stay
  consistent in every frame and worker. Pin the fingerprint and the next run
  is the same computer returning. Bring your own proxy, or let browserscale
  allocate one.

→ [Cookies, storage & sessions](/docs/guides/cookies)

### Seeing the page

- **Observation sized for a prompt.** One line per element across every
  frame and closed shadow root, with role, live value, label and flags,
  under a token budget. That is prompt-sized instead of a megabyte of HTML,
  and it takes one round trip.
- **A live DOM mirror.** Keep an incrementally updated copy of the page.
  Only what changed in the part you expanded is sent, and an `<iframe>` is
  an ordinary element holding its document.
- Plus screenshots, canvas pixel reads, element inspection at a point, and
  JavaScript evaluation in any frame.

→ [Reading the page](/docs/guides/reading) · [Live DOM mirror](/docs/guides/dom-mirror) · [Evaluation](/docs/guides/evaluation)

### Sessions at scale

- **Contexts, not machines.** Each session has its own cookies, storage,
  cache, proxy and persona, is ready in under 250 ms, and runs beside
  thousands of others without sharing state.
- **Sessions you can find again.** The browser lives server-side, so a
  session outlives the process that rented it. List what a key holds and
  reattach from any machine.
- **Operated for you.** Heavy sessions cannot starve their neighbors,
  capacity is warm before you rent, and a full host fails fast instead of
  hanging.
- **Watch it live, take over.** A low-latency WebRTC stream of the real
  page, with mouse, keyboard and clipboard takeover from the dashboard or
  the CLI.

### Scripts beside the browser (BrowserVM, early access)

- **Run your own JavaScript next to the page, not in it.** The script runs
  in an isolate of its own and reaches the document through the engine. A
  cross-origin `<iframe>` is plain `contentDocument`, values are live
  objects, an element goes straight into `browser.click`, and the page sees
  nothing injected. Steps cost microseconds instead of network round trips,
  so a loop over a hundred rows is cheap.
- Access is opened per account while in early access. Ask
  [support](mailto:support@browserscale.cloud) or on
  [Discord](https://discord.gg/SfE9C9K28D).

→ [BrowserVM](/browservm)

### Built for agents

- **A hosted MCP server** exposes the same verbs as the SDKs to Cursor,
  Claude, Codex or any MCP client, so an agent can look at a real page while
  it writes the automation. Your key stays in a header, never in the
  model's context.
- **`browserscale init`** scaffolds a runnable project with a worker loop,
  a proxy pool, an `AGENTS.md` and this documentation offline.

→ [Agentic coding](/docs/guides/agentic-coding)

## Ways to drive it

Everything above runs against the same session API, so pick whichever
entry point fits the job. You can also mix them: rent from the CLI, drive
from Go, and watch in the dashboard.

| Entry point | What it is |
| --- | --- |
| [**browserscale-go**](https://github.com/browserscale/browserscale-go) | The Go SDK. Context-first methods, explicit errors. |
| [**browserscale-ts**](https://github.com/browserscale/browserscale-ts) | The TypeScript SDK, for Node.js and the browser. |
| [**browserscale**](https://github.com/browserscale/browserscale) (CLI) | `init`, `rent`, `list`, `view`, `run` and `stop` from your terminal. |
| [**MCP server**](/docs/guides/agentic-coding) | The browser as tools for any MCP client, one-to-one with the SDK. |
| [**BrowserVM**](/browservm) | Your script, running inside the browser beside the page. |

## Install

**Go:**

```go
go get github.com/browserscale/browserscale-go
```

**TypeScript:**

```ts
npm install browserscale-ts
```

Go needs **1.22+** and Node needs **18+**. Both SDKs are open source. The Go
package is documented on
[pkg.go.dev](https://pkg.go.dev/github.com/browserscale/browserscale-go),
and the TypeScript package is
[`browserscale-ts` on npm](https://www.npmjs.com/package/browserscale-ts).
[Quickstart](/docs/quickstart) takes you from here to a running script.

## What people build with it

- **Scraping and enrichment at volume:** one isolated session per job,
  static assets cached server-side, and every response captured without
  slowing the page.
- **Account, checkout and booking flows:** portable logins, a fingerprint
  that returns as the same machine, and waits that race every way the page
  can branch.
- **AI agents that use the web:** prompt-sized observations, error codes a
  model can reason about, and an MCP server it can call directly.
- **End-to-end checks against production:** real proxies, real captchas
  and real network conditions, with a live stream to watch what happened.

## Glossary

Four nouns come up on almost every page from here on.

- **Session:** one rented browser. It lives until you call `Stop` or the
  `rentDuration` you booked expires, and it is addressed by one `sessionId`.
- **Page:** a tab or popup inside a session. A session can hold several,
  for example when a click opens `target="_blank"`.
- **Frame:** every page is a tree of frames. The main document is the root,
  each `<iframe>` is a child, and cross-origin frames are full members of
  the same tree.
- **Locator:** your declarative *"which element, under what conditions"*.
  The same locator works as a wait condition and as a click, fill or drag
  target.

## How these docs are structured

**Quickstart** gets a working script running in a few minutes. **Core
concepts** names the nouns you just used. The guides after that each take
one job (driving a page, reading it, network and identity, agents) and go
deep. The **API reference** lists every method and type for both SDKs.

## See also

- [Quickstart](/docs/quickstart): a runnable example to copy and paste.
- [Go SDK reference](/docs/api-reference/go) and [TypeScript SDK reference](/docs/api-reference/ts): every method browserscale exposes.
- [Agentic coding](/docs/guides/agentic-coding): connect the MCP server and let a coding agent build the automation.

→ Continue: [Quickstart](/docs/quickstart)
