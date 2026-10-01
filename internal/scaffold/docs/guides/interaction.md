<!--
  url: https://browserscale.cloud/docs/guides/interaction
  title: Interacting with the page
  description: Click, fill, hover, scroll, drag and select. Actions find their target, wait for it to settle, check the pixel before pressing and name whatever blocked them.
-->

# Interaction

Interaction is where a script stops describing the page and starts
changing it. browserscale's action surface is deliberately small — eight verbs
do everything from "click that button" to "drag a slider 120 px right".
The point of this guide is to give you a feel for *which verb fits
what*, what each one does for you behind the scenes, and the handful
of options that come up often enough to be worth memorising.

> **TL;DR**
>
> - Every action takes a `Locator` (see [Targeting elements](/docs/guides/locators)). `Click`, `Fill` and `Drag` acquire their target themselves: re-located for up to 5 s, scrolled into view, held until it stops moving. That covers an element rendering a moment late; after a `Navigate` or a submit, [`Wait`](/docs/guides/waiting) for the next page first, with a timeout for that step.
> - Pre-flight: the cursor moves there along a human-like path, and the exact point is verified to belong to the element before the button goes down. If something covers it, the pointer re-aims at an exposed part once; if it is still covered, the action refuses and names the blocker instead of pressing whatever lies on top.
> - Go splits the bare and the customised variant into two methods (`Click` / `ClickWith`). TypeScript keeps one method and an optional `opts` argument.
> - Three keyboard-level escape hatches sit alongside the element verbs: `InsertText` (paste at caret), `PressKey` / `ReleaseKey` (raw down/up events).
> - A failed action tells you *why*: a typed `ClickError` / `FillError` / … (carrying an `occluder`) when the element was found but the gesture was blocked, versus a plain `ELEMENT_NOT_FOUND` / `TIMEOUT` when it never showed up.

## The shared shape

Every element action follows the same pattern, so once you know one
you know all of them:

1. You pass a `Locator` — built with `CSS(...)` / `JS(...)` /
   `Node(...)` / `At(...)`. The locator's *modifiers* (`.InFrame(...)`,
   `.InAllFrames()`) are honoured.
2. The browser resolves the element — for `Click`, `Fill` and `Drag`
   it keeps re-locating it for up to 5 s if it is not there yet —
   **scrolls it into view** (walking nested scroll containers and
   out-of-process iframe chains) and waits until its bounds stop
   moving.
3. It **moves the cursor** along a human-like path to a random point
   inside the element and verifies that the point really belongs to
   the element, across frame and process boundaries. Covered, it
   re-aims at the exposed part once; still covered, it refuses with a
   typed error naming the blocker. It never presses whatever happens
   to lie on top.
4. The gesture (click, fill, drag…) fires once.
5. You get back an `ElementResult` (or a `DragResult` / a
   `SelectOptionResult`) carrying the resolved `frameId`,
   `backendNodeId`, post-scroll `isVisible`, the element's `bounds`
   and the root-viewport coordinates (`rootX` / `rootY`) where the
   gesture actually landed.

The naming pattern differs across the two SDKs. Whenever an action
has options, Go ships **two methods** — the bare one and a
`…With(opts)` variant — so you don't pay for an empty struct in the
common case. TypeScript keeps the single method and tucks the options
behind a trailing optional argument.

**Go:**

```go
// Bare call — no options.
_, _ = browser.Click(ctx, browserscale.CSS("button.submit"))

// Customised call — same target, right double-click.
_, _ = browser.ClickWith(ctx, browserscale.CSS("li.menu"), browserscale.ClickOpts{
    Button:     "right",
    ClickCount: 2,
})
```

**TypeScript:**

```ts
// Bare call — no options.
await browser.click(css("button.submit"));

// Customised call — same target, right double-click.
await browser.click(css("li.menu"), { button: "right", clickCount: 2 });
```

## Click

The workhorse. By default: find the element within 5 s, scroll it
into view, wait for it to hold still, move there on a hand-shaped
path, verify the point, then a full `mouseDown` + `mouseUp` at a
randomised point inside the element's bounding rect.

`ClickOpts` covers the three things you might want to vary:

- **`Button`** — `"left"` (default), `"right"`, `"middle"`.
- **`ClickCount`** — `1` (default) or `2` for a double-click.
- **`Action`** — `"click"` (default, full press + release),
  `"press"` (mouseDown only), `"release"` (mouseUp only — no mouse
  movement, fires at the current cursor position).

The `press` / `release` pair is the escape hatch for long-press
gestures, custom drag implementations, or any flow where you need to
hold the button down across other actions:

**Go:**

```go
// Long-press: down, hold while doing something else, up.
_, _ = browser.ClickWith(ctx, browserscale.CSS(".thumb"), browserscale.ClickOpts{Action: "press"})
// ... do other things while the button is held ...
_, _ = browser.ClickWith(ctx, browserscale.At(0, 0), browserscale.ClickOpts{Action: "release"})
```

**TypeScript:**

```ts
// Long-press: down, hold while doing something else, up.
await browser.click(css(".thumb"), { action: "press" });
// ... do other things while the button is held ...
await browser.click(at(0, 0), { action: "release" });
```

Note that `release` ignores any coordinates you pass and fires
at wherever the cursor currently is — exactly the behaviour you need to
close out a `press` started elsewhere on the page.

`Click` is the one action that also accepts `At(x, y)` (along with
`MoveTo`). Coordinate clicks are how you reach canvas elements,
captcha tiles, and HTML5-game targets that aren't addressable via
the DOM.

## Fill

Anywhere you'd type into an input: `<input>`, `<textarea>`,
`contenteditable` divs. Under the hood: the same target acquisition as
`Click`, a click to focus, then **character-by-character key events**
on the keyboard layout of the session's region, with human-like timing.

`Fill` is bound to its target. If something else takes focus half way
through, the rest of the value is never typed into it: the browser
tries to re-focus the field and otherwise fails with `focus_stolen`,
naming the element that took focus. For one-time-code boxes that move
focus themselves on every digit, use the deliberately loose `Type`.

**`Fill` appends by default.** It does not wipe the field first — what
you pass in is typed on top of whatever is already there. That matches
the common case (empty input → type the value) and lets you add to
existing content without ceremony. To overwrite, pass `ClearFirst` /
`clearFirst`, which fires Ctrl+A, Delete before typing:

**Go:**

```go
// Default: type on top of whatever is in the field.
_, _ = browser.Fill(ctx, browserscale.CSS("input[name=email]"), "user@example.com")

// Wipe first, then type the fresh value.
_, _ = browser.FillWith(ctx, browserscale.CSS("input[name=email]"), "user@example.com", browserscale.FillOpts{
    ClearFirst: true,
})
```

**TypeScript:**

```ts
// Default: type on top of whatever is in the field.
await browser.fill(css("input[name=email]"), "user@example.com");

// Wipe first, then type the fresh value.
await browser.fill(css("input[name=email]"), "user@example.com", { clearFirst: true });
```

`Fill` only accepts element locators — `At(x, y)` is rejected because
typing needs an actual focus target.

Because every keystroke goes through a real `keydown` / `keypress` /
`keyup` cycle, any JS handler the page has on the input (autocomplete,
live validation, inline search…) fires exactly as if a person typed
the value. That's the whole point — but it also means `Fill` is the
slowest action in browserscale. For pasting large blocks of text without firing
per-key events, jump straight to `InsertText` (further down).

## MoveTo

Move the mouse cursor over an element (or to coordinates) without
clicking. The same scroll-into-view and human-like cursor path as
`Click`, just stopping at the hover.

Useful for triggering CSS / JS hover states (dropdown menus,
tooltips), warming up a page that only reveals an action on
`mouseover`, or staging a cursor before a `Click(at(...))`:

**Go:**

```go
// Hover to reveal a dropdown.
_, _ = browser.MoveTo(ctx, browserscale.CSS("nav .menu-trigger"))
_, _ = browser.Wait(ctx, browserscale.CSS("nav .submenu"))
_, _ = browser.Click(ctx, browserscale.CSS("nav .submenu li:first-child"))
```

**TypeScript:**

```ts
// Hover to reveal a dropdown.
await browser.moveTo(css("nav .menu-trigger"));
await browser.wait(css("nav .submenu"));
await browser.click(css("nav .submenu li:first-child"));
```

Like `Click`, `MoveTo` also accepts `At(x, y)`.

## ScrollTo

Bring an element into the viewport without clicking it. browserscale actions
already scroll into view as part of their pre-flight, so `ScrollTo`
exists for the two cases that don't:

- You want the element visible for a `GetDOM` / `GetObservation` /
  `Evaluate` pass that won't trigger a scroll on its own.
- You're chaining multiple `At(x, y)` coordinate clicks and need to
  position the page relative to known coordinates first.

**Go:**

```go
_, _ = browser.ScrollTo(ctx, browserscale.CSS("#footer"))
// Footer is now in view — go inspect, evaluate, or click coordinates.
```

**TypeScript:**

```ts
await browser.scrollTo(css("#footer"));
// Footer is now in view — go inspect, evaluate, or click coordinates.
```

The server walks **nested scroll containers** (the inner `div` that
actually scrolls inside a paginated UI) and **out-of-process iframe
chains** (so an element deep inside an OOPIF still gets brought into
the root viewport) automatically. You don't have to think about it.

`At(x, y)` is rejected here — scrolling needs an actual element to
target.

## Drag

Two variants depending on whether you want to drop *relative to the
pickup* (slider handles, range inputs) or *at a fixed page position*
(reordering cards, file-tree nodes).

**Go:**

```go
// Drag by an offset — slider 120 px to the right.
_, _ = browser.DragBy(ctx, browserscale.CSS(".slider .handle"), 120, 0)

// Drag to absolute root-viewport coordinates — reorder a card.
_, _ = browser.DragTo(ctx, browserscale.CSS(".card.draggable"), 800, 400)
```

**TypeScript:**

```ts
// Drag by an offset — slider 120 px to the right.
await browser.dragBy(css(".slider .handle"), 120, 0);

// Drag to absolute root-viewport coordinates — reorder a card.
await browser.dragTo(css(".card.draggable"), 800, 400);
```

Both variants go through the same gesture: the handle is acquired like
a `Click` target, then mouse-move to the pickup point inside it,
`mouseDown`, drag along a human-like path to the target, `mouseUp`. The `DragResult` carries *both* endpoints —
`startX` / `startY` for the pickup, `endX` / `endY` for the drop — so
you can verify where the gesture actually started and ended.

`Drag` only accepts element locators. To drag between specific
coordinate pairs, fall back to two `Click(At(...))` calls in
`press` / `release` mode.

## Select (for `<select>` elements)

Three variants depending on what you know about the option to pick.
Same `<select>` Locator, different *option key*:

**Go:**

```go
// Zero-based index.
_, _ = browser.SelectByIndex(ctx, browserscale.CSS("select#country"), 2)

// Match the option's value attribute.
_, _ = browser.SelectByValue(ctx, browserscale.CSS("select#country"), "DE")

// Match the option's visible text (trimmed, exact match).
_, _ = browser.SelectByText(ctx, browserscale.CSS("select#country"), "Germany")
```

**TypeScript:**

```ts
// Zero-based index.
await browser.selectByIndex(css("select#country"), 2);

// Match the option's value attribute.
await browser.selectByValue(css("select#country"), "DE");

// Match the option's visible text (trimmed, exact match).
await browser.selectByText(css("select#country"), "Germany");
```

Same pre-flight as every other action: the `<select>` is scrolled
into view and the cursor animates over to it along a human-like path
— picking an option goes through the same motions a real user would.
Once the option is chosen, the standard `input` and `change` events
fire so the page's listeners run exactly as if a person had picked
it.

Opt out of the events for the (rare) case where you want to mutate
state silently — note that Go's flag is `NoEvents=true` (negative
polarity), while TS uses `fireEvents=false`:

**Go:**

```go
// Silent — no input/change events.
_, _ = browser.SelectByIndexWith(ctx, browserscale.CSS("select#hidden"), 0, browserscale.SelectOpts{
    NoEvents: true,
})
```

**TypeScript:**

```ts
// Silent — no input/change events.
await browser.selectByIndex(css("select#hidden"), 0, { fireEvents: false });
```

All three variants return a `SelectOptionResult` with the resolved
`SelectedIndex`, `SelectedValue` and `SelectedText` — useful as a
sanity check that the right option was actually chosen.

## Keyboard fallbacks

For the cases that don't fit into a single element action, three
keyboard-level primitives target whatever currently has focus:

### `InsertText` — fast paste

Commits the entire string at once via the browser's IME path. **No
per-character key events fire** — which is what you want for big text
blocks where you don't need to trigger autocomplete or live
validation:

**Go:**

```go
// Focus the field first…
_, _ = browser.Click(ctx, browserscale.CSS("textarea.bio"))
// …then paste the whole block in one shot.
_ = browser.InsertText(ctx, "Lorem ipsum dolor sit amet, …")
```

**TypeScript:**

```ts
// Focus the field first…
await browser.click(css("textarea.bio"));
// …then paste the whole block in one shot.
await browser.insertText("Lorem ipsum dolor sit amet, …");
```

Because no keyboard events fire, anything the page listens for on
`keydown` / `keyup` will *not* see this input. That's the trade-off:
fast and silent, or slow and observable.

### `PressKey` / `ReleaseKey` — raw key events

For keyboard shortcuts (Ctrl+A, Ctrl+S, Enter to submit) and any
flow where you need a key event to be the actual signal. Both fire a
single event — pair them for a full press cycle.

The modifier bitmask is **Alt=1, Ctrl=2, Meta=4, Shift=8**. Combine
with `|` for multi-modifier shortcuts.

**Go:**

```go
// Ctrl+A on the focused element.
_ = browser.PressKey(ctx, "a", "KeyA", 2, 0)
_ = browser.ReleaseKey(ctx, "a", "KeyA", 2, 0)

// Just an Enter — to submit a form without clicking the button.
_ = browser.PressKey(ctx, "Enter", "Enter", 0, 0)
_ = browser.ReleaseKey(ctx, "Enter", "Enter", 0, 0)
```

**TypeScript:**

```ts
// Ctrl+A on the focused element.
await browser.pressKey("a", { code: "KeyA", modifiers: 2 });
await browser.releaseKey("a", { code: "KeyA", modifiers: 2 });

// Just an Enter — to submit a form without clicking the button.
await browser.pressKey("Enter");
await browser.releaseKey("Enter");
```

The `key` value follows the DOM `KeyboardEvent.key` standard
(`"Enter"`, `"ArrowLeft"`, `"a"`, …). `code` follows
`KeyboardEvent.code` (`"Enter"`, `"KeyA"`, `"ArrowLeft"`, …) and
falls back to `key` when omitted.

## Auto-behaviours, at a glance

A quick recap of what the server does for you so you don't have to:

| Action | Scrolls into view | Animates cursor | Fires events |
| --- | --- | --- | --- |
| `Click`, `Fill` | yes | yes (human-like path) | full DOM event stream |
| `MoveTo` | yes | yes (human-like path) | mouseover, mouseenter |
| `Drag*` | yes (pickup) | yes (pickup → drop) | mousedown / mousemove / mouseup |
| `ScrollTo` | yes (the whole point) | — | scroll events on the container |
| `Select*` | yes | yes (human-like path) | input + change (unless suppressed) |
| `InsertText` | — | — | input (no key events) |
| `PressKey` / `ReleaseKey` | — | — | keydown / keyup |

`Click`, `Fill` and `Drag` additionally re-locate their target for up
to 5 s, wait for it to settle and verify the point before pressing.
A manual `ScrollTo` before a `Click` is never needed. A `Wait` is,
wherever the page first has to load or change — see
[Waiting](/docs/guides/waiting#when-to-wait).

## When an action fails

Actions fail in two distinct ways, and the SDKs let you tell them
apart so you can recover intelligently instead of blindly retrying:

- **The element was never reached.** The locator matched nothing
  within the timeout, the frame was gone, the page died, or the
  locator itself was malformed. This surfaces with a code like
  `ELEMENT_NOT_FOUND`, `FRAME_NOT_FOUND`, `TIMEOUT`, `PAGE_NOT_ALIVE`
  or `INVALID_LOCATOR`. Nothing on the page was touched.
- **The element was there, but the gesture couldn't land.** The
  classic case: another element — a cookie banner, a modal overlay —
  was still covering your target at the exact point the gesture would
  fire, even after the pointer re-aimed.
  This surfaces as a **typed error** carrying the detail you need to
  recover: `ClickError` with an `occluder` describing the blocker, and
  the matching `FillError` / `DragError` / `MoveError` / `ScrollError`
  / `SelectOptionError` for the other verbs.

The typed errors are the same shape in both SDKs. Go returns them as a
normal `error` you unwrap with `errors.As`; TypeScript throws them as
classes you match with `instanceof`:

**Go:**

```go
_, err := browser.Click(ctx, browserscale.CSS("button.submit"))
var clickErr *browserscale.ClickError
if errors.As(err, &clickErr) {
    // The button was found but something covered it.
    if clickErr.Occluder != nil {
        log.Printf("blocked by <%s id=%q>", clickErr.Occluder.TagName, clickErr.Occluder.ID)
        // e.g. close that overlay, then retry the click.
    }
} else if err != nil {
    // ELEMENT_NOT_FOUND, TIMEOUT, … — the element was never reached.
}
```

**TypeScript:**

```ts
try {
  await browser.click(css("button.submit"));
} catch (e) {
  if (e instanceof ClickError) {
    // The button was found but something covered it.
    console.log("blocked by", e.occluder?.tagName, e.occluder?.id);
    // e.g. close that overlay, then retry the click.
  } else if (e instanceof BrowserScaleError) {
    // ELEMENT_NOT_FOUND, TIMEOUT, … — the element was never reached.
  }
}
```

The `occluder` is a full `OccluderInfo` — tag name, id, class, text
and its own `frameId` / `backendNodeId` — so you can locate and clear
the blocker (find and click its close button) before retrying, rather
than firing the same doomed click again. It also carries the blocker's
computed visibility (`pointerEvents`, `visibility`, `opacity`, `zIndex`,
`position` and `hittableWhileInvisible`), so you can tell a transparent
pass-through layer that shouldn't have intercepted at all from one that
genuinely swallows the click and must be removed. `position` of
`"fixed"` / `"sticky"` means the blocker is pinned (by itself or an
ancestor) and stays put no matter where the pointer goes — clear it by
scrolling the target out from under it; ordinary overlays often collapse
once the pointer leaves. `Fill` and `Drag` wrap the same core failure:
their errors expose the underlying `ClickError` (as `clickError`) plus a
partial `result`, so the occluder is reachable no matter which verb
tripped over it.

## Gotchas

- **Actions wait for their target, not for your outcome.** `Click`
  finds the button; it does not know whether the next page should be
  the dashboard or an error. Wait for that afterwards — see
  [Waiting](/docs/guides/waiting). For overlays that may or may not
  appear, register a [reaction](/docs/guides/reactions) once instead of
  handling them before every click.
- **Validation is client-side, not server-side.** `Drag` / `Fill` /
  `Select*` / `ScrollTo` reject `At(x, y)` before the request ever
  leaves your machine — you get an immediate, descriptive error rather
  than a confusing server-side failure.
- **`Select*` flag polarity differs across SDKs.** Go uses negative
  polarity (`NoEvents: true` opts out of `change` / `input`), TS uses
  positive (`fireEvents: false` does the same). Defaults are identical
  — only the spelling differs.
- **`InsertText` and `PressKey` need focus.** Neither targets an
  element — they go to whatever currently has focus on the page. Do a
  `Click` first if you need a specific input to receive them.
- **`release` ignores its target.** A `Click` with
  `Action: "release"` fires at the current cursor position regardless
  of the locator you pass — that's the whole point, so a long-press
  started elsewhere can be closed cleanly.

## See also

- [Targeting elements](/docs/guides/locators) — the constructors and modifiers every action accepts.
- [Waiting](/docs/guides/waiting) — waiting for the outcome of an action.
- [Reactions](/docs/guides/reactions) — cookie banners and modals, handled by the browser.
- API reference: [Go interaction methods](/docs/api-reference/go#Click) · [TS interaction methods](/docs/api-reference/ts#click).
- Error types: [Go `ClickError`](/docs/api-reference/go#ClickError) · [TS `ClickError`](/docs/api-reference/ts#ClickError) and their `OccluderInfo`.

→ Continue: [Reactions](/docs/guides/reactions)
