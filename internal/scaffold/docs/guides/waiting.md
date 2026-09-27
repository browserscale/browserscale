<!--
  url: https://browserscale.cloud/docs/guides/waiting
  title: Waiting
  description: Explicit waits, timeouts, and the arm-before-trigger pattern every browserscale script needs to stay deterministic.
-->

# Waiting

Pages take time. Network round-trips, JS bundles, animations, third-party
widgets, captchas — there is almost always a moment between *"I told the
browser to do something"* and *"the thing I want is on the page"*. The
job of `Wait` is to bridge that gap honestly: poll the page until what
you described is actually there, then return — or give up with a clear
error if it never arrives.

> **TL;DR**
>
> - Actions never auto-wait. If the element isn't there yet, `Click` returns `ELEMENT_NOT_FOUND` immediately.
> - `Wait` polls one or more locator conditions on the server side and returns when the first one matches.
> - Default timeout is 30 s. Override with `Timeout(ms)` in Go / `{ timeoutMs }` in TS.
> - The returned `WaitResult` tells you *which* condition matched (`.index`) and where (`backendNodeId`, `frameId`, `bounds`).

## Why Wait exists

Other automation tools auto-wait inside every action — Playwright's
`page.click('button')` will internally wait up to 30 s for the element
to be actionable. browserscale actions do not. Every action method dispatches
exactly once, and if the target is missing you get an immediate error
from the server.

That tradeoff is deliberate. Auto-waiting hides what the script is
actually doing, and on the kinds of pages browserscale is built for — anti-bot,
heavy JS, multi-step flows — implicit waiting is what turns a five-line
script into a five-minute mystery when something breaks. With explicit
`Wait`, every pause is something you can see in your code.

The rule of thumb:

**Go:**

```go
// Anti-pattern: hope the button is there in time.
_, _ = browser.Click(ctx, browserscale.CSS("button.submit"))

// Pattern: describe what you're waiting for, then act.
_, _ = browser.Wait(ctx, browserscale.CSS("button.submit"))
_, _ = browser.Click(ctx, browserscale.CSS("button.submit"))
```

**TypeScript:**

```ts
// Anti-pattern: hope the button is there in time.
await browser.click(css("button.submit"));

// Pattern: describe what you're waiting for, then act.
await browser.wait(css("button.submit"));
await browser.click(css("button.submit"));
```

You will write that `Wait → act` pair a lot. Get comfortable with it
early.

## Anatomy of a wait

A `Wait` call carries three things:

1. **One or more locator conditions** — anything `CSS(...)` or `JS(...)`
   produces. (`Node` and `At` are rejected client-side; see the
   [Targeting elements guide](/docs/guides/locators) for why.)
2. **A timeout** — how long to keep polling before giving up.
3. **An implicit frame scope** — main frame by default, or whatever
   the *first condition that has* `.InFrame(...)` / `.InAllFrames()`
   on it specifies (see the frame section below).

The server polls all conditions in the target frame on a fast
interval and returns the *first* one to match. The others are
abandoned — there is no second-place winner.

The result is a `WaitResult` with five fields:

| Field | What it tells you |
| --- | --- |
| `Index` / `index` | Position of the condition that matched in your input list |
| `FrameId` / `frameId` | The frame where the match was found |
| `BackendNodeId` / `backendNodeId` | Handle to the matched element (0 for non-Element `JS` results) |
| `IsVisible` / `isVisible` | Whether the element was visible at match time |
| `Bounds` / `bounds` | CSS-pixel rect in root-viewport coordinates |

`Index` is the most useful field when you use multiple conditions —
it's how you know which branch of the race won. More on that in a
moment.

## A single condition

The shape you'll use 90 % of the time:

**Go:**

```go
r, err := browser.Wait(ctx, browserscale.CSS(".checkout-complete"))
if err != nil {
    log.Fatal("checkout never finished:", err)
}
fmt.Println("element bounds:", r.Bounds)
```

**TypeScript:**

```ts
try {
  const r = await browser.wait(css(".checkout-complete"));
  console.log("element bounds:", r.bounds);
} catch (err) {
  console.error("checkout never finished:", err);
}
```

A few things are happening implicitly here:

- The selector must match an element that is **visible** (default
  `Visible(true)`) and whose bounding rect has been **stable for at
  least 500 ms** (default `Steady(500)`). Both defaults come from the
  `CSS(...)` constructor; override per locator if needed (see the
  [Targeting elements guide](/docs/guides/locators)).
- The wait runs in the **main frame** of the active page.
- The timeout is **30 s** — the SDK's `DefaultWaitTimeoutMs`.

## Custom timeout

A common case is *"give up faster than the default 30 s"*, for instance
because you race a slow path against a fast one and want to switch
strategy quickly.

**Go:**

```go
// 5 seconds, after that bail.
_, err := browser.Wait(ctx,
    browserscale.CSS(".result"),
    browserscale.Timeout(5000),
)
```

**TypeScript:**

```ts
// 5 seconds, after that bail.
await browser.wait(css(".result"), { timeoutMs: 5000 });
```

In Go, `Timeout(ms)` is one of the variadic arguments to `Wait`. In
TypeScript, it lives inside the `opts` object — `{ timeoutMs }`. There
is no separate "no timeout" mode; pick a value you can live with even
if the page is broken.

## Racing several conditions

The more interesting case is *"either A or B should happen — whichever
gets there first, that's the answer"*. browserscale has first-class support for
that pattern: pass multiple locators, get back the `Index` of whichever
matched.

Go takes the conditions as variadic arguments to the same `Wait`
function. TypeScript splits it into `wait` (one condition) and
`waitAny` (many) so the types stay tidy.

**Go:**

```go
r, err := browser.Wait(ctx,
    browserscale.CSS(".success"),       // index 0
    browserscale.CSS(".error"),         // index 1
    browserscale.CSS(".captcha"),       // index 2
    browserscale.Timeout(10000),
)
if err != nil {
    log.Fatal(err)
}
switch r.Index {
case 0:
    fmt.Println("happy path")
case 1:
    fmt.Println("form error — re-fill and retry")
case 2:
    fmt.Println("captcha appeared — solve it")
}
```

**TypeScript:**

```ts
const r = await browser.waitAny(
    [
        css(".success"),  // index 0
        css(".error"),    // index 1
        css(".captcha"),  // index 2
    ],
    { timeoutMs: 10000 },
);
switch (r.index) {
    case 0:
        console.log("happy path");
        break;
    case 1:
        console.log("form error — re-fill and retry");
        break;
    case 2:
        console.log("captcha appeared — solve it");
        break;
}
```

A few notes on the race:

- The first condition that matches at any poll wins; the others are
  abandoned. Polling is fast enough that the order of conditions in
  the list is rarely observable in practice.
- The condition list is open-ended; you can mix `CSS` and `JS`, with
  different `.Visible(...)` / `.Steady(...)` modifiers per locator.
- On timeout you get a typed `WaitError` and no result. Its
  `conditions` array reports, *per locator you passed*, whether that
  condition was ever **found** and whether it ever became **visible**
  / **steady** — so you can see which branch was close and which never
  appeared at all. Go unwraps it with `errors.As`; TypeScript matches
  it with `instanceof WaitError`.

This is also how you handle pages that load progressively: race the
final element you actually want against an error toast that means
*"give up early"*.

## Waiting in a specific frame

The frame the wait runs in comes from the **first condition that has a
frame set** — there is no separate call-level frame option for `Wait`
yet. Two practical consequences:

1. To wait in a known iframe, put `.InFrame(id)` on (at least) one of
   the conditions.
2. To search every frame, put `.InAllFrames()` on (at least) one.

**Go:**

```go
pages, _ := browser.GetPages(ctx)
iframeId := pages[0].FrameTree.Children[0].FrameId

// Wait inside that iframe.
_, _ = browser.Wait(ctx, browserscale.CSS("button.accept").InFrame(iframeId))

// Or: search every frame for a consent button.
_, _ = browser.Wait(ctx, browserscale.CSS("button.consent").InAllFrames())
```

**TypeScript:**

```ts
const pages = await browser.getPages();
const iframeId = pages[0].frameTree.children[0].frameId;

// Wait inside that iframe.
await browser.wait(css("button.accept").inFrame(iframeId));

// Or: search every frame for a consent button.
await browser.wait(css("button.consent").inAllFrames());
```

Mixing different frame scopes across the conditions in a single race
isn't supported — only the first non-empty frame is used. If you need a
race across genuinely different frames, run two parallel waits in
goroutines / `Promise.all`.

## Using what comes back

The `WaitResult` is more than a "yes it happened" signal. Three of its
fields are worth keeping in mind:

- **`BackendNodeId`** is a stable handle to the matched element you can
  pass to a follow-up action via `Node(...)`. This is faster than
  re-resolving the selector and guarantees you act on the same element
  the wait matched.
- **`FrameId`** tells you which frame won an `InAllFrames` race, so a
  follow-up action can target that exact frame.
- **`IsVisible`** reflects the element's visibility at match time. With
  the default `Visible(true)` gating, it is always true on success. If
  you opted out with `.Visible(false)`, it may be false. For non-Element
  `JS` truthy values it is always false — there is no element to
  measure.

**Go:**

```go
r, _ := browser.Wait(ctx, browserscale.CSS("button.submit").InAllFrames())

// Click the exact element we just matched, in the exact frame it lived in.
_, _ = browser.ClickWith(ctx, browserscale.Node(r.BackendNodeId), browserscale.ClickOpts{
    InFrame: r.FrameId,
})
```

**TypeScript:**

```ts
const r = await browser.wait(css("button.submit").inAllFrames());

// Click the exact element we just matched, in the exact frame it lived in.
await browser.click(node(r.backendNodeId), { inFrame: r.frameId });
```

This `wait → act on the returned node` pattern is the cleanest way to
avoid a race between "the wait matched element X" and "by the time
the click ran, the selector resolved to a different element Y".

## Anti-patterns to avoid

- **`time.Sleep` / `setTimeout` instead of `Wait`.** Sleeps are
  fragile (right today, wrong tomorrow) and slow (you always pay the
  full delay, even when the page was ready in 50 ms). The only place a
  hard sleep is acceptable is when you genuinely need to *give time to
  something with no observable signal* (e.g. a debounce on the page).
- **Manual polling with `Evaluate`.** If you find yourself looping
  `Evaluate("...condition...")` with sleeps in between, you've
  re-implemented `Wait` — badly. Use `JS(...)` as a wait condition
  instead.
- **Pre-emptive `Wait`s on top of explicit results.** When you already
  hold a `WaitResult` or an `ElementResult` for an element, you don't
  need to wait for it again before acting — pass `Node(backendNodeId)`
  to the next action directly.
- **30-minute timeouts.** A long timeout doesn't make the page
  faster, it makes failures slow. Pick a budget for the operation and
  stick to it; rely on retries at the next layer up.

## Cancellation

In Go, the `ctx` parameter passed to every method is the cancellation
channel. A `context.Cancel` or `context.WithTimeout` aborts the wait
in-flight and returns the context error — useful for wiring a `Wait`
into a larger budget such as an HTTP handler deadline or a job-level
shutdown signal.

**Go:**

```go
waitCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
defer cancel()

_, err := browser.Wait(waitCtx, browserscale.CSS(".done"))
if errors.Is(err, context.DeadlineExceeded) {
    // Caller budget ran out — bail out, retry later.
}
```

**TypeScript:**

```ts
// TypeScript does not currently expose AbortSignal cancellation. Use
// the per-call timeoutMs option to bound how long any single wait can
// run, and structure the surrounding code so a failed wait short-
// circuits the rest of the flow.
try {
    await browser.wait(css(".done"), { timeoutMs: 8000 });
} catch {
    // Timed out — bail out, retry later.
}
```

The server timeout (`Timeout(ms)` / `timeoutMs`) and any client-side
deadline both cap how long the wait can run. Whichever fires first
wins — a 1 s `ctx` cancel beats a 30 s server timeout, and a 5 s
server timeout beats a 30 s `ctx`. They never combine into something
longer than the smaller of the two.

## Gotchas

- **A wait is not an action.** It does not click, scroll, or otherwise
  change the page. If your script *only* waits and never interacts, you
  probably wrote the wrong thing.
- **`Wait` rejects `Node(...)` and `At(...)` client-side.** These two
  locators don't carry a selector or JS expression for the server to
  poll. The SDK throws before sending.
- **No `WaitOpts.InFrame`.** As covered above, the frame for a wait
  comes from the first condition that has one. There is no separate
  call-level `InFrame` for `Wait` (unlike actions, which do have it).

## See also

- [Targeting elements](/docs/guides/locators) — the locator constructors and modifiers `Wait` accepts.
- [Frames & iframes](/docs/guides/frames) — how the frame tree works and what `InAllFrames` actually iterates.
- API reference: [Go `Wait`](/docs/api-reference/go#Wait) · [TS `wait` / `waitAny`](/docs/api-reference/ts#wait).
- Timeout diagnostics: [Go `WaitError`](/docs/api-reference/go#WaitError) · [TS `WaitError`](/docs/api-reference/ts#WaitError) and its per-condition `WaitConditionStatus`.

→ Continue: [Loading pages](/docs/guides/loading)
