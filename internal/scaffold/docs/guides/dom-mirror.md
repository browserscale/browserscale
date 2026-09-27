<!--
  url: https://browserscale.cloud/docs/guides/dom-mirror
  title: Live DOM mirror
  description: Hold a browserscale page's DOM as one live tree across every frame, updated incrementally, reporting changes only inside the part you expanded.
-->

# Live DOM mirror

`GetDOM` hands you the document as it was when you asked. To find out
whether it still looks like that, you have to ask again — which is why
[`GetDOMHash`](/docs/guides/reading#getdomhash--the-change-detector)
exists, and why the usual shape is a polling loop that re-serializes
the whole page to notice that one attribute changed.

A mirror inverts that. You get the top of the tree once, and from then
on the browser tells you what moved — but only inside the part you
actually expanded. A page rewriting a list sixty times a second inside a
collapsed subtree costs you one number per batch instead of a
re-serialized document. This is what makes a DevTools-style tree view
practical against a live page.

> **TL;DR**
>
> - `MirrorDom` returns a `DomMirror` holding the tree, plus a handler called after every change.
> - **One mirror covers the whole page.** An `<iframe>` is an ordinary element whose single child is the document it hosts.
> - `Expand` fetches a node's children *and* starts reporting changes inside them. `Collapse` stops again. That boundary is the whole performance story.
> - Nodes are immutable. A change replaces the nodes from the root down to the one that moved and leaves everything else identical, so `React.memo` and friends skip the untouched parts.
> - A node's address is the pair (`FrameId`, `BackendNodeId`). Ids restart per frame, so the id alone is ambiguous.
> - `GetDomRevision` is the O(1) change detector if you would rather poll than consume events. Prefer it over `GetDOMHash`.

## The model

Three ideas carry the whole API, and the rest follows from them.

**It is one tree.** Not a tree per frame with some stitching on your
side. An `<iframe>` appears as the element it is, and the document it
hosts is its one child — present once the element has been expanded.
Nothing about walking the tree has to know that a process boundary runs
through it. Underneath there is still one observer per document, because
a cross-origin iframe is a different document in a different process,
but that is engine bookkeeping rather than something you model.

**Expanding is subscribing.** `Expand` is not just "fetch children" —
it moves the boundary of what the browser reports. Inside an expanded
node you get real changes; inside a collapsed one you get an updated
`ChildNodeCount` and nothing else. Frames nobody opened cost nothing at
all.

**A node's address is a pair.** `BackendNodeId` values are handed out
per renderer and restart per frame, so two frames can and do use the
same number. Every lookup takes `(FrameId, BackendNodeId)`, and every
`DomNode` carries its own `FrameId`.

## Start a mirror

`MirrorDom` takes options, a change handler, and an optional resync
handler. It returns once the mirror is running and the first snapshot is
in place:

**Go:**

```go
mirror, err := browser.MirrorDom(ctx, browserscale.DomMirrorOptions{Pierce: true},
    func(m *browserscale.DomMirror) {
        render(m.Root())
    },
    func(reason string) {
        log.Printf("mirror rebuilt: %s", reason)
    })
if err != nil {
    log.Fatal(err)
}
defer mirror.Stop(context.Background())
```

**TypeScript:**

```ts
const mirror = await browser.mirrorDom(
    { pierce: true },
    () => render(mirror.root),
    (reason) => console.log("mirror rebuilt:", reason),
);

try {
    // ... drive the browser ...
} finally {
    await mirror.stop();
}
```

The subscription is established before the snapshot is taken, so no
change between the two is lost. As with a network capture, `Stop` is
what stops the mirror server-side — a cancelled context only shuts down
your local reader.

Two options, both fixed for the life of the mirror:

| Option | Meaning |
| --- | --- |
| `Depth` | Levels to serialize up front. `0` uses the server default of 2 — `#document` → `<html>` → `<head>`/`<body>`, enough to draw a collapsed tree. `-1` walks everything and gives up what the mirror is for. |
| `Pierce` | Descend into author shadow roots, exposing them as `ShadowRoots` on their host. |

## Expand and collapse

This is the pair a tree view calls when the user opens and closes nodes.
`Expand` takes a node you hold and a depth (`0` uses the server default
of 1):

**Go:**

```go
body := mirror.Node(mirror.MainFrameId(), bodyId)
if err := mirror.Expand(ctx, body, 1); err != nil {
    log.Fatal(err)
}

for _, child := range mirror.Root().Children {
    fmt.Println(child.NodeName, child.ChildNodeCount)
}

// Later, when the user closes it again:
_ = mirror.Collapse(ctx, body)
```

**TypeScript:**

```ts
const body = mirror.getNode(mirror.mainFrameId, bodyId);
if (body) await mirror.expand(body, 1);

for (const child of mirror.root?.children ?? []) {
    console.log(child.nodeName, child.childNodeCount);
}

// Later, when the user closes it again:
if (body) await mirror.collapse(body);
```

`IsExpanded` tells you which state a node is in, which is what a
disclosure triangle renders from. `Children` is non-nil once a node has
been expanded — and non-nil but empty for an expanded node that has
none, so it distinguishes "no children" from "not asked yet".

**Collapse what you stop looking at.** Skipping it is not an error, it
is a slow leak: the browser's revealed set only ever grows, and a set
that contains everything is no longer filtering anything. The cost you
avoided by not expanding comes back.

A node that left the page while your `Expand` was in flight is not an
error and changes nothing.

## Frames are just elements

Expanding an `<iframe>` fetches the document it hosts and starts
mirroring that frame — however deeply nested, and whether or not it is
cross-origin. There is no frame lifecycle to subscribe to and no
per-frame session to manage.

Two fields matter here. `FrameId` is the frame a node lives *in*.
`ContentFrameId` is set on an element that hosts a frame (`<iframe>`,
`<frame>`, `<object>`) and names the frame it hosts — which is the frame
its child document's ids belong to.

**Go:**

```go
// Walk into whatever this iframe hosts.
if frame.ContentFrameId != "" {
    _ = mirror.Expand(ctx, frame, 1)
    doc := frame.Children[0] // the hosted #document
    fmt.Println("frame", doc.FrameId, "has", doc.ChildNodeCount, "children")
}

fmt.Println("mirrored frames:", mirror.FrameIds())
```

**TypeScript:**

```ts
// Walk into whatever this iframe hosts.
if (frame.contentFrameId) {
    await mirror.expand(frame, 1);
    const doc = frame.children?.[0]; // the hosted #document
    console.log("frame", doc?.frameId, "has", doc?.childNodeCount, "children");
}

console.log("mirrored frames:", mirror.frameIds);
```

`FrameIds` lists every frame that currently has a document in the tree,
main frame first. A frame whose `<iframe>` has not been expanded is not
mirrored and not listed. When a frame navigates, it arrives as its
owner element's child being replaced — the same edit as any other.

## Immutability, and why rendering is cheap

Nodes are immutable once handed out. Applying a change replaces the
nodes from the root down to the one that moved and leaves every other
object identical. Two things follow:

- A tree you took from `Root` stays a consistent snapshot to walk while
  the mirror moves on. No locking, no tearing.
- Unchanged subtrees keep their identity, so an identity check is
  enough to skip them. Rendering straight from `root` with memoized
  components does the minimum work by construction.

```tsx
const Node = React.memo(function Node({ node }: { node: DomNode }) {
  // Skipped entirely when `node` is identical to last render — which it
  // is for every subtree the change did not touch.
  return (
    <li>
      {node.nodeName.toLowerCase()}
      {node.children && (
        <ul>
          {node.children.map((c) => (
            <Node key={`${c.frameId}:${c.backendNodeId}`} node={c} />
          ))}
        </ul>
      )}
    </li>
  );
});
```

Note the key: `(frameId, backendNodeId)`, for the reason from the model
section. Keying on `backendNodeId` alone breaks the moment a second
frame is expanded.

Do not modify the nodes. The mirror hands out its own state.

## The change handler

Treat it as **"something moved, re-read the root"** rather than as one
call per edit. Calls are serialized, so the handler needs no locking of
its own, and it is safe to call back into the mirror from it. They are
also coalesced: several changes in quick succession may produce a single
call, which always sees the newest tree.

The handler also fires once for the opening snapshot, so a UI that
renders from it needs no separate initial-render path.

`Seq` is the page sequence of the last change applied — one clock for
the whole page, so a change in an out-of-process iframe and one in the
main document are ordered against each other.

## Reach a node you do not hold

`Expand` needs a node from your tree. For a node that is not in it —
an [`InspectAtPosition`](/docs/guides/reading#inspectatposition--whats-under-x-y)
hit, say — there is nothing to walk down from, because you do not have
the ancestors either. `Reveal` builds that chain for you:

**Go:**

```go
hit, err := browser.InspectAtPosition(ctx, x, y)
if err != nil {
    log.Fatal(err)
}

chain, err := mirror.Reveal(ctx, hit.BackendNodeId, hit.FrameId)
if err != nil {
    log.Fatal(err)
}
for _, ancestor := range chain {
    fmt.Println(ancestor.NodeName)
}
```

**TypeScript:**

```ts
const hit = await browser.inspectAtPosition(x, y);

const chain = await mirror.reveal(hit.backendNodeId, hit.frameId);
for (const ancestor of chain) {
    console.log(ancestor.nodeName);
}
```

It returns the chain from the main document down, each ancestor
carrying its own children, and splices it into the tree. The target may
sit in a frame nobody opened — that works, and the frames along the way
start being mirrored, exactly as if you had expanded your way there by
hand. This is what makes a hit test usable against a lazily loaded tree.
An empty result means the node is not on the page.

## Resyncs

Sometimes the local copy is void and the only correct move is to throw
it away and start again. That happens automatically; the resync handler
exists to explain to a user why their expanded nodes just collapsed.

| Reason | What happened |
| --- | --- |
| `documentReplaced` | The page navigated. Every id from before is meaningless. |
| `overflow` | Changes arrived faster than they could be described incrementally. |
| `rendererGone` | The renderer process died or was replaced. |
| `slowReader` | This reader fell far enough behind that the server stopped keeping its backlog. |
| `manual` | You called `Resync` yourself. |

Treat the set as open — a newer browser may add one. After a resync the
tree is back to the opening depth, so anything the user had expanded
needs re-expanding; that is exactly what the handler is for.

## Polling instead: `GetDomRevision`

If you are not consuming mirror events at all and just want to know
whether the document moved, `GetDomRevision` is a frame's mutation
counter — incremented on every change the document sees, O(1) in the
browser:

**Go:**

```go
before, _ := browser.GetDomRevision(ctx, "")
_, _ = browser.Click(ctx, browserscale.CSS("button#load-more"))

after, _ := browser.GetDomRevision(ctx, "")
if after != before {
    fmt.Println("the document changed")
}
```

**TypeScript:**

```ts
const before = await browser.getDomRevision("");
await browser.click(css("button#load-more"));

const after = await browser.getDomRevision("");
if (after !== before) {
    console.log("the document changed");
}
```

Prefer it over `GetDOMHash`, which serializes the entire tree just to
hash it. The two answer different questions: a hash compares *content*,
a revision only says whether this document *moved* since you last
asked. The counter is meaningful only within the current document — a
navigation resets what it is counting.

## Gotchas

- **The id alone is not an address.** Always pair `BackendNodeId` with
  `FrameId`, in lookups and in render keys.
- **`Depth: -1` defeats the purpose.** It walks the entire page up
  front, which is the cost a mirror exists to avoid. Start shallow and
  expand.
- **Not collapsing is a slow leak.** The revealed set only grows.
  Collapse what the user closed.
- **Changes inside a collapsed node are invisible** except as an
  updated `ChildNodeCount`. If you expected an event and got none,
  check `IsExpanded`.
- **The change handler is coalesced, not one-per-edit.** Do not count
  calls or try to diff from them; re-read `Root`.
- **A cancelled context leaves the mirror running server-side.** Only
  `Stop` / `StopDomMirror` stops it.
- **Ids from before a resync are stale.** Reading children for one
  comes back empty rather than failing, which looks exactly like a node
  with no children.
- **`Pierce` is fixed at start.** To change it, stop the mirror and
  start a new one.
- **`Attributes` is a flat list** — `name, value, name, value, …` — as
  CDP sends it, not a map.
- **To stop from inside a handler, call `StopDomMirror` on the
  browser.** `Stop` waits for the handler to return, so calling it
  there waits on itself.

## See also

- [Reading the page](/docs/guides/reading) — `GetObservation`, `GetDOM` and the one-shot reads a mirror does not replace.
- [Frames & iframes](/docs/guides/frames) — what `FrameId` and `ContentFrameId` refer to.
- [Capturing traffic](/docs/guides/capture) — the same streaming shape, applied to the network instead of the document.
- API reference: [Go `MirrorDom`](/docs/api-reference/go#MirrorDom) · [TS `mirrorDom`](/docs/api-reference/ts#mirrorDom).

→ Continue: [Evaluation](/docs/guides/evaluation)
