<!--
  url: https://browserscale.cloud/docs/concepts
  title: Core concepts
  description: Sessions, locators, frames, the flat frame model, and the action / wait / read loop that every browserscale script is built on.
-->

# Core concepts

Three nouns carry every browserscale script: **`CloudBrowser`**, **`Locator`**, and
**`Wait`**. Once those click into place — plus a one-paragraph mental
model of the session lifecycle — the rest of the SDK is just method
discovery in the API reference.

> **TL;DR**
>
> - `CloudBrowser` is your live handle to a remote browser session.
> - `Locator` is a declarative *"which element / under which conditions"*; the same object works for both waits and actions.
> - `Wait` blocks until a Locator matches — but you rarely need it because every action auto-waits.
> - The session is billed per minute (rounded up), and credits are deducted upfront when you rent.

## CloudBrowser

`CloudBrowser` is the gRPC-backed handle you receive from `RentBrowser`.
Think of it as a small, stateful client to one specific browser context:
every method on it is one round-trip to the server, and the order matters
(navigate first, click after).

**Go:**

```go
browser, err := browserscale.RentBrowser(ctx, cfg)
if err != nil { log.Fatal(err) }
defer browser.Close()

_, _ = browser.Navigate(ctx, "https://browserscale.cloud", 0)
```

**TypeScript:**

```ts
const browser = await rentBrowser(cfg);
try {
    await browser.navigate("https://browserscale.cloud");
} finally {
    await browser.stopBrowser();
}
```

The handle is **not** safe to share across goroutines or async contexts —
every call serialises on the underlying gRPC stream, so concurrent use
just queues. If you need real parallelism, rent a second session.

## Locator

A `Locator` answers *"which element on the page do I mean?"*. The same
Locator object is what you pass to `Click`, `Fill`, `Drag` — and what you
pass to `Wait`. One declarative thing, two uses.

### Four constructors

| Constructor | What you give it | When to reach for it |
| --- | --- | --- |
| `CSS(selector)` | A CSS selector string | Your default — almost every locator you ever write |
| `JS(expression)` | A JS expression that returns an Element or a truthy value | When CSS can't describe it (text content, computed state, custom logic) |
| `Node(backendNodeId)` | A node id returned by a previous call | Acting on something `Wait` / `GetDOM` / `Evaluate` already found, without re-resolving |
| `At(x, y)` | Viewport coordinates in CSS pixels | Pixel-precise hits (canvas, captcha cells) — accepted by `Click` and `MoveTo` only |

### You usually need nothing else

The thing to internalise is that **`CSS(...)` is already complete**. When
you write it, the SDK quietly attaches two sensible defaults: the element
must be visible, and its bounding rect must hold still for 500 ms before
the wait matches. That covers the 95 % case — buttons that fade in, modals
that slide, list items that lazy-load — without you thinking about it.

**Go:**

```go
// 95% case — nothing extra. Auto-waits for visible + steady.
_, _ = browser.Click(ctx, browserscale.CSS("button.submit"))
```

**TypeScript:**

```ts
// 95% case — nothing extra. Auto-waits for visible + steady.
await browser.click(css("button.submit"));
```

You only reach for modifiers when those defaults need to bend:

**Go:**

```go
// Element is deliberately offscreen (display:none) — opt out of visible.
_, _ = browser.Click(ctx, browserscale.CSS(".lazy-mounted").Visible(false))

// You want to fire as soon as it appears — skip the 500ms steady wait.
_, _ = browser.Click(ctx, browserscale.CSS(".banner").Steady(0))

// Search inside every frame instead of just the main document.
_, _ = browser.Click(ctx, browserscale.CSS("button.consent").InAllFrames())
```

**TypeScript:**

```ts
// Element is deliberately offscreen (display:none) — opt out of visible.
await browser.click(css(".lazy-mounted").visible(false));

// You want to fire as soon as it appears — skip the 500ms steady wait.
await browser.click(css(".banner").steady(0));

// Search inside every frame instead of just the main document.
await browser.click(css("button.consent").inAllFrames());
```

The full modifier set is `.Visible / .Steady / .InFrame / .InAllFrames`,
all chainable in any order. The Targeting elements guide goes deep on
each one.

## Wait

Every element action already waits for its target — `Click(ctx, CSS(...))`
does not need a separate `Wait` call in front of it. You reach for `Wait`
directly only when you want to **observe** a state change without acting:
confirm a modal appeared, race two possible outcomes, or block on a JS
condition that is not even an element.

The Go API is variadic — conditions and options interleave freely. The TS
API splits the two: `wait(condition, opts?)` for one, `waitAny([...], opts?)`
for several.

**Go:**

```go
// Either a success banner or a redirect URL — whichever comes first.
res, err := browser.Wait(ctx,
    browserscale.CSS(".banner-success"),
    browserscale.JS("location.pathname === '/welcome'"),
    browserscale.Timeout(5000),
)
if err != nil { log.Fatal(err) }
fmt.Println("matched condition index:", res.Index)
```

**TypeScript:**

```ts
// Either a success banner or a redirect URL — whichever comes first.
const res = await browser.waitAny(
    [css(".banner-success"), js("location.pathname === '/welcome'")],
    { timeoutMs: 5000 },
);
console.log("matched condition index:", res.index);
```

When several conditions are passed, the **first one to match wins** —
the others are abandoned the instant a winner is found. `WaitResult.Index`
(`.index` in TS) tells you which entry won, so you can branch on it.

## Session lifecycle

```
   RentBrowser ──▶  drive (navigate, click, …)  ──▶  StopBrowser / Close
        │
        └── credits deducted upfront, per minute, rounded up
            (refunded automatically if the rent itself fails)
```

Three things can end a session:

1. **You call `StopBrowser`** (or `Close` in Go) — the canonical, clean
   shutdown. Always defer this so a panic mid-script still releases the
   session.
2. **The `rentDuration` you set on `NewBrowserConfig` expires** —
   the server tears the session down and any in-flight call returns an
   error. The full reserved window was already paid for at rent time, so
   stopping early does *not* refund the unused minutes.
3. **The process holding the handle dies without releasing it** —
   harmless on our side: the server reaps the session on its own after a
   short grace period. You still spent the upfront credits.

The cleanest pattern: `defer browser.Close()` (Go) or
`try { … } finally { await browser.stopBrowser(); }` (TS). It costs
nothing when the script runs to completion and saves a session from
hanging when something blows up.

## The SDK defaults at a glance

A handful of constants govern *"what does browserscale do when I leave the option
off?"*. They are exported so you can read them, and overridable per call:

| Default | Value | Where it applies |
| --- | --- | --- |
| `DefaultWaitTimeoutMs` | 30 000 ms (30 s) | `Wait` and every action's implicit wait |
| `DefaultVisible` | `true` | `CSS` / `JS` element locators require the match to be visible |
| `DefaultSteadyMs` | 500 ms | `CSS` / `JS` element locators require the bounding rect to stop moving for this long |

Translated: *by default, every action waits up to 30 seconds for an
element that is on-screen and not animating.* You rarely need to think
about this — the moment you do, override per locator with
`.Visible(false)` / `.Steady(0)`, or per call with `Timeout(...)`.

## See also

- The [Targeting elements guide](/docs/guides/locators) walks through every Locator constructor and modifier in depth.
- The [Waiting guide](/docs/guides/waiting) covers multi-condition waits, anti-patterns and timeouts.
- The [Go SDK reference](/docs/api-reference/go) and [TypeScript SDK reference](/docs/api-reference/ts) list every method on `CloudBrowser`.

→ Continue: [Targeting elements](/docs/guides/locators)
