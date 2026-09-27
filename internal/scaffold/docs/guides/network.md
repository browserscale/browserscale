<!--
  url: https://browserscale.cloud/docs/guides/network
  title: Network
  description: Inspect, intercept, and modify network traffic in a browserscale session — armed before the triggering action, then awaited.
-->

# Network

The network layer is what sits between the page and the open internet:
ads load through it, APIs run through it, fonts and tracker pixels and
WebSockets all pass through it. browserscale gives you four levers on that
layer — block, cache, observe, mutate — and they all share the same
URL-wildcard vocabulary, so you build muscle memory once and reuse it
everywhere.

All four act on requests you name in advance. If the question is
instead "what did this session actually do", that is a capture rather
than an interception — see [Capturing
traffic](/docs/guides/capture).

> **TL;DR**
>
> - Two **persistent** controls: `SetBlockList` (drop matching requests) and `SetStaticPaths` (serve matching requests from a server-side cache).
> - Two **one-shot** observers: `WaitForAnyRequest` and `WaitForAnyResponse` race a set of URL patterns and resolve on the first hit. Set `Abort` to drop the request with an empty 200 instead of letting it through.
> - One **one-shot mutator**: `ModifyRequest` waits for a single matching request, rewrites headers and/or body, then forwards it.
> - URL patterns are simple wildcards — `*.example.com/*`, `*/api/login`. No regex.
> - Header changes are plain `HeaderModification` structs: `{Action: "add", Name: "X-Trace", Value: "abc", Before: "Cookie"}`.
> - To log **every** request instead of trapping one, use `CaptureNetwork` — see [Capturing traffic](/docs/guides/capture).

## The four levers at a glance

| Lever | Lifetime | Direction | Use it for |
| --- | --- | --- | --- |
| `SetBlockList` | persistent (until replaced) | request | Ads, trackers, analytics, anything you never want loaded. |
| `SetStaticPaths` | persistent (until replaced) | request | Replay frozen JS/CSS/image bundles from a snapshot. Big speed/cost wins on repeated runs. |
| `WaitForAnyRequest` | one-shot | request | Verify the page actually fires the API call you expect, optionally dropping it before it leaves. |
| `WaitForAnyResponse` | one-shot | response | Capture an API response body for assertions or to seed downstream logic. |
| `ModifyRequest` | one-shot | request | Add auth headers, rewrite a body, swap a User-Agent for one specific call. |

Everything beyond this point is just filling those rows in.

Every row is a *control*: it names requests in advance and can pause,
drop or rewrite them. `CaptureNetwork` is the passive counterpart —
it never pauses anything and reports every request the session
completed, whichever frame or worker issued it. The two compose fine;
[Capturing traffic](/docs/guides/capture) covers when to reach for
which.

## URL patterns

Every network method speaks the same wildcard syntax: `*` matches any
character span. There is no regex, no glob brackets, no anchoring —
the pattern is matched against the full request URL. A few examples
that come up constantly:

- `*.doubleclick.net/*` — every request to any doubleclick subdomain
- `*googletagmanager.com*` — anywhere in the URL contains that string
- `*/api/login` — any host, path ends in `/api/login`
- `https://example.com/v2/*` — exact host + path prefix

Patterns are case-insensitive and compared against the URL as the
browser sees it (after redirects, with query string included).

## Block requests with `SetBlockList`

`SetBlockList` swaps the session's current blocklist for a new one.
Matching requests never leave the browser — they fail as if the network
itself rejected them. Pass an empty slice to clear the list.

**Go:**

```go
_ = browser.SetBlockList(ctx, []string{
    "*.doubleclick.net/*",
    "*googletagmanager.com*",
    "*.hotjar.com/*",
    "*/collect?*",
})
```

**TypeScript:**

```ts
await browser.setBlockList([
    "*.doubleclick.net/*",
    "*googletagmanager.com*",
    "*.hotjar.com/*",
    "*/collect?*",
]);
```

Two things worth burning in:

- The call **replaces** the list, it does not append. Track the full
  list in your code if you want to add patterns dynamically.
- Blocked requests fail at the network layer. Page JavaScript will
  see a normal request error (often as a rejected `fetch` or a
  console warning) — useful for ad/tracker filtering, occasionally
  surprising if the site degrades unhappily without that one
  beacon.

## Cache static assets with `SetStaticPaths`

`SetStaticPaths` wires the session to a server-side blob cache. On
cache hit, matching GETs are answered from the snapshot without
touching the origin; on cache miss, the request goes out normally and
the response is stored for next time. Same shape: empty patterns
disables.

**Go:**

```go
_ = browser.SetStaticPaths(ctx, "snap-2026-05", []string{
    "*.example.com/static/*",
    "*.example.com/*.js",
    "*.example.com/*.css",
})
```

**TypeScript:**

```ts
await browser.setStaticPaths("snap-2026-05", [
    "*.example.com/static/*",
    "*.example.com/*.js",
    "*.example.com/*.css",
]);
```

Worth using when you run the same flow repeatedly against the same
origin: the JS/CSS/image bundles don't move between runs, but they
re-download every time you spin up a fresh session. Snapshotting them
once typically cuts both wall-clock time and outbound bandwidth on
repeat runs significantly. The `blobName` is the key your snapshots
live under server-side — choose a stable per-environment name and
keep using it.

## Observe a single request with `WaitForAnyRequest`

`WaitForAnyRequest` blocks until the *next* request matching any of
your patterns is observed, then resolves with which pattern matched
(by index) and the request that triggered it. It is **one-shot**:
the next match wins, subsequent matching requests are not
intercepted.

### The basic call

The simplest case is "the page is doing things on its own and I want
to watch the next interesting request go by" — e.g. an SPA that
polls an endpoint in the background, or a flow already driven by
some other code. You just call `WaitForAnyRequest` with one or more
patterns and read the result:

**Go:**

```go
idx, req, err := browser.WaitForAnyRequest(ctx, 5000, []browserscale.RequestPattern{
    {URL: "*/api/*"},
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("matched", idx, ":", req.Method, req.Url)
```

**TypeScript:**

```ts
const { index, request } = await browser.waitForAnyRequest(
    [{ url: "*/api/*" }],
    { timeoutMs: 5000 },
);
console.log("matched", index, ":", request?.method, request?.url);
```

That's the whole API surface — one call, one result, no concurrency.
If a matching request fires within the timeout, you get it; if not,
the call returns an error.

### Arm before you trigger

The basic call works fine when the trigger is outside your script.
The moment **you** trigger the request — by clicking a button, calling
`Navigate`, running `Evaluate`, etc. — ordering becomes the whole
game.

The naive version looks like this:

**Go:**

```go
// WRONG: race between the click and the wait registering.
_, _ = browser.Click(ctx, browserscale.CSS("button#login"))
idx, req, err := browser.WaitForAnyRequest(ctx, 5000, []browserscale.RequestPattern{
    {URL: "*/api/login"},
})
```

**TypeScript:**

```ts
// WRONG: race between the click and the wait registering.
await browser.click(css("button#login"));
const { index, request } = await browser.waitForAnyRequest(
    [{ url: "*/api/login" }],
);
```

By the time `WaitForAnyRequest` registers its interceptor on the
server, the login request may already have flown by — you'll time out
on a request you saw with your own eyes.

The fix is **register the interceptor first, then trigger, then read
the result.** In TypeScript that's a non-awaited promise; in Go it's
a goroutine plus a channel — same shape, different syntax:

**Go:**

```go
type result struct {
    req *browserscale.InterceptedRequest
    err error
}
done := make(chan result, 1)
go func() {
    _, req, err := browser.WaitForAnyRequest(ctx, 5000, []browserscale.RequestPattern{
        {URL: "*/api/login"},
    })
    done <- result{req, err}
}()

_, _ = browser.Click(ctx, browserscale.CSS("button#login"))

r := <-done
if r.err != nil {
    log.Fatal(r.err)
}
fmt.Println("matched:", r.req.Method, r.req.Url)
```

**TypeScript:**

```ts
const pending = browser.waitForAnyRequest(
    [{ url: "*/api/login" }],
    { timeoutMs: 5000 },
);

await browser.click(css("button#login"));

const { index, request } = await pending;
console.log("matched", index, ":", request?.method, request?.url);
```

A few things worth noting:

- **Click is a network-trigger, but so is `Navigate`, `Fill` (which
  may fire `input`-driven XHRs), key presses, and arbitrary `Evaluate`
  calls.** The "arm before trigger" rule applies to any action that
  could fire the request you want to catch.
- **In Go, the goroutine takes a tiny moment to schedule and call
  into the server.** For pages where the click immediately fires the
  request (no JS hop), prefer arming the wait *before* even
  navigating to the page — or chain a quick `Wait` for the button
  itself between the goroutine start and the click. The brief window
  is usually a non-issue, but it's worth knowing about for very fast
  flows.
- **The same pattern works for `WaitForAnyResponse` and
  `ModifyRequest`.** Both are one-shot interceptors and both lose
  races for the same reason.

### The captured request

Once the wait resolves, `InterceptedRequest` carries:

| Field | What it holds |
| --- | --- |
| `Method` | HTTP verb (`GET`, `POST`, `PUT`, …). |
| `Url` | Full request URL the browser actually sent. |
| `Headers` | `[]Header` (name/value pairs). Order preserved. |
| `Body` | POST/PUT/PATCH body as a string. Empty for non-body requests. |
| `ResourceType` | Browser classification — `Document`, `XHR`, `Fetch`, `Script`, `Image`, `Stylesheet`, etc. Useful for filtering after the fact. |

### Drop a request mid-flight with `Abort`

Pass `Abort: true` on a pattern and the matching request gets dropped
before it leaves the browser, with the page seeing an empty `200 OK`
in its place. Useful for stubbing an API while you assert against
something else:

**Go:**

```go
idx, req, _ := browser.WaitForAnyRequest(ctx, 5000, []browserscale.RequestPattern{
    {URL: "*/api/telemetry", Abort: true},     // drop
    {URL: "*/api/login"},                      // observe
})
fmt.Println("idx", idx, "url", req.Url)
```

**TypeScript:**

```ts
const { index, request } = await browser.waitForAnyRequest([
    { url: "*/api/telemetry", abort: true },
    { url: "*/api/login" },
]);
console.log("idx", index, "url", request?.url);
```

`Abort` is per-pattern: in a multi-pattern wait you can drop some
matches and pass others through. The page never learns the request
was intercepted — it just sees an empty 200.

## Observe a single response with `WaitForAnyResponse`

Same shape, response phase. You get the status, headers and body the
origin actually returned:

**Go:**

```go
idx, resp, err := browser.WaitForAnyResponse(ctx, 5000, []browserscale.RequestPattern{
    {URL: "*/api/me"},
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("status", resp.StatusCode, "body bytes:", len(resp.Body))
_ = idx
```

**TypeScript:**

```ts
const { index, response } = await browser.waitForAnyResponse(
    [{ url: "*/api/me" }],
    { timeoutMs: 5000 },
);
console.log("status", response?.statusCode, "body bytes:", response?.body.length);
```

`Abort: true` here replaces the real response with an empty 200
*after* the request reached the origin — handy for "let the API
record the call but don't let the page see the data". Bodies are
captured as strings; very large payloads are truncated server-side at
~10 MB.

## Modify a request before it goes out

`ModifyRequest` is `WaitForAnyRequest` + rewrite. It blocks for the
next request matching one URL pattern, applies your header changes
and/or body replacement, then forwards the modified request to the
network. It is **one-shot**.

Each modification is a plain `HeaderModification` struct/object —
no builders, just data:

**Go:**

```go
req, err := browser.ModifyRequest(ctx, "*/api/me", "", 5000, []browserscale.HeaderModification{
    {Action: browserscale.HeaderModificationAdd, Name: "X-Trace", Value: "abc123", Before: "Cookie"},
    {Action: browserscale.HeaderModificationEdit, Name: "User-Agent", Value: "MyAgent/1.0"},
    {Action: browserscale.HeaderModificationRemove, Name: "Accept-Language"},
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("forwarded headers:", req.Headers)
```

**TypeScript:**

```ts
const req = await browser.modifyRequest("*/api/me", {
    modifications: [
        { action: "add", name: "X-Trace", value: "abc123", before: "Cookie" },
        { action: "edit", name: "User-Agent", value: "MyAgent/1.0" },
        { action: "remove", name: "Accept-Language" },
    ],
    timeoutMs: 5000,
});
console.log("forwarded headers:", req?.headers);
```

The three actions cover the lifecycle:

| Action | What it does | Notes |
| --- | --- | --- |
| `"add"` | Inserts a new header. | Set `Before` / `After` (Go) or `before` / `after` (TS) to an existing header name to control insertion position. Without an anchor, appended at the end. |
| `"edit"` | Overwrites an existing header's value. | No-op if the request never carried the header — use `"add"` instead. |
| `"remove"` | Drops the header from the outgoing request. | Use to strip `Cookie`, tracking headers, or any other site-set field. `Value` is ignored. |

### Replace the body too

Pass a non-empty `body` (Go positional / TS opts) to swap the
POST/PUT/PATCH body for whatever you want. The Content-Length and
related headers are recomputed by the browser. Leave empty to keep
the original body.

**Go:**

```go
_, _ = browser.ModifyRequest(ctx, "*/api/checkout",
    \`{"sku":"test","quantity":1}\`,
    5000, nil,
)
```

**TypeScript:**

```ts
await browser.modifyRequest("*/api/checkout", {
    body: '{"sku":"test","quantity":1}',
});
```

The return value is the *modified* `InterceptedRequest` — the
headers and body that actually went on the wire after your changes
were applied, so you can assert against them or log them.

## Combining the levers — practical recipes

**Drop ads and analytics for the whole session, then capture one
specific API hit** — block list is persistent, so it goes once at
the top; the response wait is one-shot and armed before the click:

**Go:**

```go
_ = browser.SetBlockList(ctx, []string{
    "*.doubleclick.net/*", "*googletagmanager.com*", "*.hotjar.com/*",
})

type respResult struct {
    resp *browserscale.InterceptedResponse
    err  error
}
done := make(chan respResult, 1)
go func() {
    _, r, err := browser.WaitForAnyResponse(ctx, 5000, []browserscale.RequestPattern{
        {URL: "*/api/orders"},
    })
    done <- respResult{r, err}
}()

_, _ = browser.Click(ctx, browserscale.CSS("button#submit"))

r := <-done
if r.err != nil {
    log.Fatal(r.err)
}
fmt.Println("order id:", parseOrderId(r.resp.Body))
```

**TypeScript:**

```ts
await browser.setBlockList([
    "*.doubleclick.net/*", "*googletagmanager.com*", "*.hotjar.com/*",
]);

// Arm the response interceptor, click, then await.
const pending = browser.waitForAnyResponse(
    [{ url: "*/api/orders" }],
    { timeoutMs: 5000 },
);
await browser.click(css("button#submit"));
const { response } = await pending;
console.log("order id:", parseOrderId(response?.body ?? ""));
```

**Inject an auth token into one outgoing API call without touching
the page's JS:**

**Go:**

```go
done := make(chan error, 1)
go func() {
    _, err := browser.ModifyRequest(ctx, "*/api/me", "", 5000,
        []browserscale.HeaderModification{{Action: browserscale.HeaderModificationAdd, Name: "Authorization", Value: "Bearer " + token}},
    )
    done <- err
}()
_, _ = browser.Click(ctx, browserscale.CSS("button#load-profile"))
if err := <-done; err != nil {
    log.Fatal(err)
}
```

**TypeScript:**

```ts
const pending = browser.modifyRequest("*/api/me", {
    modifications: [{ action: "add", name: "Authorization", value: "Bearer " + token }],
    timeoutMs: 5000,
});
await browser.click(css("button#load-profile"));
await pending;
```

Same shape as the basic example — arm first, trigger, await — just
with `ModifyRequest` in the wait slot.

## Gotchas

- **One-shot means one-shot.** `WaitForAnyRequest`,
  `WaitForAnyResponse`, `ModifyRequest` consume exactly the next
  match. If you want to keep intercepting, call them again — they're
  cheap to re-arm.
- **`SetBlockList` / `SetStaticPaths` replace, not append.** Keep the
  authoritative list in your script and resend the whole thing when
  it changes.
- **Arm before you trigger** — covered above. If a network wait keeps
  timing out on a request you *know* fires, it's almost always
  because the click happened before the interceptor was registered.
- **`Abort` returns an empty 200.** The page does *not* see a network
  error. Sites that branch on response shape rather than HTTP status
  will likely treat the empty body as "request succeeded, no data".
- **No regex, no anchors.** Patterns are wildcards on the full URL.
  `*` matches any character span; everything else is literal.
- **Response bodies are truncated at ~10 MB.** Don't rely on
  `WaitForAnyResponse` to mirror multi-megabyte downloads — it's an
  observation tool, not a download endpoint.
- **`"edit"` is a no-op when the header is missing.** If you want
  the header to exist regardless of the original request, prefer
  `"add"` (or send a `"remove"` + `"add"` pair to force-replace it).
- **`Before`/`After` only affect `"add"`.** They're ignored on
  `"edit"` / `"remove"` because there's nothing to position.

## See also

- [Capturing traffic](/docs/guides/capture) — the passive counterpart: every request the session completed, streamed to a handler.
- [Loading pages](/docs/guides/loading) — `LoadHTML` for serving a synthetic response instead of one URL pattern.
- [Cookies & sessions](/docs/guides/cookies) — auth flows that pair with `ModifyRequest` for header-level overrides.
- API reference: [Go `SetBlockList` / `ModifyRequest` / …](/docs/api-reference/go#SetBlockList) · [TS `setBlockList` / `modifyRequest` / …](/docs/api-reference/ts#setBlockList).

→ Continue: [Capturing traffic](/docs/guides/capture)
