<!--
  url: https://browserscale.cloud/docs/guides/locators
  title: Targeting elements
  description: CSS, JavaScript, node-handle and coordinate locators for picking elements on the page in the browserscale Go and TypeScript SDKs.
-->

# Targeting elements

Almost every browserscale method needs to know *which element on the page* you
mean — a button to click, an input to fill, something to wait for. The
answer is always a `Locator`. This guide explains the four ways to
build one, the four modifiers you can apply, and the small set of rules
that govern which one to reach for in practice.

> **TL;DR**
>
> - A `Locator` is one declarative object — the *same* object works as a wait condition and as an action target.
> - Four constructors: `CSS(selector)`, `JS(expression)`, `Node(backendNodeId)`, `At(x, y)`.
> - Four modifiers: `.Visible(...)`, `.Steady(...)`, `.InFrame(...)`, `.InAllFrames()`.
> - Defaults already cover the 95 % case — you mostly write `CSS(...)` and nothing else.

## One object, two uses

Before we dig into the constructors, hold on to the central idea: a
Locator is built once and reused. The same object you pass to `Wait`
is what you pass to `Click`, `Fill`, `Drag` or any other action.

**Go:**

```go
target := browserscale.CSS("button.submit")

// Use it as a wait condition…
_, _ = browser.Wait(ctx, target)

// …then act on it.
_, _ = browser.Click(ctx, target)
```

**TypeScript:**

```ts
const target = css("button.submit");

// Use it as a wait condition…
await browser.wait(target);

// …then act on it.
await browser.click(target);
```

That's the mental model the whole guide leans on. Now the constructors.

## The four constructors

### `CSS(selector)` — your default

Use a standard CSS selector and let the server find the element. This
is the right answer for almost every locator you will ever write.

**Go:**

```go
browserscale.CSS("button.submit")
browserscale.CSS("input[name='email']")
browserscale.CSS("#sidebar > .item:first-child")
```

**TypeScript:**

```ts
css("button.submit");
css("input[name='email']");
css("#sidebar > .item:first-child");
```

A few things to know:

- The server runs `querySelector` (not `querySelectorAll`) — **the first
  match wins**. If you need a specific one of many, narrow the selector.
- Only standard CSS is supported. jQuery-style extensions like
  `:contains("text")` won't work — for text-content matches reach for
  `JS(...)` instead.
- When used as a wait condition, the SDK silently attaches two defaults
  (`Visible(true)`, `Steady(500)`). Those are explained in the modifier
  section below; for now just know that `CSS("button.submit")` already
  means *"a button.submit that is visible and not animating"*.

### `JS(expression)` — when CSS isn't expressive enough

Pass any JavaScript expression. The server evaluates it in the target
frame and looks at what comes back. `JS` has **two modes**, and which
one is active depends on the expression's return value:

1. **Returns a DOM Element** → behaves exactly like `CSS(...)` from
   that point on. Visibility and steady checks apply.
2. **Returns a truthy non-Element value** (boolean, number, string,
   plain object) → matches as soon as the value is truthy. Visibility
   and steady are silent no-ops because there is no element to measure.

Mode 1 is the escape hatch for selectors CSS can't express:

**Go:**

```go
// Click a button by its text content — impossible in plain CSS.
target := browserscale.JS(\`
    Array.from(document.querySelectorAll('button'))
         .find(b => b.textContent.trim() === 'Submit')
\`)
_, _ = browser.Click(ctx, target)
```

**TypeScript:**

```ts
// Click a button by its text content — impossible in plain CSS.
const target = js(\`
    Array.from(document.querySelectorAll('button'))
         .find(b => b.textContent.trim() === 'Submit')
\`);
await browser.click(target);
```

Mode 2 is for waiting on state that has no DOM equivalent:

**Go:**

```go
// Wait for an app-level "I'm ready" flag.
_, _ = browser.Wait(ctx, browserscale.JS("window.__ready === true"))

// Wait for a network request that updated a Redux store.
_, _ = browser.Wait(ctx, browserscale.JS("store.getState().user.id !== null"))
```

**TypeScript:**

```ts
// Wait for an app-level "I'm ready" flag.
await browser.wait(js("window.__ready === true"));

// Wait for a network request that updated a Redux store.
await browser.wait(js("store.getState().user.id !== null"));
```

`JS` is powerful, but it also runs your code in the page. Keep
expressions short and side-effect-free — anything observable from JS
is also observable to the site you are automating.

### `Node(backendNodeId)` — re-use what you already found

Every action and wait returns the `backendNodeId` of the element it
resolved. `Node(id)` lets you act on that *exact* element again without
the server re-running a selector query.

**Go:**

```go
// Open a dropdown, then click the same trigger again to close it.
res, _ := browser.Click(ctx, browserscale.CSS("button.dropdown-trigger"))
// res.BackendNodeId is the exact node we just clicked.
_, _ = browser.Click(ctx, browserscale.Node(res.BackendNodeId))
```

**TypeScript:**

```ts
// Open a dropdown, then click the same trigger again to close it.
const res = await browser.click(css("button.dropdown-trigger"));
// res.backendNodeId is the exact node we just clicked.
await browser.click(node(res.backendNodeId));
```

Two important caveats:

- **Action-only.** Passing a `Node(...)` to `Wait` is rejected at send
  time with a clear error — wait conditions need a selector or a JS
  expression so the page has something to *watch*.
- **The id is per-document.** Navigate to a new URL and old
  backendNodeIds become stale. Re-resolve after every navigation.

### `At(x, y)` — pixel coordinates

Sometimes the thing you want to click is not addressable through the
DOM at all: a canvas pixel, a captcha grid cell, an HTML5 game
element. `At(x, y)` skips element resolution entirely and dispatches
the action at the given **viewport-relative CSS-pixel coordinates**.

**Go:**

```go
// Click somewhere on a canvas.
_, _ = browser.Click(ctx, browserscale.At(420, 580))
```

**TypeScript:**

```ts
// Click somewhere on a canvas.
await browser.click(at(420, 580));
```

The catch: `At` is only accepted by **`Click` and `MoveTo`**. Anything
else (`Drag`, `Fill`, `ScrollTo`, `Select…`) needs a real element so
the server can scroll it into view first. Trying to pass `At` to one
of those returns an error before the request ever leaves your machine.

## Which constructor goes where

Quick reference for "can I use *this* locator with *that* method?".

| Constructor | `Wait` | `Click`, `MoveTo` | `Fill`, `Select…`, `Drag`, `ScrollTo` |
| --- | --- | --- | --- |
| `CSS(...)` | yes | yes | yes |
| `JS(...)` | yes | yes | yes |
| `Node(...)` | **no** — nothing to watch | yes | yes |
| `At(...)` | **no** — needs an element | yes | **no** — needs an element |

The `no` cells are validated client-side before the request goes out,
so you get an immediate, descriptive error instead of a confusing
server-side failure.

## The four modifiers

Each modifier is a method on the Locator that returns the modified
Locator. They are chainable in any order. There is a subtle but
important difference in how they behave in Go vs TS — covered at the
end of this section.

### `Visible(bool)` — wait-only

When the Locator is used as a wait condition, the matched element must
be **visible** to count: not `display:none`, not `visibility:hidden`,
non-zero size. This is the default and is what you want almost always
— without it, `Wait` happily returns on hidden elements that the user
can't even see.

Pass `false` when you specifically need to wait for a hidden element:

**Go:**

```go
// Wait for an off-screen container to exist (display:none for now).
_, _ = browser.Wait(ctx, browserscale.CSS("#hidden-form").Visible(false))
```

**TypeScript:**

```ts
// Wait for an off-screen container to exist (display:none for now).
await browser.wait(css("#hidden-form").visible(false));
```

`Visible` has **no effect on actions** — actions never check visibility
before dispatching. If a `Click` lands on something invisible, that
isn't the visibility check's job; it's the page's problem.

### `Steady(ms)` — wait-only

Real pages animate. Banners slide in, modals fade, lists lazy-load
their layout, iframes get pushed around when an ad finishes rendering.
A wait that fires the moment the element appears can dispatch a click
*while the target is mid-flight*.

`Steady(ms)` says *"don't match until the element's bounding rect
hasn't moved for this many milliseconds"*. Default is 500 ms, which
covers most CSS animations and pulled-in iframes.

**Go:**

```go
// A slow-settling CTA — wait a full second of stability before matching.
_, _ = browser.Wait(ctx, browserscale.CSS("button.cta").Steady(1000))

// Plain text that never animates — match the instant it appears.
_, _ = browser.Wait(ctx, browserscale.CSS(".status-line").Steady(0))
```

**TypeScript:**

```ts
// A slow-settling CTA — wait a full second of stability before matching.
await browser.wait(css("button.cta").steady(1000));

// Plain text that never animates — match the instant it appears.
await browser.wait(css(".status-line").steady(0));
```

Two things to remember:

- **Wait-only**, like `Visible`. Actions never re-measure.
- **No-op on `JS` expressions that return a non-Element value** —
  there is no bounding rect to measure for a boolean.

### `InFrame(frameId)` — wait and actions

Every page is a tree of frames. By default, both wait conditions and
actions look only in the **main frame**. `InFrame(id)` scopes the
Locator to a specific frame instead — you get the id from `GetPages()`
or from a previous result's `frameId` field.

**Go:**

```go
pages, _ := browser.GetPages(ctx)
iframeId := pages[0].FrameTree.Children[0].FrameId

_, _ = browser.Click(ctx, browserscale.CSS("button.accept").InFrame(iframeId))
```

**TypeScript:**

```ts
const pages = await browser.getPages();
const iframeId = pages[0].frameTree.children[0].frameId;

await browser.click(css("button.accept").inFrame(iframeId));
```

For actions, there is also a `XxxOpts.InFrame` per call. The
**call-level option wins** over the Locator's own frame, which makes it
easy to reuse a base Locator across frames without rebuilding it.

### `InAllFrames()` — wait and actions

When you don't know which frame the element is in (a consent banner
that might be in the main page or in any ad iframe, a captcha that
mounts wherever), use `InAllFrames()`. The server searches every
frame on the page and returns the first match.

**Go:**

```go
// "Accept all" appears somewhere — main page or ad iframe.
_, _ = browser.Click(ctx, browserscale.CSS("button.consent").InAllFrames())
```

**TypeScript:**

```ts
// "Accept all" appears somewhere — main page or ad iframe.
await browser.click(css("button.consent").inAllFrames());
```

`InAllFrames()` is just shorthand for `InFrame(AllFrames)`. Use
whichever reads better in your code.

### A subtle Go vs TS difference

Modifiers feel chainable in both languages, but the implementations
differ:

**Go:**

```go
// Go modifiers mutate the Locator in place and return the same pointer.
base := browserscale.CSS(".banner")
base.Steady(0)            // <-- mutates base!
// base now has steady=0
```

**TypeScript:**

```ts
// TS modifiers return a NEW Locator. The original is untouched.
const base = css(".banner");
const fast = base.steady(0);
// base still has steady=500, fast has steady=0
```

The TS API is safer for sharing a base Locator across calls. The Go
API is faster (one fewer allocation per chain). The practical takeaway:
in Go, **don't reuse a Locator after chaining modifiers** unless you
mean to apply them everywhere.

## Recipes

A handful of patterns that come up over and over:

**Click a button by its visible text.** CSS can't, so use `JS`:

**Go:**

```go
target := browserscale.JS(\`
    Array.from(document.querySelectorAll('button'))
         .find(b => b.textContent.trim() === 'Sign in')
\`)
_, _ = browser.Click(ctx, target)
```

**TypeScript:**

```ts
const target = js(\`
    Array.from(document.querySelectorAll('button'))
         .find(b => b.textContent.trim() === 'Sign in')
\`);
await browser.click(target);
```

**Wait for an arbitrary JS condition.** No element involved:

**Go:**

```go
_, _ = browser.Wait(ctx, browserscale.JS("performance.timing.loadEventEnd > 0"))
```

**TypeScript:**

```ts
await browser.wait(js("performance.timing.loadEventEnd > 0"));
```

**Wait for an element that is in the DOM but hidden.** Classic case: a
hidden CSRF input that the page injects after a network round-trip.
The default visibility check would never match, so opt out:

**Go:**

```go
_, _ = browser.Wait(ctx, browserscale.CSS("input[name='csrf_token']").Visible(false))
```

**TypeScript:**

```ts
await browser.wait(css("input[name='csrf_token']").visible(false));
```

**Click the same element twice without re-resolving.** Use the
returned `backendNodeId`:

**Go:**

```go
r, _ := browser.Click(ctx, browserscale.CSS("button.toggle"))
_, _ = browser.Click(ctx, browserscale.Node(r.BackendNodeId))
```

**TypeScript:**

```ts
const r = await browser.click(css("button.toggle"));
await browser.click(node(r.backendNodeId));
```

**Click a specific captcha cell.** Use coordinates from
`InspectAtPosition` or your own image-recognition pipeline:

**Go:**

```go
_, _ = browser.Click(ctx, browserscale.At(420, 580))
```

**TypeScript:**

```ts
await browser.click(at(420, 580));
```

**Click into any frame.** The element will be found wherever it lives:

**Go:**

```go
_, _ = browser.Click(ctx, browserscale.CSS("button.dismiss").InAllFrames())
```

**TypeScript:**

```ts
await browser.click(css("button.dismiss").inAllFrames());
```

## Gotchas to keep in mind

A short list of things that cause confusion on day one:

- **`Node()` / `At()` as wait conditions** are rejected before the
  request leaves the SDK. Wait needs something to watch — a selector or
  a JS expression — and neither of these provides one.
- **`Visible` / `Steady` on actions are silently ignored.** They only
  affect wait. If you want to gate an action on visibility, do an
  explicit `Wait` first, then act.
- **JS expressions returning non-Element values** make `Visible` and
  `Steady` no-ops. The condition matches as soon as the value is
  truthy — there is nothing to measure.
- **Locator equality** isn't a thing — two `CSS("button")` calls
  produce two distinct Locator objects with equal contents. Compare on
  result fields (`backendNodeId`, `bounds`) instead.
- **Building the `Locator` struct/class by hand** is discouraged. Go
  has unexported fields and TS has a private constructor specifically
  to push you toward the factory functions.

## See also

- The [Waiting guide](/docs/guides/waiting) covers multi-condition races, custom timeouts, and why you need to wait explicitly before acting.
- The [Frames & iframes guide](/docs/guides/frames) goes deep on the frame tree and OOPIF limits.
- API reference: [Go locators](/docs/api-reference/go#CSS) · [TS locators](/docs/api-reference/ts#css).

→ Continue: [Waiting](/docs/guides/waiting)
