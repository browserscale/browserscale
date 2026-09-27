<!--
  url: https://browserscale.cloud/docs/guides/reading
  title: Reading the page
  description: Read text content, attributes, the DOM, screenshots and accessibility observations from a live browserscale session.
-->

# Reading the page

Sometimes a script needs to *look* at the page rather than change it —
to feed an LLM the current state, diff a layout between two snapshots,
figure out what's under a coordinate, or read back what the user
highlighted. browserscale ships a small set of read-only methods for exactly
that. They all return data and never poke the page.

> **TL;DR**
>
> - `GetObservation` is the easy on-ramp: a compact, agent-friendly view of what's visible right now — headers included, so it also answers "where am I?".
> - `GetDOM` is the full CDP DOM tree as JSON — use when you need every node, not when you need a quick overview.
> - `GetDOMHash` is a 16-char fingerprint of the DOM tree — pair it with `GetDOM` to skip unchanged snapshots.
> - `InspectAtPosition`, `HighlightNode` and `GetSelection` cover the live-UI cases: hit-test, debug overlay, copy what's selected.
> - Reading methods don't wait. Pair them with `Wait` whenever you depend on something specific being there.

## GetObservation — the agent-friendly summary

The first thing to reach for on an unfamiliar page, and the cheapest
way to re-read the current state afterwards. The server walks every
frame, keeps what is actually visible, and returns it as text meant to
be handed to a model unchanged.

**Go:**

```go
obs, err := browser.GetObservation(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println(obs)
```

**TypeScript:**

```ts
const obs = await browser.getObservation();
console.log(obs);
```

Each frame opens with header lines, then emits one line per element,
indented by tree depth:

```
# frame 8F03BF2EEB5ADC9CE47B15775289BBD3 https://example.com/register
# title "Account registration"
# scroll y=1200/8400
input#email[47] type="email" name="loginId" value="a@b.com" required click "E-Mail"
input#agree[51] type="checkbox" checked="false" required click "Accept terms"
select#pref[62] name="prefecture" value="13" options="01:Hokkaido,02:Aomori,…" click
button[70] type="submit" click "Continue"
```

Because the headers already carry the URL, the title and the scroll
offset, you rarely need an `Evaluate` round-trip just to work out where
a flow ended up. The title line is omitted when the document has none,
the scroll line when the frame doesn't scroll.

### Reading a line

The leading token is `tag#id[backendNodeId]`, followed by attribute
keys and then bare boolean flags. Attribute keys are `role`, `type`,
`name`, `placeholder`, `value`, `checked`, `options`, `href`, `src`,
`frameId` (on iframes, naming the frame section further down) and
`hidden`.

Two details matter more than the rest. `value` is read from the live
IDL property, so it reflects what was actually typed rather than the
initial markup — which is how you verify a `Fill` landed; on password
fields it is reported as a length, e.g. `value="(8 chars)"`. And the
trailing quoted string is always the element's label or text, **never**
its value or placeholder, so an empty field and a prefilled one stay
distinguishable.

The bare flags are `disabled`, `required`, `readonly`, `selected`,
`offscreen`, `click`, `id-not-unique` and `id-not-selectable`.
`click` means the element carries an interactivity signal: it is a
control or link, or it has a role, a tabindex, a click handler or an
introduced `cursor:pointer` — which is how `<div onclick>` buttons get
caught. `offscreen` appears only on interactive elements and means you
must scroll before acting.

### What gets included

Traversal follows the flat tree, so open **and closed** shadow roots
are included; user-agent shadow roots are not. `<select>` options are
enumerated explicitly (capped at 60 per select) because you need them
to call `SelectByValue`. Generic wrappers with no interactivity signal,
no id or role and no own text are omitted, and their children reported
at the parent's depth. Element text is emitted exactly once: a
container only shows text that no descendant row already carries.

Subtrees without a layout box (`display: none`) are pruned entirely.
Elements that are merely unseeable — `visibility: hidden`, `opacity: 0`,
zero-sized — are reported only when they are still interactive, and
carry a `hidden="<reason>"` attribute. That is deliberate: when a click
fails, the reason is in the observation instead of the element having
silently vanished from it.

### Budgets

The limit that normally binds is **`maxTotalTokens`** (default 8000),
a budget across *all* frames measured in estimated tokens rather than
characters — the same character count is worth roughly four times as
many tokens in CJK text as in ASCII, so a character limit would mean
something different on every page. Frames are visited in tree order and
each gets whatever is left, so a page full of iframes can't multiply the
limit.

**`maxElementsPerFrame`** (default 800) is a safety net against runaway
documents, and **`maxTextLength`** (default 300) caps human-readable
strings; identifier-like attributes such as `type` and `name` have their
own shorter cap and are unaffected.

Running out of budget does not cut the walk off in document order — that
would reliably drop the end of the page and with it the submit button.
Instead the remainder of the frame degrades to interactive elements only,
marked by a `# budget spent` line. When you see one and still need the
rest, scroll and observe again, or narrow the walk:

**Go:**

```go
// Just what's on screen right now.
obs, _ := browser.GetObservationWith(ctx, browserscale.ObservationOpts{
    ViewportOnly: true,
})
```

**TypeScript:**

```ts
// Just what's on screen right now.
const obs = await browser.getObservation({ viewportOnly: true });
```

### Scoping to a subtree

Omit the scope fields for the whole page (the default). After the first
full look, pass exactly one of `Selector`, `JSExpression` or
`BackendNodeId` to observe only that element's subtree — follow-up looks
at a form then cost the form, not the ads around it. Addressing matches
`Click` / `Fill`. Child iframes reached inside the scope are still
visited (so a checkout form keeps its payment iframe); frames outside
the scope are not. Text mode marks a scoped result with a
`# scope tag[backendNodeId]` header line.

**Go:**

```go
// Re-read just the registration form.
obs, _ := browser.GetObservationWith(ctx, browserscale.ObservationOpts{
    Selector: "form#register",
})
// Or by a handle from the previous observation:
obs, _ = browser.GetObservationWith(ctx, browserscale.ObservationOpts{
    BackendNodeId: 120,
    InFrame:       frameId,
})
```

**TypeScript:**

```ts
// Re-read just the registration form.
const obs = await browser.getObservation({ selector: "form#register" });
// Or by a handle from the previous observation:
const again = await browser.getObservation({
  backendNodeId: 120,
  frameId,
});
```

### Choosing what to target

`backendNodeId` — the `47` in `input#email[47]` — is a handle for the
current session. Pass it straight to an action via `Node(...)` and you
never guess a selector:

**Go:**

```go
_, _ = browser.Click(ctx, browserscale.Node(47).InFrame(frameId))
```

**TypeScript:**

```ts
await browser.click(node(47).inFrame(frameId));
```

It does not survive a new document, though, so it is worthless in a
script you intend to run again. Prefer the anchor you can still use
tomorrow: `name`, then an id flagged with neither `id-not-unique` nor
`id-not-selectable`, then a label or text relation — and never a
generated class name. The two id flags are worth internalising:
`id-not-unique` means the id is duplicated in this tree scope and `#id`
happens to resolve to *this* element, while `id-not-selectable` means it
resolves to a different one, so the element can't be reached by its id
at all.

The moment the observation is in front of you is the only one where
`name`, the id flags and the label are all visible at once, so pick the
durable target then rather than later. Acting through it right away has
a second payoff: every call that worked during exploration is already a
line of your script, and since actions return the `backendNodeId` they
resolved to, comparing that against the observation proves the anchor
hits the element you meant.

### The JSON form

Passing `format: "json"` returns the same rows with explicit keys, plus
per-frame counts, a `degraded` flag and a `truncated` reason
(`max_elements`, `node_budget` or `token_budget`). It is roughly twice
the size of the text form for identical information, so it is meant for
programmatic consumers — feed the text form to a model.

**Go:**

```go
obs, _ := browser.GetObservationWith(ctx, browserscale.ObservationOpts{
    Format: "json",
})
var parsed Observation
_ = json.Unmarshal([]byte(obs), &parsed)
```

**TypeScript:**

```ts
const obs = await browser.getObservation({ format: "json" });
const parsed = JSON.parse(obs);
```

## GetDOM — the full CDP tree

When the observation isn't enough — you need *every* node, the
nesting, the full attributes — switch to `GetDOM`. The payload is a
JSON string in standard CDP `DOM.Node` shape, with same-origin
`<iframe>` / `<frame>` / `<object>` children inlined into the same
tree. Cross-origin (out-of-process) frames stop the tree; call
`GetDOM` again with that frame's `frameId` to descend.

It is a snapshot, and the inlining is a one-time copy. A [live DOM
mirror](/docs/guides/dom-mirror) gives you the same `DOM.Node` shape
as one tree that stays current across *every* frame, cross-origin ones
included, with no per-frame recursion on your side.

**Go:**

```go
// Full tree of the main frame.
domJson, err := browser.GetDOM(ctx, "", -1)
if err != nil {
    log.Fatal(err)
}
// domJson is a CDP DOM.Node tree — feed it into anything that speaks CDP.
```

**TypeScript:**

```ts
// Full tree of the main frame.
const { dom } = await browser.getDOM();
// dom is a CDP DOM.Node tree — feed it into anything that speaks CDP.
```

The return shape differs across the two SDKs in one cosmetic way: Go
gives you the JSON string directly, TypeScript wraps it in a
`{ dom, hash }` object where `hash` is reserved for future use and is
**not** populated by this call. Either way, when you want the
fingerprint, call [`GetDOMHash`](#getdomhash--the-change-detector).

Two parameters tune the call:

- **`frameId`** (Go: positional, TS: positional) — empty / omitted
  targets the main frame. Use a specific frame's id to descend into
  an OOPIF.
- **`depth`** (Go: positional, TS: `opts.depth`) — `-1` for the full
  tree, `0` for the root only, `N` for the root plus N descendant
  levels.

**Go:**

```go
// Just the top two levels — cheap probe before pulling the whole thing.
shallow, _ := browser.GetDOM(ctx, "", 2)
```

**TypeScript:**

```ts
// Just the top two levels — cheap probe before pulling the whole thing.
const { dom } = await browser.getDOM("", { depth: 2 });
```

## GetDOMHash — the change detector

`GetDOMHash` returns the first 8 bytes of `sha256(dom)` as a 16-char
hex string. Computing it on the server is much cheaper than
transferring the full tree, so it answers *"is the content still
identical to what I saw?"* without a download.

Two better options exist for the questions people usually ask it,
though. For *"did this document move since I last asked?"* prefer
[`GetDomRevision`](/docs/guides/dom-mirror#polling-instead-getdomrevision):
a hash has to serialize the whole tree to compute one, while a
revision is a counter the browser already keeps, so it is O(1). And if
you want to *follow* the page rather than poll it, use a [live DOM
mirror](/docs/guides/dom-mirror) — it reports what changed instead of
making you re-fetch and diff.

The loop below still works, and is fine as a one-off check. On a page
that changes often it re-serializes the whole document every time the
hash moves, which is the cost the mirror exists to avoid:

**Go:**

```go
var lastHash string
for {
    hash, err := browser.GetDOMHash(ctx, "")
    if err != nil {
        return err
    }
    if hash != lastHash {
        lastHash = hash
        domJson, _ := browser.GetDOM(ctx, "", -1)
        process(domJson)
    }
    time.Sleep(500 * time.Millisecond)
}
```

**TypeScript:**

```ts
let lastHash = "";
while (running) {
    const hash = await browser.getDOMHash();
    if (hash !== lastHash) {
        lastHash = hash;
        const { dom } = await browser.getDOM();
        process(dom);
    }
    await sleep(500);
}
```

Don't reach for this as a wait substitute. If you're trying to wait
for "the page to stop changing", use `Wait` with a CSS or JS
condition for the actual element you care about — see
[Waiting](/docs/guides/waiting).

## InspectAtPosition — what's under (x, y)?

A hit-test at a viewport-relative pixel coordinate, returning the
topmost element under that point. Elements with
`pointer-events: none` are skipped, so the result is the actual click
target — not the visually-topmost node. This is what the live-UI
hover overlay calls under the hood.

**Go:**

```go
res, err := browser.InspectAtPosition(ctx, 200, 300)
if err != nil {
    log.Fatal(err)
}
fmt.Println(res.TagName, res.TextContent)
// res.BackendNodeId == 0 means nothing was found.
```

**TypeScript:**

```ts
const r = await browser.inspectAtPosition(200, 300);
console.log(r.tagName, r.textContent);
// r.backendNodeId === 0 means nothing was found.
```

The result carries everything you need to act on the element next:
`BackendNodeId`, `FrameId`, `TagName`, trimmed `TextContent`,
post-scroll `IsVisible` and the bounding rect.

Typical use cases:

- *Stream-based UIs* where the user clicks on the video feed and the
  script needs to translate the click into a real DOM target.
- *Coordinate-driven recipes* (canvas/captcha tile, HTML5 game)
  where you want to verify *what's actually there* before firing a
  `Click(at(...))`.

## HighlightNode — the debug overlay

The visual companion to `InspectAtPosition`. Paints a coloured
overlay on top of the node identified by `backendNodeId`. The overlay
stays until the next call — pass a non-positive `backendNodeId` to
clear it.

**Go:**

```go
r, _ := browser.InspectAtPosition(ctx, 400, 250)
_ = browser.HighlightNode(ctx, r.BackendNodeId, r.FrameId)
// ... screenshot or just watch the live stream ...
_ = browser.HighlightNode(ctx, 0, "")             // clear the overlay
```

**TypeScript:**

```ts
const r = await browser.inspectAtPosition(400, 250);
await browser.highlightNode(r.backendNodeId, r.frameId);
// ... screenshot or just watch the live stream ...
await browser.highlightNode(0);                   // clear the overlay
```

Pure debugging affordance — `HighlightNode` only paints visuals, it
doesn't change the page's behaviour. Use it freely in development; in
production scripts there's usually no reason to call it.

## GetSelection — read what's highlighted

Walks every frame and returns the first non-empty text selection it
finds. Returns `""` when nothing is selected anywhere. Useful for
"copy what the user highlighted" flows and for tests that exercise
selection-based UI (e.g. "did our right-click translate this text?").

**Go:**

```go
sel, err := browser.GetSelection(ctx)
if err != nil {
    log.Fatal(err)
}
if sel == "" {
    fmt.Println("nothing selected")
} else {
    fmt.Println("user selected:", sel)
}
```

**TypeScript:**

```ts
const sel = await browser.getSelection();
if (sel === "") {
    console.log("nothing selected");
} else {
    console.log("user selected:", sel);
}
```

## When to reach for which

| You want… | Use |
| --- | --- |
| A short summary an LLM can read | `GetObservation` |
| Every node and attribute, once | `GetDOM` |
| The tree, kept up to date as the page changes | [A live DOM mirror](/docs/guides/dom-mirror) |
| "Did this document move since I last asked?" | [`GetDomRevision`](/docs/guides/dom-mirror#polling-instead-getdomrevision) — O(1) |
| "Is the content still byte-identical?" | `GetDOMHash` |
| The element under a specific pixel | `InspectAtPosition` |
| A visual marker on a node while debugging | `HighlightNode` |
| The text the user is currently highlighting | `GetSelection` |
| To wait for something to appear | Not these — use [`Wait`](/docs/guides/waiting) |

## Gotchas

- **They don't wait.** None of these methods polls for an element to
  appear. If you call `GetObservation` before the page has rendered,
  you get whatever was visible at that instant. Use `Wait` first for
  the anchor element you care about.
- **TS `getDOM` returns `{ dom, hash }`; Go returns just the DOM
  string.** Cosmetic wrapper difference only — in TS the `hash`
  field is reserved for future use and is not populated by `getDOM`.
  Call `getDOMHash` explicitly in both SDKs.
- **OOPIFs stop the DOM tree.** `GetDOM` inlines same-origin frames
  but stops at cross-origin ones. Recurse by calling `GetDOM` again
  with the OOPIF's `frameId` (look it up via `GetPages` — covered in
  [Frames & iframes](/docs/guides/frames)).
- **`InspectAtPosition` returns `BackendNodeId == 0` for misses.**
  Treat that as "nothing there" rather than an error.
- **`HighlightNode` is sticky.** The overlay persists until you call
  again — make sure your cleanup path clears it (`backendNodeId = 0`)
  or your screenshots will keep showing stale highlights.

## See also

- [Live DOM mirror](/docs/guides/dom-mirror) — the same tree, but followed instead of re-fetched, and one tree across every frame.
- [Evaluation](/docs/guides/evaluation) — when you need to *run JavaScript* instead of (or after) reading the structure.
- [Waiting](/docs/guides/waiting) — the explicit pause to put in front of any read that depends on a specific element.
- API reference: [Go DOM helpers](/docs/api-reference/go#GetObservation) · [TS DOM helpers](/docs/api-reference/ts#getObservation).

→ Continue: [Live DOM mirror](/docs/guides/dom-mirror)
