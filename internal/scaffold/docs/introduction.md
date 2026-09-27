<!--
  url: https://browserscale.cloud/docs
  title: Docs
  description: browserscale documentation: rent a real cloud Chromium session and drive it from Go or TypeScript. Quickstart, core concepts, guides, and the full SDK reference.
-->

# Introduction

browserscale is a browser-as-a-service platform for running
large-scale automation on real browser (Chromium) sessions. You write
your script in **Go** (`browserscale-go`) or **TypeScript** (`browserscale-ts`), and browserscale
runs the browser work in our infrastructure — behind proxies, with
separate browser contexts and undetected fingerprints for each parallel
task. Scale from one job to hundreds or thousands without shipping a
headless binary or babysitting your own browser farm.

> **TL;DR**
>
> - browserscale rents you a **real browser**, somewhere in the cloud, that you drive remotely.
> - You write Go or TypeScript — there is no `chromium` binary on your machine.
> - Proxies, fingerprints, captcha solving and network mocking are first-class.

## Install

Pull the SDK for your language from its public registry:

**Go:**

```go
go get github.com/browserscale/browserscale-go
```

**TypeScript:**

```ts
npm install browserscale-ts
```

Go needs **1.21+**, Node needs **18+**. Both SDKs are open source and
talk to the same browserscale session API — see [Quickstart](/docs/quickstart)
for a full runnable example.

**Packages and source**

- **Go (`browserscale-go`)** — source on [GitHub](https://github.com/browserscale/browserscale-go), docs on [pkg.go.dev](https://pkg.go.dev/github.com/browserscale/browserscale-go). Install with `go get github.com/browserscale/browserscale-go`.
- **TypeScript (`browserscale-ts`)** — the [`browserscale-ts` package on npm](https://www.npmjs.com/package/browserscale-ts), source on [GitHub](https://github.com/browserscale/browserscale-ts). Install with `npm install browserscale-ts`.

## Use cases

browserscale is built for workloads where every task needs a real browser identity,
not just a DOM parser or a local headless process. Common use cases
include:

- Running large scrape, crawl or enrichment queues with one isolated
  browser (Chromium) session per job.
- Automating account, checkout, booking or form flows that depend on a
  stable fingerprint, cookies and realistic browser behavior.
- Testing production journeys through real proxies, captchas and network
  conditions instead of mocked local pages.
- Parallelizing browser work across many independent sessions without
  managing Chromium installs, workers or browser lifecycle yourself.

## Features

- **Real browser (Chromium) sessions as a service** — run automation
  against a full browser engine in the cloud, with pages, frames,
  cookies, storage and network state.
  - Real Chromium runtime, not a DOM-only scraper.
  - Persistent session state while the rental is alive.
- **Parallel isolated contexts** — each task gets its own session,
  fingerprint and lifecycle, so large queues do not share browser state.
  - Separate cookies, storage, pages and network state per session.
  - Independent lifecycle per job, from rental to cleanup.
  - Browser contexts, not VMs or processes — sub-250 ms spin-up, scaling to thousands in parallel.
- **Fingerprint and proxy handling** — route traffic through
  proxies and use browser identities designed for real production sites.
  - Server-side fingerprint ids that can be pinned and reused.
  - Browser properties, canvas, audio and WebGL responses tied to the
    selected fingerprint.
  - Matching speech synthesis voices, media codecs, DRM support and many
    more browser surfaces, so the session looks like a real user browser
    instead of a generic Chromium build.
  - Locale and timezone defaults aligned with the rented identity unless
    you override them.
  - Native Chrome control without CDP, Playwright or Puppeteer leaks such
    as `Runtime.enable` side-effects or DevTools handshake fingerprints.
  - Bring your own proxy or let browserscale allocate one server-side.
- **Native engine-level control** — automation runs *below* the page,
  inside the browser engine, instead of driving it from outside over the
  DevTools protocol like Playwright or Puppeteer. This is where both the
  undetectability and the speed come from.
  - Commands run natively inside Chromium itself, straight in the renderer
    process — nothing is injected into the page, no `Runtime.enable`, no
    DevTools handshake — so page JavaScript cannot observe the automation
    and no trace is left behind.
  - Waits are armed once inside the engine and run fully async; there is
    no round-trip "is it there yet?" polling loop over a WebSocket, so a
    wait fires the instant its condition is met.
  - Steady-time checks are built in — an element is only reported back
    once it exists and holds its position in the DOM, so animations and
    layout shifts are absorbed before your click instead of being
    discovered after it missed. It's all configurable from the SDK, so you
    can relax the checks to match hidden or off-screen elements too.
- **One flat frame tree** — the main document, same-origin iframes and
  cross-origin OOPIFs are all just a `frameId` in a single tree, with none
  of the flattened-session, `Runtime.enable` and per-frame
  execution-context juggling other stacks force on you.
  - `Wait` and `Click` act across every frame at once, or scope to a
    single iframe — the SDK resolves the nesting for you whether the
    target sits in a nested OOPIF or the top document.
  - Searching across all frames, `Wait` hands back the `frameId` that
    matched, ready to pass straight into the next action with no second
    lookup or manual frame switching.
- **Flow-optimized SDKs** — use JavaScript locators when CSS is not
  enough, and explicit wait logic when real pages can branch.
  - `Wait` can race multiple outcomes and return which one matched.
  - JS locators can target elements by page logic, then hand the result
    to normal actions instead of synthetic DOM clicks.
- **Agent-friendly observation** — hand a model a compact view of the page
  instead of a megabyte of raw HTML.
  - `GetObservation` returns the visible elements as one line each, under
    headers that already carry the URL, title and scroll position.
  - Live form state comes with it: the value actually typed into a field,
    checkbox state, `<select>` options, and flags for duplicate ids.
  - Elements from nested iframes and cross-origin OOPIFs are included in
    the same observation, each tagged with its frame — as are open and
    closed shadow roots.
  - The budget is measured in tokens across the whole page, and running
    out degrades to interactive elements only instead of truncating in
    document order and losing the submit button.
- **Live DOM mirror** — follow the page's structure instead of re-fetching
  it. `MirrorDom` holds the whole page as one incrementally updated tree.
  - Changes are reported only inside the part you expanded, so a subtree
    nobody opened costs a child count per batch instead of a re-serialized
    document — which is what makes a DevTools-style view practical against
    a page that rewrites a list sixty times a second.
  - An `<iframe>` is an ordinary element whose one child is the document it
    hosts, cross-origin ones included, so there is no per-frame recursion
    and frames nobody opened cost nothing.
  - `GetDomRevision` is an O(1) mutation counter for callers that would
    rather poll than consume events, replacing a `GetDOMHash` loop that
    re-serializes the tree to notice one attribute changed.
- **Human-like interaction** — mouse movement and input are generated to
  look like real browser usage instead of instant synthetic jumps.
  - Mouse paths use browserscale's own movement algorithm for more natural cursor
    timing and trajectories.
- **WebRTC live video stream** — watch and control the rented browser
  directly from the browserscale web interface.
  - Low-latency browser video is streamed to the page for debugging,
    demos and manual takeover.
  - Mouse and keyboard input are sent back over WebRTC data channels.
- **Captcha support, no third-party solvers** — common captcha flows can be
  detected and solved from the same automation run.
  - Passive anti-bot checks are handled without extra client code.
  - Interactive challenges can be solved with `SolveCaptcha` /
    `solveCaptcha`.
  - Solving is done by browserscale's own AI solver, which learns the known challenge
    types — puzzle, OCR, slide, hold and more — on its own and keeps
    improving as they evolve.
  - No token is ever synthesized or fetched from an external API: the
    challenge is completed in the valid live browser and the provider's own
    JavaScript issues the token itself — which is why even new or unknown
    protections pass.
- **Real hardware, real GPUs** — sessions run hardware-accelerated on real
  consumer GPUs, not on VM cores with a WebGL faking layer.
  - Canvas and WebGL readbacks (`toDataURL`, `getImageData`) return genuinely
    rendered pixels — there is no spoofing layer and no fingerprint hash
    database to maintain or unmask.
  - New and deeper passive bot protections pass by default, because rendering
    is executed for real instead of being simulated.
- **Network control at the source** — interception sits in the browser's
  network stack itself, so every request from every frame (including
  cross-origin OOPIFs) passes through it. No handler races, nothing slips
  through.
  - Wait for requests and responses as part of the browser flow.
  - Modify headers, mock responses or block matching requests.
  - Mark repeated JS, CSS and image paths as static once with
    `SetStaticPaths` and browserscale serves them from a server-side cache, cutting
    proxy bandwidth and load time on repeat runs.
  - `CaptureNetwork` streams every request the session completes to a
    handler, from the browser process rather than a page — cross-process
    iframes, workers and each hop of a redirect chain included, with the
    headers that actually went on the wire. Nothing is paused, so the page
    still loads at full speed.
- **Go and TypeScript SDKs** — drive the same browser API from Go
  (`browserscale-go`) or TypeScript (`browserscale-ts`).
  - Use the same session concepts and browser commands in both SDKs.
  - Connect over the browserscale session API instead of running Chromium locally.
- **Built for AI coding agents** — hand the work to the agent in your editor
  instead of writing the boilerplate yourself. See
  [Agentic coding](/docs/guides/agentic-coding).
  - A hosted MCP server exposes the browser as tools that map one-to-one
    onto SDK methods, so an agent can look at a real page while it writes.
  - `browserscale init` scaffolds a runnable Go module with a worker
    loop, a proxy pool, an `AGENTS.md` and this documentation offline.

## Glossary

Four nouns show up on basically every page from here on. Skim them now,
come back if a later page uses one in a way that surprises you.

- **Session** — one rented browser. It lives until you call `Stop` or
  until the `rentDuration` you booked expires. Each session is tied to
  one `sessionId` and one gRPC URL.
- **Page** — a single tab or popup inside a session. A session can hold
  several pages — for example when a click opens `target="_blank"`.
- **Frame** — every page is a tree of frames. The main document is the
  root frame, each `<iframe>` is a child, and cross-origin frames
  (OOPIFs) are first-class members of the tree.
- **Locator** — your declarative *"which element, under what
  conditions"*. The same Locator object works as a wait condition and as
  a click / fill / drag target.

## How these docs are structured

These pages are written to be read top-to-bottom. **Quickstart** gets a
working script onto your machine in under a minute. **Core concepts**
then names the three SDK nouns you just used, so the rest of the API
reference starts to feel like vocabulary rather than mystery. The
**Guides** chapters that follow each pick one domain (waiting,
networking, captchas, …) and go deep.

If you would rather skip prose and start clicking, jump straight to
Quickstart — you can always come back here once the script runs.

## See also

- [Quickstart](/docs/quickstart) — copy-paste a runnable example.
- [Go SDK reference](/docs/api-reference/go) and [TypeScript SDK reference](/docs/api-reference/ts) — every method browserscale exposes.
- [Agentic coding](/docs/guides/agentic-coding) — connect the MCP server and let a coding agent build the automation.

→ Continue: [Quickstart](/docs/quickstart)
