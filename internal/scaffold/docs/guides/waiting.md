<!--
  url: https://browserscale.cloud/docs/guides/waiting
  title: Waiting
  description: Wait after every navigation and page change, with a timeout per step. Race success, error and challenge states, get the matched element and frame, and a per-condition diagnosis on timeout.
-->

# Waiting

Pages take time. Network round-trips, JS bundles, animations, third-party
widgets, captchas — there is almost always a moment between *"I told the
browser to do something"* and *"the page is in the state I need"*. The
job of `Wait` is to bridge that gap honestly: return the moment what you
described holds, tell you which of your conditions it was — or give up
with a diagnosis of how close each one came.

> **TL;DR**
>
> - `Wait` marks every point where the page has to get somewhere: after a `Navigate`, after a submit, after any step that loads a new page or view. Wait for the element you need next, with a timeout sized for that step.
> - One condition or several. Race the outcomes a step can have (next page, form error, challenge) and branch on the `Index` that matched.
> - The documents a condition concerns report the moment it holds; nothing re-checks on a timer while the page is idle.
> - Default timeout is 30 s. Override per call with `Timeout(ms)` in Go / `{ timeoutMs }` in TS.
> - `Click`, `Fill` and `Drag` also find their own target, for up to 5 s. That covers an element rendering a moment late, not a page load.
> - On timeout, a `WaitError` says per condition how far it got: `not_found`, `found_hidden`, `found_occluded` (with the element covering it) or `pending_steady`.

## When to wait

A flow is a series of steps, and between most of them the page has
work to do: `Navigate` returns when the document commits, before the
page's JavaScript has built anything; a submitted form goes to a server
that may take ten seconds to answer; a single-page app swaps views
without navigating at all. `Wait` is how a script says *"this is where
the page has to get somewhere, and this is how long it may take"*.

Three situations call for it:

- **After every `Navigate`, and every step that changes the page.**
  Wait for the element the next step needs, with a budget that fits the
  page — a slow sign-up page gets 25 s, not the 5 s an action allows
  itself.
- **When the next step depends on what happened.** After submitting,
  the page can show the next step, an error, a challenge or a login
  modal. Pass all of them, get back the one that happened.
- **Before reading.** An `Evaluate` or `GetObservation` sees the page
  as it is right now; wait for the state you want to read first.

**Go:**

```go
_, _ = browser.Navigate(ctx, "https://signup.example/", 0)

// The page is usable once the email field is there. Give it time.
_, err := browser.Wait(ctx, browserscale.CSS("input[name=email]"), browserscale.Timeout(25000))
if err != nil {
    return err // says whether the field never appeared, stayed hidden or was covered
}
_, _ = browser.Fill(ctx, browserscale.CSS("input[name=email]"), email)
_, _ = browser.Click(ctx, browserscale.CSS("button[type=submit]"))

// Next step, an error, or a challenge. Branch on what happened.
r, err := browser.Wait(ctx,
    browserscale.CSS("input[type=password]"),       // index 0
    browserscale.CSS(".error-message"),             // index 1
    browserscale.CSS("iframe[title*=challenge]").InAllFrames(), // index 2
    browserscale.Timeout(25000),
)
```

**TypeScript:**

```ts
await browser.navigate("https://signup.example/");

// The page is usable once the email field is there. Give it time.
await browser.wait(css("input[name=email]"), { timeoutMs: 25000 });
await browser.fill(css("input[name=email]"), email);
await browser.click(css("button[type=submit]"));

// Next step, an error, or a challenge. Branch on what happened.
const r = await browser.waitAny(
    [
        css("input[type=password]"),              // index 0
        css(".error-message"),                    // index 1
        css("iframe[title*=challenge]").inAllFrames(), // index 2
    ],
    { timeoutMs: 25000 },
);
```

`Navigate → Wait → act` and `act → Wait`: you will write these pairs
a lot, and they make a flow readable too — every `Wait` is a place
where the page changes.

### What actions already do

`Click`, `Fill` and `Drag` do a short wait of their own: they re-locate
the target for up to 5 s, scroll it into view, hold until its bounds
stop moving and verify the point before pressing. So an element that
renders a few frames after the one before it needs nothing extra.

That budget belongs to the action, not to the page. It is short on
purpose, so a wrong selector fails fast. For anything that can take
longer — a page load, a redirect chain, a server answering a form —
put a `Wait` in front with a timeout for that step. (`Fill` also takes
a `TimeoutMs` for its own budget; `Click`'s stays at 5 s.) A timed-out
`Wait` also tells you *how far* the element got, which a click that
gave up cannot.

## Anatomy of a wait

A `Wait` call carries three things:

1. **One or more locator conditions** — anything `CSS(...)` or `JS(...)`
   produces. (`Node` and `At` are rejected client-side; see the
   [Targeting elements guide](/docs/guides/locators) for why.)
2. **A timeout** — how long to wait before giving up.
3. **An implicit frame scope** — main frame by default, or whatever
   the *first condition that has* `.InFrame(...)` / `.InAllFrames()`
   on it specifies (see the frame section below).

Each condition is handed to the documents it concerns, and each
document reports the moment the condition starts holding — so a match
usually arrives within a frame or two, and an idle page costs nothing
while you wait. The *first* condition to match wins. The others are
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

The timeout is per call, so size it for the step it guards. A sign-up
page behind a slow backend or a queue can take well over the default;
a toast after a save should be there within seconds, and waiting 30 s
for it only makes the failure slow.

**Go:**

```go
// A slow page: give it longer than the default.
_, err := browser.Wait(ctx,
    browserscale.CSS("#otp-code"),
    browserscale.Timeout(40000),
)

// A toast: if it is not there in 5 s, it is not coming.
_, err = browser.Wait(ctx,
    browserscale.CSS(".toast-saved"),
    browserscale.Timeout(5000),
)
```

**TypeScript:**

```ts
// A slow page: give it longer than the default.
await browser.wait(css("#otp-code"), { timeoutMs: 40000 });

// A toast: if it is not there in 5 s, it is not coming.
await browser.wait(css(".toast-saved"), { timeoutMs: 5000 });
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

- The first condition to start holding wins; the others are
  abandoned. The order of conditions in the list does not decide
  anything.
- The condition list is open-ended; you can mix `CSS` and `JS`, with
  different `.Visible(...)` / `.Steady(...)` modifiers per locator.
- On timeout you get a typed `WaitError` and no result. Its
  `conditions` array reports, *per locator you passed*, the last state
  it reached: `not_found` (never appeared), `found_hidden` (rendered but
  not visible), `found_occluded` (visible but covered, with the covering
  element as `occluder`) or `pending_steady` (there, but still moving).
  Those four need four different fixes, which is why they are reported
  separately. Go unwraps it with `errors.As`; TypeScript matches it with
  `instanceof WaitError`.

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
   That includes frames created while the wait is running, so there
   is no need to wait for an iframe before waiting for what is in it.

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
- **Letting an action's 5 s cover a page load.** Right after a
  `Navigate` or a submit, a bare `Click` works on a fast day and times
  out on a slow one. Put a `Wait` with a budget for that step in front.
  Within a page that is already there, the action's own wait is enough.
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
  change the page. For something that may or may not appear and just
  needs dismissing — a cookie banner, a newsletter modal — register a
  [reaction](/docs/guides/reactions) instead of waiting for it.
- **`Wait` rejects `Node(...)` and `At(...)` client-side.** These two
  locators don't carry a selector or JS expression for the page to
  watch. The SDK throws before sending.
- **No `WaitOpts.InFrame`.** As covered above, the frame for a wait
  comes from the first condition that has one. There is no separate
  call-level `InFrame` for `Wait` (unlike actions, which do have it).

## See also

- [Targeting elements](/docs/guides/locators) — the locator constructors and modifiers `Wait` accepts.
- [Frames & iframes](/docs/guides/frames) — how the frame tree works and what `InAllFrames` actually iterates.
- [Reactions](/docs/guides/reactions) — for the banner that may or may not appear.
- API reference: [Go `Wait`](/docs/api-reference/go#Wait) · [TS `wait` / `waitAny`](/docs/api-reference/ts#wait).
- Timeout diagnostics: [Go `WaitError`](/docs/api-reference/go#WaitError) · [TS `WaitError`](/docs/api-reference/ts#WaitError) and its per-condition `WaitConditionStatus`.

→ Continue: [Loading pages](/docs/guides/loading)
