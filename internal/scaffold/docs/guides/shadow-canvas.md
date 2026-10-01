<!--
  url: https://browserscale.cloud/docs/guides/shadow-canvas
  title: Shadow DOM & canvas
  description: Read across browser security boundaries in a browserscale session: pierce closed shadow roots and read pixels from tainted cross-origin canvases.
-->

# Shadow DOM & canvas

Some things on a page are deliberately walled off from automation:
elements sealed inside a **closed shadow root**, and pixels drawn onto
a **cross-origin (tainted) canvas**. Both boundaries are enforced by
the browser itself, so a tool that sits *outside* the engine — driving
over the DevTools protocol, or injecting page JavaScript — hits a wall.
browserscale runs natively inside the renderer, so it can read across
both without tripping any security exception or leaving a trace.

> **TL;DR**
>
> - **Closed shadow roots** — `element.shadowRoot` returns `null` and page JS can't reach inside. The `__wrc.shadow(el)` helper, available inside any `Evaluate` / `Wait` expression, returns the root regardless of open/closed mode.
> - **Tainted canvas** — `toDataURL()` / `getImageData()` throw a `SecurityError` on a canvas that has drawn cross-origin content. `ReadCanvas` reads the raw pixels anyway, straight from the renderer.
> - Both are engine-level reads: nothing is injected, no global is installed, and the page cannot observe that you looked.
> - `__wrc` is a function-scoped local, not a `window` property — a site can't poll for it, enumerate it, or call it.

## Piercing closed shadow roots — `__wrc.shadow`

Web components can attach their shadow root in `closed` mode. When they
do, the standard `el.shadowRoot` getter returns `null`, and there is no
supported way for page JavaScript to reach the elements inside. Regular
CSS selectors don't cross the boundary either — neither in the SDK's
locators nor in `document.querySelector`.

browserscale exposes a native helper, **`__wrc.shadow(el)`**, to the
expressions you pass to `Evaluate` and `Wait`. It returns `el`'s shadow
root whether it is open or closed, reading the DOM natively in the
renderer (no page JS runs, no bindings are touched):

**Go:**

```go
// A third-party widget renders its markup inside a closed shadow root.
// el.shadowRoot would be null; __wrc.shadow(el) returns it anyway.
r, err := browser.Evaluate(ctx, \`
    (() => {
        const host = document.querySelector("my-widget");
        const root = __wrc.shadow(host);              // open OR closed
        const label = root.querySelector(".status");
        return label ? label.textContent.trim() : null;
    })()
\`)
if err != nil {
    log.Fatal(err)
}
fmt.Println(r.Value)
```

**TypeScript:**

```ts
// A third-party widget renders its markup inside a closed shadow root.
// el.shadowRoot would be null; __wrc.shadow(el) returns it anyway.
const r = await browser.evaluate<string | null>(\`
    (() => {
        const host = document.querySelector("my-widget");
        const root = __wrc.shadow(host);              // open OR closed
        const label = root.querySelector(".status");
        return label ? label.textContent.trim() : null;
    })()
\`);
console.log(r.value);
```

### Interacting with an element inside a closed root

CSS selectors can't reach inside a shadow root, but every action verb
has a second targeting mode: a **JS expression that evaluates to the
element**. That expression runs on the same `__wrc`-aware eval path as
`Evaluate`, so `__wrc.shadow(...)` works there too. The server resolves
the returned element, scrolls it into view, and dispatches a real,
trusted event at its actual coordinates — no manual rect maths, no
`At(x, y)`.

**Go:**

```go
// Target the inner button by JS expression — Click resolves it and
// clicks it for real, closed shadow root and all.
_, err := browser.Click(ctx, browserscale.JS(\`
    __wrc.shadow(document.querySelector("my-widget")).querySelector("button.accept")
\`))
if err != nil {
    log.Fatal(err)
}
```

**TypeScript:**

```ts
// Target the inner button by JS expression — click resolves it and
// clicks it for real, closed shadow root and all.
await browser.click(js(\`
    __wrc.shadow(document.querySelector("my-widget")).querySelector("button.accept")
\`));
```

The same JS-expression targeting works for `Fill`, `ScrollTo`,
`SelectOption`, `Drag` and `ReadCanvas` — anywhere you would otherwise
pass a CSS selector.

The manual **rect + `Click(At(x, y))`** detour is only needed when the
target lives inside a **cross-origin iframe** (a separate renderer
process), where it can't be resolved by `backendNodeId` from the parent
frame. There you read the iframe's position, add the element's offset,
and click the computed viewport coordinate.

### Waiting on something inside a shadow root

`__wrc.shadow` works inside `Wait`'s JavaScript conditions too. The
expression is re-checked as the page changes, so you can wait for an element that
only exists behind the boundary:

**Go:**

```go
// Wait until the widget inside the closed root reports "ready".
_, err := browser.Wait(ctx, browserscale.JS(\`
    !!__wrc.shadow(document.querySelector("my-widget"))
        ?.querySelector(".status.ready")
\`))
if err != nil {
    log.Fatal(err)
}
```

**TypeScript:**

```ts
// Wait until the widget inside the closed root reports "ready".
await browser.wait(js(\`
    !!__wrc.shadow(document.querySelector("my-widget"))
        ?.querySelector(".status.ready")
\`));
```

The helper never becomes a real global. When your expression mentions
`__wrc`, the browser evaluates it as the body of a wrapper function that
receives `__wrc` as a local argument, then discards it — so
`window.__wrc` is always `undefined` for the page. A site cannot detect
that the helper exists, and expressions that don't reference it run
verbatim with zero overhead.

## Reading a tainted canvas — `ReadCanvas`

A `<canvas>` becomes **tainted** the moment it draws content from
another origin — an image, a video frame, or a WebGL texture loaded
cross-origin without permissive CORS headers. After that, the browser
blocks pixel readback: `toDataURL()` and `getImageData()` throw a
`SecurityError`. That's a deliberate anti-exfiltration measure, and it
also blocks any legitimate automation that needs the pixels.

`ReadCanvas` reads the canvas straight from the renderer's backing
store, so it returns the pixels whether or not the canvas is
origin-clean. It addresses the element the same way every other action
does — by CSS selector, JS expression, or `backendNodeId`:

**Go:**

```go
res, err := browser.ReadCanvas(ctx, browserscale.CSS("#chart canvas"))
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%dx%d, originClean=%v\\n", res.Width, res.Height, res.OriginClean)

img, _ := base64.StdEncoding.DecodeString(res.DataBase64)
_ = os.WriteFile("chart.png", img, 0o644)
```

**TypeScript:**

```ts
const res = await browser.readCanvas(css("#chart canvas"));
console.log(res.width, res.height, "originClean=", res.originClean);

await fs.writeFile("chart.png", Buffer.from(res.dataBase64, "base64"));
```

`OriginClean` (Go) / `originClean` (TS) tells you whether the canvas was
tainted — it's informational only. The read succeeds either way; the
flag just lets you log or branch if you care.

### Format, quality and sub-rectangles

The customizable variant takes an options bag: pick an encoding, set a
quality for lossy formats, or read just a slice of the canvas (the same
`sx, sy, sw, sh` window `getImageData` uses). Omitting the rectangle
reads the whole canvas.

**Go:**

```go
// Read only a 150x300 slice, encoded as JPEG at quality 80.
res, err := browser.ReadCanvasWith(ctx, browserscale.CSS("canvas"),
    browserscale.ReadCanvasOpts{
        Format:  "jpeg",
        Quality: 80,
        SX:      0, SY: 0, SW: 150, SH: 300,
    })
if err != nil {
    log.Fatal(err)
}
_ = res
```

**TypeScript:**

```ts
// Read only a 150x300 slice, encoded as JPEG at quality 80.
const res = await browser.readCanvas(css("canvas"), {
    format: "jpeg",
    quality: 80,
    sx: 0, sy: 0, sw: 150, sh: 300,
});
```

`Format` accepts `"png"` (default), `"jpeg"`, `"webp"`, or `"rgba"` for
the raw unpremultiplied pixel buffer when you want to run your own
image analysis without a decode step.

### `ReadCanvas` vs `Screenshot`

They look similar but answer different questions:

| You want… | Use |
| --- | --- |
| The exact pixels of one `<canvas>`, tainted or not, at its own resolution | `ReadCanvas` |
| A composited picture of the visible viewport (all elements, layout, overlays) | `Screenshot` |

`Screenshot` captures what's painted in the viewport — it can't give you
a tainted canvas's underlying pixels, and it's framed by the viewport,
not the element. `ReadCanvas` targets a single canvas element and reads
its backing store directly, even if the canvas is partly scrolled
off-screen.

## Gotchas

- **CSS selectors don't pierce shadow roots — JS-expression targets
  do.** A CSS-selector target resolves against the light DOM only. To
  act inside a shadow root, target with a JS expression that returns the
  element (`__wrc.shadow(host).querySelector(...)`); `Click`, `Fill`,
  `ReadCanvas` and friends then act on it directly. The `Click(At(x, y))`
  detour is only for targets inside a cross-origin iframe.
- **`__wrc` is not a global.** Don't try to read `window.__wrc` — it's a
  function-scoped local that only exists while your expression runs.
- **`__wrc.shadow(el)` needs a host with a shadow root.** Passing an
  element that has none returns `null`; guard with optional chaining
  (`?.`) as in the wait example.
- **`ReadCanvas` needs a real `<canvas>`.** `At(x, y)` coordinates are
  not a valid target — pass a selector, JS expression, or `Node(id)`.
- **The rectangle is in canvas pixels, not CSS pixels.** A canvas with a
  `width`/`height` attribute larger than its CSS box reads at its
  intrinsic resolution; size `sw`/`sh` against that.

## See also

- [Evaluation](/docs/guides/evaluation) — the `Evaluate` call that hosts `__wrc.shadow`, and the locate-then-act pattern.
- [Reading the page](/docs/guides/reading) — DOM, observation and screenshot reads that complement canvas readback.
- [Interaction](/docs/guides/interaction#click) — JS-expression targeting for elements inside shadow roots, and `Click(At(x, y))` for raw coordinates.
- API reference: [Go `ReadCanvas`](/docs/api-reference/go#ReadCanvas) · [TS `readCanvas`](/docs/api-reference/ts#readCanvas).

→ Continue: [Agentic coding](/docs/guides/agentic-coding)
