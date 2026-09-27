<!--
  url: https://browserscale.cloud/docs/guides/evaluation
  title: Evaluating JavaScript
  description: Run JavaScript inside the page or a specific frame from your Go or TypeScript script and return typed values back.
-->

# Evaluation

Sometimes the SDK's typed verbs aren't quite the right shape — you
want a value out of the page, a custom predicate, or a one-off DOM
query that doesn't deserve its own helper. `Evaluate` runs an
arbitrary JavaScript expression in the page and gives you back either
a regular value or a handle to a DOM element you can act on. Running
the same expression inside any frame on the page — even a cross-origin
one — is just `EvaluateInFrame` with the frame's id.

> **TL;DR**
>
> - `Evaluate(expr)` runs JavaScript in the main frame, `EvaluateInFrame(frameId, expr)` runs it in any frame by id.
> - The return splits two ways: non-Element values land in `Value` (JSON-decoded); a DOM Element populates `BackendNodeId` / `IsVisible` / `Bounds` instead, ready to hand to `Node(id)` for typed follow-up actions.
> - Cross-origin / OOPIF frames are first-class — once you have a `frameId` (from `Wait`, `GetPages`, or any action's result), evaluating in them is a single call. No proxy chains.
> - Don't use `Evaluate` as a wait loop. `Wait` is the right tool for "wait until something is true".
> - Don't use it to read page state either. URL, title, scroll offset, live input values, checkbox state and duplicate-id warnings all come back from one [`GetObservation`](/docs/guides/reading).

## The basic call

A single expression goes in, the JSON-serialised result comes out:

**Go:**

```go
r, err := browser.Evaluate(ctx, "document.title")
if err != nil {
    log.Fatal(err)
}
fmt.Println(r.Value)        // → "Example Domain"
```

**TypeScript:**

```ts
const r = await browser.evaluate<string>("document.title");
console.log(r.value);        // → "Example Domain"
```

A couple of small but important rules:

- The string you pass must be a **single expression**, not a script
  body. For anything multi-statement, wrap it in an IIFE — see the
  recipes below.
- The result is whatever your expression evaluates to, JSON-serialised
  by the server and parsed back client-side. Anything not
  JSON-representable (functions, `undefined`, circular structures…)
  comes back as the zero value.
- In TypeScript, `evaluate` is generic: `evaluate<MyType>(...)` lets
  you type the return value at the call site instead of casting later.

## The two return-value shapes

There is only one `EvaluateResult` type, but it has two distinct
populated shapes depending on what your expression returned:

- **Value path** — the expression returned a JSON-serialisable value
  (string, number, array, plain object). `Value` carries it; the
  Element fields stay zeroed.
- **Element path** — the expression returned a single DOM Element.
  `Value` is empty/null, but `BackendNodeId` is a non-zero handle to
  that element, with `IsVisible` and `Bounds` populated as a bonus.

The Element path is the part most people miss the first time — and
it's exactly what makes `Evaluate` a serious tool rather than a
debug helper. It lets you write one expression that *finds* the
element and hand the result to a typed SDK action with no second
lookup:

**Go:**

```go
// Find the first visible "Add to cart" button via JS, then click it.
r, err := browser.Evaluate(ctx, \`
    [...document.querySelectorAll("button")]
      .find(b => b.offsetParent && /add to cart/i.test(b.textContent))
\`)
if err != nil || r.BackendNodeId == 0 {
    log.Fatal("no add-to-cart button")
}
_, _ = browser.Click(ctx, browserscale.Node(r.BackendNodeId))
```

**TypeScript:**

```ts
// Find the first visible "Add to cart" button via JS, then click it.
const r = await browser.evaluate(\`
    [...document.querySelectorAll("button")]
      .find(b => b.offsetParent && /add to cart/i.test(b.textContent))
\`);
if (r.backendNodeId === 0) throw new Error("no add-to-cart button");
await browser.click(node(r.backendNodeId));
```

The pattern — *one expression to locate, one typed call to act* — is
the same one [`Wait`](/docs/guides/waiting) gives you with
`r.BackendNodeId`. `Evaluate` just lets you swap CSS for arbitrary
JavaScript when CSS isn't expressive enough.

A quick mental rule: if your expression's last value is an
`Element`, you'll be looking at the Element path. Anything else (text,
number, plain object, array, `null`) takes the Value path.

## EvaluateInFrame — same call, any frame

`EvaluateInFrame` is `Evaluate` plus a `frameId` argument. Same return
shape, same rules — the JS just runs in the frame you point at
instead of the main one:

**Go:**

```go
title, _ := browser.EvaluateInFrame(ctx, iframeId, "document.title")
fmt.Println(title.Value)
```

**TypeScript:**

```ts
const title = await browser.evaluateInFrame<string>(iframeId, "document.title");
console.log(title.value);
```

browserscale treats every frame on the page as a plain string id. The id you
get from a `Wait` result, the id you get from `GetPages.FrameTree`,
the id baked into a locator's `.InFrame(...)` — they all interchange
freely, and they all work with `EvaluateInFrame`. Same-origin or
cross-origin, in-process or out-of-process: the call shape doesn't
change. *That* is the part that's nice about browserscale's frame model — the
plumbing CDP-based tools usually expose is gone.

## The canonical pattern: wait across all frames, evaluate in the match

`Wait` can fan out across every frame with `.InAllFrames()`, and the
result tells you which frame actually matched. `EvaluateInFrame` then
runs your JS in that exact frame:

**Go:**

```go
// 1. Wait for the consent button to appear, anywhere on the page.
r, err := browser.Wait(ctx, browserscale.CSS("button.accept-all").InAllFrames())
if err != nil {
    log.Fatal(err)
}

// 2. Inspect or transform something in *that* frame.
info, _ := browser.EvaluateInFrame(ctx, r.FrameId, \`
    (() => {
        const btn = document.querySelector("button.accept-all");
        return { text: btn.textContent.trim(), href: location.href };
    })()
\`)
fmt.Println(info.Value)

// 3. Or just click it — Wait already gave you the backendNodeId.
_, _ = browser.Click(ctx, browserscale.Node(r.BackendNodeId).InFrame(r.FrameId))
```

**TypeScript:**

```ts
// 1. Wait for the consent button to appear, anywhere on the page.
const r = await browser.wait(css("button.accept-all").inAllFrames());

// 2. Inspect or transform something in *that* frame.
const info = await browser.evaluateInFrame<{ text: string; href: string }>(
    r.frameId,
    \`(() => {
        const btn = document.querySelector("button.accept-all");
        return { text: btn.textContent.trim(), href: location.href };
    })()\`,
);
console.log(info.value);

// 3. Or just click it — wait already gave you the backendNodeId.
await browser.click(node(r.backendNodeId).inFrame(r.frameId));
```

Three things worth pulling out of that snippet:

- `Wait(...).InAllFrames()` is how you say *"I don't know where this
  element lives, find it for me"*. The match wins and you get the
  exact `FrameId` back.
- `EvaluateInFrame` itself does **not** take `ALL_FRAMES` — it always
  runs in exactly one frame. The `Wait → EvaluateInFrame` handshake
  *is* the cross-frame search pattern.
- Note the IIFE wrapping the multi-statement body. Plain
  `const btn = …; return …;` would fail; an arrow IIFE makes it a
  single expression.

## Anti-patterns

Two ways people abuse `Evaluate` that you should not:

### Don't use it as a wait loop

If you find yourself writing `setTimeout` / polling loops inside an
`Evaluate` expression, or repeatedly calling `Evaluate` from your
script to check for some condition, stop. That's `Wait`'s job. See
[Waiting](/docs/guides/waiting) for the explicit pattern.

**Go:**

```go
// ❌ Don't do this — burns round-trips and never times out cleanly.
for {
    r, _ := browser.Evaluate(ctx, "!!document.querySelector('.done')")
    if v, ok := r.Value.(bool); ok && v { break }
    time.Sleep(100 * time.Millisecond)
}

// ✅ Do this instead.
_, _ = browser.Wait(ctx, browserscale.CSS(".done"))
```

**TypeScript:**

```ts
// ❌ Don't do this — burns round-trips and never times out cleanly.
while (true) {
    const r = await browser.evaluate<boolean>("!!document.querySelector('.done')");
    if (r.value === true) break;
    await sleep(100);
}

// ✅ Do this instead.
await browser.wait(css(".done"));
```

### Don't use it for actions the SDK already has

`Evaluate("document.querySelector('button').click()")` *will* fire a
click event in the DOM, but it dispatches synthetic JS events with no
real mouse path, no scroll-into-view, no focus management — and modern
anti-bot stacks have been trained to spot exactly that. The same goes
for assigning `.value` instead of typing: the events the page listens
for never fire. Use [`Click`](/docs/guides/interaction#click) and
[`Fill`](/docs/guides/interaction#fill) when you mean click and type,
`InsertText` for a silent bulk paste, and `Evaluate` for things the SDK
doesn't have a verb for.

### Don't use it to re-read page state

Reaching for `document.title`, `location.href`, `document.body.innerText`
or a hand-rolled `querySelectorAll('input')` map after every step is the
most common way to burn tokens. One
[`GetObservation`](/docs/guides/reading) already returns the URL, title
and scroll offset in its frame headers, plus every visible control with
its live `value`, `checked` state and duplicate-id flags — for a fraction
of what re-deriving it costs.

Where `Evaluate` genuinely earns its keep alongside an observation is
the opposite direction: **verifying a target before you commit to it**.
Run the exact expression you plan to pass as a `JS(...)` locator and
confirm it resolves to one element, and the selector you write into your
script is one you've already proven.

**Go:**

```go
// Prove the anchor is unique before baking it into the script.
n, _ := browser.Evaluate(ctx, "document.querySelectorAll('input[name=loginId]').length")
fmt.Println(n.Value)   // → 1
```

**TypeScript:**

```ts
// Prove the anchor is unique before baking it into the script.
const n = await browser.evaluate<number>("document.querySelectorAll('input[name=loginId]').length");
console.log(n.value);  // → 1
```

## Gotchas

- **Single expression only.** Multi-statement bodies need an IIFE:
  `(() => { /* statements */; return value; })()`.
- **`undefined`, functions and circular structures don't serialise.**
  They come back as the zero value (`nil` in Go, `null` in TS).
- **Element returns reset the Value field.** When `BackendNodeId > 0`,
  `Value` is `nil` / `null` regardless of what other properties the
  element had. Read from `BackendNodeId` / `IsVisible` / `Bounds`
  instead.
- **Errors thrown in the expression surface as call errors.**
  `throw new Error("nope")` becomes a returned error on the SDK call —
  there's no separate "expression threw" channel.
- **`EvaluateInFrame` doesn't take `ALL_FRAMES`.** That's the whole
  reason `Wait(...).InAllFrames()` exists. Find the frame with
  `Wait`, run the JS with `EvaluateInFrame` — one frame at a time.

## See also

- [Reading the page](/docs/guides/reading) — when you want to *look* at the page without running JS.
- [Frames & iframes](/docs/guides/frames) — where `frameId`s come from and how the frame tree is shaped.
- [Targeting elements](/docs/guides/locators) — the `Node(id)` constructor that turns an `Evaluate` result into the next call's target.
- API reference: [Go `Evaluate` / `EvaluateInFrame`](/docs/api-reference/go#Evaluate) · [TS `evaluate` / `evaluateInFrame`](/docs/api-reference/ts#evaluate).

→ Continue: [Frames & iframes](/docs/guides/frames)
