<!--
  url: https://browserscale.cloud/docs/guides/reactions
  title: Reactions
  description: Handle cookie banners, modals and interstitials once: a reaction is carried out by the browser in every frame, while your flow keeps running.
-->

# Reactions

A cookie bar is not a step in your flow. Neither is the newsletter
modal that opens after eight seconds, the "rate this app" sheet or the
chat bubble that slides over the checkout button. They may or may not
appear, at no fixed point, and writing a probe for each of them before
every click is how a five-line flow becomes fifty. A **reaction** is
the other way round: you describe the interruption once, and the
browser deals with it whenever it shows up.

> **TL;DR**
>
> - `AddReaction(match)` registers a standing rule: when `match` appears, click it. `AddReactionWith(match, ReactionOpts{On: ...})` (Go) / `addReaction(match, { on })` (TS) clicks something else instead, such as the modal's close button.
> - The reaction is carried out by the browser itself, with the same click as `Click`: scroll, human-like path, pixel check. Nothing round-trips to your process when it fires.
> - It only acts while the pointer is idle and yields to any action in flight, so it slots into the gaps of an action that is already retrying: a click blocked by a modal lands because the reaction dismissed the modal meanwhile.
> - One-shot: it fires once and removes itself. `ListReactions` shows what is still armed, `RemoveReaction(id)` takes one down.
> - Scope it to one frame or to all of them with `.InAllFrames()` — including frames created after you registered it.

## Register once

Register reactions at the start of the flow. From then on they run
beside it.

**Go:**

```go
_, _ = browser.Navigate(ctx, "https://shop.example", 0)

// The consent banner, wherever it renders — including inside a
// consent-provider iframe.
_, err := browser.AddReaction(ctx,
    browserscale.CSS("button#accept-all").InAllFrames())
if err != nil {
    log.Fatal(err)
}

// The flow itself does not mention the banner at all.
_, _ = browser.Click(ctx, browserscale.CSS("a.product"))
_, _ = browser.Click(ctx, browserscale.CSS("button.add-to-cart"))
```

**TypeScript:**

```ts
await browser.navigate("https://shop.example");

// The consent banner, wherever it renders — including inside a
// consent-provider iframe.
await browser.addReaction(css("button#accept-all").inAllFrames());

// The flow itself does not mention the banner at all.
await browser.click(css("a.product"));
await browser.click(css("button.add-to-cart"));
```

`match` has to be a `CSS(...)` or `JS(...)` locator; `Node(...)` and
`At(...)` are rejected, because there is nothing to watch for. By
default the match has to be visible, the same as a wait condition —
add `.Visible(false)` to fire on an element that is present but not
yet shown.

## Click something else

Often the element that tells you the interruption is there is not the
one you want to press. Watch for the modal, click its close button:

**Go:**

```go
_, _ = browser.AddReactionWith(ctx,
    browserscale.CSS("#newsletter-modal"),
    browserscale.ReactionOpts{On: browserscale.CSS(".modal-close")},
)
```

**TypeScript:**

```ts
await browser.addReaction(css("#newsletter-modal"), {
    on: css(".modal-close"),
});
```

The `On` / `on` target is resolved in the frame the match was found
in, so a close button inside the same iframe as its modal needs no
frame id. `Button` and `ClickCount` change the click itself, the same
as on `ClickWith`.

## How it fits around your actions

A reaction never fights your script for the pointer. It waits until no
input action is in flight and the pointer is idle, then clicks. Because
`Click`, `Fill` and `Drag` keep re-trying their own target within their
budget, the two work together:

1. Your `Click` on the checkout button finds it covered by a modal and
   keeps working its way in.
2. In the gap, the reaction sees the modal and clicks its close button.
3. The click's next attempt finds the button exposed and lands.

Without the reaction the same click would end in a `ClickError` naming
the modal as the occluder — which is also how you find out which
reactions a site needs.

## One-shot, and what is still armed

A reaction fires once and removes itself, so a banner that comes back
later does not get clicked again by accident. If an interruption can
recur, register it again after it fired.

**Go:**

```go
id, _ := browser.AddReaction(ctx, browserscale.CSS(".chat-invite .dismiss"))

pending, _ := browser.ListReactions(ctx)
for _, r := range pending {
    log.Printf("armed: %s %s%s", r.ReactionID, r.MatchSelector, r.MatchJsExpression)
}

// false if it already fired or never existed.
removed, _ := browser.RemoveReaction(ctx, id)
_ = removed
```

**TypeScript:**

```ts
const id = await browser.addReaction(css(".chat-invite .dismiss"));

const pending = await browser.listReactions();
for (const r of pending) console.log("armed:", r.reactionId, r.matchSelector);

// false if it already fired or never existed.
const removed = await browser.removeReaction(id);
```

Reactions stay armed across navigations of the page and are torn down
when the page or the session ends.

## Reaction or wait?

Both watch the page for a condition, but they answer different
questions:

| The thing… | Use |
| --- | --- |
| may or may not appear, and only needs dismissing | a reaction |
| decides what your flow does next (success, error, challenge) | a [`Wait`](/docs/guides/waiting) with one condition per outcome |
| is the element you are about to act on | nothing — the action finds it |

## Gotchas

- **Scope matters for consent banners.** Many consent managers render
  inside their own iframe. Without `.InAllFrames()` the reaction only
  watches the main document.
- **Be specific.** A reaction on `button` clicks the first visible
  button it sees. Match the interruption, not a generic element.
- **Reactions click; they do not type.** For an interruption that needs
  input, wait for it as one of the outcomes and handle it in your flow.

## See also

- [Interaction](/docs/guides/interaction) — the click a reaction performs, and what `ClickError` reports when something is in the way.
- [Waiting](/docs/guides/waiting) — for the states that decide what happens next.
- [Frames & iframes](/docs/guides/frames) — how `InAllFrames` covers frames created later.
- API reference: [Go `AddReaction`](/docs/api-reference/go#AddReaction) · [TS `addReaction`](/docs/api-reference/ts#addReaction) and [`ReactionOpts`](/docs/api-reference/go#ReactionOpts).

→ Continue: [Frames & iframes](/docs/guides/frames)
