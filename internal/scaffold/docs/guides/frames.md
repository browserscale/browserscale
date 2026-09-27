<!--
  url: https://browserscale.cloud/docs/guides/frames
  title: Frames & iframes
  description: browserscale's flat frame model: same-origin and OOPIF frames are uniformly addressable, with offsets exposed on every frame.
-->

# Frames & iframes

Modern pages embed iframes for ads, payment widgets, consent banners,
chat widgets, third-party maps, captchas, social-login popups — the
list keeps growing. browserscale's job is to make all of that feel like one
page. Every frame on a page — same-origin, cross-origin, in-process,
out-of-process, nested seven levels deep — is addressed by a single
opaque `frameId` string. Every locator, every action, every `Wait`
takes that id the same way. There's no second API for cross-origin
frames.

> **TL;DR**
>
> - A frame is just a `frameId` string. Pass it to `.InFrame(...)` on a locator, or to `EvaluateInFrame`, or any action's `inFrame` option, and the call runs inside that frame.
> - Every result that touches a frame returns the `frameId` it acted in — `Wait`, `Click`, `Fill`, `Evaluate`, all of them. Most discovery happens by-the-way, no separate "list frames" call needed.
> - `Wait(...).InAllFrames()` is the everyday way to find an element when you don't know which frame it's in. The result tells you both the `frameId` and the `backendNodeId`.
> - `GetPages` is the explicit, recursive map of the whole frame tree — useful for inspection, rarely needed in the hot path.
> - Each frame carries an `AbsoluteRect` in root-viewport coordinates, even when it's an OOPIF nested several levels deep. Coordinate clicks land where you'd expect.

## The flat frame model

`FrameInfo` is small on purpose — there is no frame *object* with
methods to call, only data you read and a `frameId` you pass to the
next call:

| Field | What it tells you |
| --- | --- |
| `frameId` | The id you hand to `.InFrame(...)` / `EvaluateInFrame` / `*Opts.inFrame`. |
| `url` | Whatever URL the frame currently has loaded. |
| `isOOPIF` | `true` if the frame runs in a separate process (cross-origin). Informational only — behaviour is identical across same-origin and OOPIF. |
| `hasJSContext` | `true` once the frame has an execution context attached. `EvaluateInFrame` only works on frames with this flag set. |
| `isLoading` | `true` while the frame is navigating. Pair with `Wait` rather than polling this. |
| `isVisible` | `false` for `display:none` or zero-sized frames — quick skip for "frames the user can't see". |
| `absoluteRect` | Position and size in **root-viewport coordinates**, even for OOPIFs nested deep. This is what makes coordinate clicks work uniformly across frames. |
| `relativeRect` | Position relative to the parent frame (always `0,0` for the main frame). |
| `children` | Nested frames, recursively. |

The point of that table is what's *not* on it: no proxy handle, no
"contentWindow" pointer, no per-frame CDP target you have to attach
to. The `frameId` is the whole thing.

## Discovery via `Wait` — the everyday case

In practice, you almost never enumerate the frame tree just to find
something. Most of the time you know "this consent button shows up
somewhere on the page" and `Wait(...).InAllFrames()` finds it for
you, telling you which frame it landed in as a side effect:

**Go:**

```go
r, err := browser.Wait(ctx, browserscale.CSS("button.accept-all").InAllFrames())
if err != nil {
    log.Fatal(err)
}
// r.FrameId is exactly the id you'd pass to any frame-scoped call.
// r.BackendNodeId is the matched node, valid only inside r.FrameId.
_, _ = browser.Click(ctx, browserscale.Node(r.BackendNodeId).InFrame(r.FrameId))
```

**TypeScript:**

```ts
const r = await browser.wait(css("button.accept-all").inAllFrames());
// r.frameId is exactly the id you'd pass to any frame-scoped call.
// r.backendNodeId is the matched node, valid only inside r.frameId.
await browser.click(node(r.backendNodeId).inFrame(r.frameId));
```

That is the canonical pattern, and it's how the rest of this guide
expects you to discover frame ids. `Click`, `Fill`, `Drag`, `MoveTo`,
`ScrollTo`, `Evaluate` — all of them return the `frameId` of the
frame they ran in. As you script a flow, you accumulate the ids of
the frames you've touched without ever asking for them explicitly.

## Discovery via `GetPages` — the explicit map

When you need the whole picture — every frame currently on the page,
the URL each one has loaded, where they sit, whether they're ready —
`GetPages` returns it as a recursive tree:

**Go:**

```go
pages, _ := browser.GetPages(ctx)
for _, p := range pages {
    fmt.Println("page:", p.Url, "main:", p.FrameTree.FrameId)
    walk(&p.FrameTree, 0)
}

func walk(f *browserscale.FrameInfo, depth int) {
    indent := strings.Repeat("  ", depth)
    fmt.Printf("%s- %s (oopif=%v, jsctx=%v) %s\\n",
        indent, f.FrameId, f.IsOOPIF, f.HasJSContext, f.Url)
    for _, c := range f.Children {
        walk(c, depth+1)
    }
}
```

**TypeScript:**

```ts
const pages = await browser.getPages();
for (const p of pages) {
    console.log("page:", p.url, "main:", p.frameTree.frameId);
    walk(p.frameTree, 0);
}

function walk(f: FrameInfo, depth: number) {
    const indent = "  ".repeat(depth);
    console.log(\`\${indent}- \${f.frameId} (oopif=\${f.isOOPIF}, jsctx=\${f.hasJSContext}) \${f.url}\`);
    for (const c of f.children) walk(c, depth + 1);
}
```

A few things drop out of the tree that are surprisingly useful in
scripts:

- **`isOOPIF`** lets you spot third-party iframes (consent banners,
  ads, payment widgets) without parsing URLs. But don't *branch* on
  it for behaviour — the rest of the API works the same regardless.
- **`hasJSContext`** is the only practical "is this frame ready
  enough to receive `EvaluateInFrame`?" flag. A freshly navigated
  frame can be in the tree before its execution context is attached.
- **`isVisible`** is a quick filter for "frames the user can actually
  see" — skip the rest when you're rendering an agent observation.

## Acting in a frame

Every locator has the same two frame modifiers, and they're how you
say "this call targets *that* frame":

**Go:**

```go
// Target a specific frame.
_, _ = browser.Click(ctx, browserscale.CSS("button").InFrame(iframeId))

// Fan out across every frame.
_, _ = browser.Wait(ctx, browserscale.CSS("input.search").InAllFrames())
```

**TypeScript:**

```ts
// Target a specific frame.
await browser.click(css("button").inFrame(iframeId));

// Fan out across every frame.
await browser.wait(css("input.search").inAllFrames());
```

A handful of actions also accept an `inFrame` option that wins over
whatever the locator already had — `Click`, `Fill`, and the `Select*`
variants. Handy when you want to reuse a base locator across several
frames:

**Go:**

```go
base := browserscale.CSS("button.submit")  // no frame baked in
_, _ = browser.ClickWith(ctx, base, browserscale.ClickOpts{InFrame: frameA})
_, _ = browser.ClickWith(ctx, base, browserscale.ClickOpts{InFrame: frameB})
```

**TypeScript:**

```ts
const base = css("button.submit");           // no frame baked in
await browser.click(base, { inFrame: frameA });
await browser.click(base, { inFrame: frameB });
```

For everything else (`MoveTo`, `ScrollTo`, `Drag*`), the frame comes
from the locator — bake it in with `.InFrame(...)` or `.InAllFrames()`.

For arbitrary JS inside a specific frame, `EvaluateInFrame` is the
direct route (see [Evaluation](/docs/guides/evaluation) for the
return-value semantics).

The frame surface is intentionally uniform: same-origin or
cross-origin, top-level or three levels deep — same calls, same
return shape, no extra setup.

## Coordinate clicks across frames

`AbsoluteRect` is in **root-viewport coordinates**, no matter how
many OOPIF boundaries the frame is buried behind. That means
coordinate-based actions (`Click(At(x, y))`, `MoveTo(At(x, y))`)
work uniformly without you having to translate positions through any
frame chain by hand:

**Go:**

```go
pages, _ := browser.GetPages(ctx)
target := findFrame(&pages[0].FrameTree, "https://challenges.cloudflare.com")
if target == nil {
    log.Fatal("frame not found")
}

// Click the centre of the frame in root-viewport coordinates.
cx := target.AbsoluteRect.X + target.AbsoluteRect.Width/2
cy := target.AbsoluteRect.Y + target.AbsoluteRect.Height/2
_, _ = browser.Click(ctx, browserscale.At(cx, cy))
```

**TypeScript:**

```ts
const pages = await browser.getPages();
const target = findFrame(pages[0].frameTree, "https://challenges.cloudflare.com");
if (!target) throw new Error("frame not found");

// Click the centre of the frame in root-viewport coordinates.
const cx = target.absoluteRect.x + target.absoluteRect.width / 2;
const cy = target.absoluteRect.y + target.absoluteRect.height / 2;
await browser.click(at(cx, cy));
```

That works for any frame, any depth, any origin — the chromium
plumbing that would otherwise force you into per-frame coordinate
transforms is just gone.

## Multiple tabs and popups — today vs. tomorrow

`GetPages` is plural for a reason: the underlying browser context
can hold more than one page (additional tabs, popups, target=_blank
windows). Today, a session implicitly targets **one page** — the one
it was rented with. Other pages show up in `GetPages` so you can see
they exist, but switching the session to drive a different tab is
not yet wired up.

That doesn't change much in practice: most automation work happens on
a single page across its full frame tree, which is exactly what browserscale
supports cleanly. Multi-page (switch active tab / drive a popup) is
on the roadmap — when it lands, the surface will extend rather than
replace what's documented here.

## Gotchas

- **Frame ids don't outlive their document.** A navigation in a
  frame mints a new `frameId`. Re-discover via `Wait(...).InAllFrames()`
  or a fresh `GetPages` after anything that could reload the frame.
- **`backendNodeId`s are per-document.** A `Node(id)` you got from a
  `Wait` result only makes sense paired with the `FrameId` from the
  same result. Don't carry a backendNodeId across navigations.
- **`hasJSContext: false` means `EvaluateInFrame` will fail.** If
  you're getting odd errors right after a frame appears in the tree,
  wait for the actual element you care about with `Wait` instead of
  polling `GetPages`.
- **`AllFrames` ≠ a `frameId`.** It is the sentinel
  `"ALL_FRAMES"` string. `.InAllFrames()` and
  `inFrame: AllFrames` use it; `EvaluateInFrame` does not accept it
  (use `Wait(...).InAllFrames()` first to discover which frame to
  evaluate in — covered in [Evaluation](/docs/guides/evaluation)).
- **Don't branch behaviour on `isOOPIF`.** It's informational only.
  The API works identically whether the frame is cross-origin or not;
  branching on the flag tangles your code without changing what you
  can do.

## See also

- [Targeting elements](/docs/guides/locators) — where `.InFrame(...)` and `.InAllFrames()` are introduced.
- [Waiting](/docs/guides/waiting) — the primary way `frameId`s flow into your script.
- [Evaluation](/docs/guides/evaluation) — `EvaluateInFrame` and the cross-frame discovery handshake.
- API reference: [Go `GetPages` / `FrameInfo`](/docs/api-reference/go#GetPages) · [TS `getPages` / `FrameInfo`](/docs/api-reference/ts#getPages).

→ Continue: [Network](/docs/guides/network)
