<!--
  url: https://browserscale.cloud/docs/guides/loading
  title: Page loading
  description: Navigate, commit vs. load events, and the right signal to wait for before driving a page with browserscale.
-->

# Loading pages

Two SDK methods put content on the page: `Navigate` fetches it from
the network, `LoadHTML` serves it from a buffer you supply. Most
scripts only ever need the first; the second is the trick that turns
fixtures and snapshotted replays from a research project into a
two-liner.

> **TL;DR**
>
> - `Navigate(url)` opens a URL and returns *as soon as the response commits* — before `DOMContentLoaded`, before `load`.
> - `LoadHTML(url, html)` installs a one-shot interceptor; the next request to `url` gets your bytes back instead of going to the network.
> - Combine them: `LoadHTML(...)` + `Navigate(...)` is the canonical pattern for "render this fixture as if it were a real page".
> - Both methods change *what's on the page* but neither waits for *anything visible* — pair them with `Wait` before you act.

## Navigate

The default way to load a page. Pass a URL, get back the frame id of
the navigated frame plus the final URL after any redirects.

**Go:**

```go
r, err := browser.Navigate(ctx, "https://example.com", 0)
if err != nil {
    log.Fatal(err)
}
fmt.Println("landed on:", r.Url, "in frame", r.FrameId)
```

**TypeScript:**

```ts
const r = await browser.navigate("https://example.com");
console.log("landed on:", r.url, "in frame", r.frameId);
```

A few things worth knowing about the semantics:

- **Returns on commit, not on load.** The call resolves the moment the
  server's response is received and a new document is selected. JS has
  not run yet, `DOMContentLoaded` has not fired, images have not
  loaded. If you need the page to be *ready*, follow up with a `Wait`
  — that is the explicit contract in browserscale and exactly what the
  [Waiting guide](/docs/guides/waiting) covers.
- **Redirects are followed.** Whatever URL you actually land on comes
  back as `Url` / `url` in the result, not the URL you asked for.
- **Cross-origin is fine.** The session keeps its proxy, fingerprint
  and cookies across origins for the rest of its lifetime.

### Custom timeout

The default navigation timeout is **30 s**. If the host is slow or
unreachable, the call returns an error at that point. Override per call
when you need a different budget:

**Go:**

```go
// Give the host 5 seconds, not 30.
_, err := browser.Navigate(ctx, "https://slow.example", 5000)
```

**TypeScript:**

```ts
// Give the host 5 seconds, not 30.
await browser.navigate("https://slow.example", { timeoutMs: 5000 });
```

In Go it is the third positional argument (`0` falls back to the server
default). In TypeScript it is `opts.timeoutMs`. There is no separate
*"wait for `load` instead of commit"* mode — the commit-only behaviour
is intentional and not configurable.

### The `Navigate → Wait` pair

Because `Navigate` does not wait for anything visible, the canonical
loading pattern is two calls:

**Go:**

```go
_, _ = browser.Navigate(ctx, "https://shop.example/checkout", 0)
_, _ = browser.Wait(ctx, browserscale.CSS("button.pay"))   // page is now usable
_, _ = browser.Click(ctx, browserscale.CSS("button.pay"))
```

**TypeScript:**

```ts
await browser.navigate("https://shop.example/checkout");
await browser.wait(css("button.pay"));            // page is now usable
await browser.click(css("button.pay"));
```

Treat that triple — `Navigate → Wait → act` — as the default opening
move for any new flow. It is the price of browserscale's "no implicit waiting"
design and it pays back as scripts that survive page-load changes
later.

## LoadHTML

`LoadHTML` doesn't load anything by itself. It registers a **one-shot
response interceptor** that intercepts the *next* navigation to a given
URL and serves your bytes instead of going to the network. Once that
interceptor fires, it is gone.

The typical use is *"render this fixture as if it came from the real
domain"* — useful for tests where you need the page in a specific
state, offline replays of pages you captured earlier, or sandboxed
fragments you want to drive without hitting a third-party server.

The flow is always the same two steps:

**Go:**

```go
// 1. Tell the server: "if anyone navigates to this URL next, give them this body".
_ = browser.LoadHTML(ctx,
    "https://example.com/checkout",
    \`<!doctype html><h1>Checkout</h1><button class="pay">Pay</button>\`,
    nil,
    0,
)

// 2. Actually trigger the load.
_, _ = browser.Navigate(ctx, "https://example.com/checkout", 0)

// From here on it behaves like any other page.
_, _ = browser.Wait(ctx, browserscale.CSS("button.pay"))
```

**TypeScript:**

```ts
// 1. Tell the server: "if anyone navigates to this URL next, give them this body".
await browser.loadHTML(
    "https://example.com/checkout",
    \`<!doctype html><h1>Checkout</h1><button class="pay">Pay</button>\`,
);

// 2. Actually trigger the load.
await browser.navigate("https://example.com/checkout");

// From here on it behaves like any other page.
await browser.wait(css("button.pay"));
```

Two things are easy to miss the first time:

- The URL you pass to `LoadHTML` is the URL the page will *appear* to
  have loaded — it ends up in `document.location.href`, it scopes the
  cookies, it is what JavaScript sees. Keep it accurate; if your
  fixture pretends to be `example.com`, scripts on the page will run
  against that origin.
- The interceptor is **one-shot**. After it serves the first matching
  request it disappears. To replay the same URL again, call `LoadHTML`
  again before the next `Navigate`.

### Custom status code and headers

By default the synthetic response is `200 OK` with
`Content-Type: text/html; charset=utf-8`. Pass extra headers and
status codes when you need to simulate something more interesting:

**Go:**

```go
// Simulate a server error page — what does our retry logic do?
_ = browser.LoadHTML(ctx,
    "https://api.example.com/checkout",
    \`<h1>Service unavailable</h1>\`,
    []browserscale.Header{
        {Name: "X-Trace-Id", Value: "fixture-1"},
    },
    503,
)
_, _ = browser.Navigate(ctx, "https://api.example.com/checkout", 0)
```

**TypeScript:**

```ts
// Simulate a server error page — what does our retry logic do?
await browser.loadHTML(
    "https://api.example.com/checkout",
    \`<h1>Service unavailable</h1>\`,
    {
        headers: [{ name: "X-Trace-Id", value: "fixture-1" }],
        statusCode: 503,
    },
);
await browser.navigate("https://api.example.com/checkout");
```

The status code reaches the page exactly as a real server would
deliver it, which makes `LoadHTML` a clean way to test how your script
reacts to 3xx/4xx/5xx responses without coordinating an actual backend.

## When to reach for which

A quick mental decision tree:

| You want… | Use |
| --- | --- |
| Load a real page from its real URL | `Navigate` |
| Load a real page, but skip the slow `load` event | `Navigate` (it already returns on commit) |
| Render HTML you have in memory or on disk | `LoadHTML` + `Navigate` |
| Test how the page reacts to an error response | `LoadHTML` with `statusCode` + `Navigate` |
| Replay a snapshot of a page you captured earlier | `LoadHTML` (one per navigation) + `Navigate` |
| Inject HTML into the *current* page mid-flight | Not `LoadHTML` — use `Evaluate` to mutate the DOM |

The last row trips people up: `LoadHTML` is about *the next
navigation's response*, not about appending or replacing content on a
page that's already open. For DOM-level changes mid-flight, the right
tool is `Evaluate`.

## Gotchas

- **Commit semantics confuse Playwright veterans.** Playwright's
  `page.goto()` waits for `load` by default. browserscale's `Navigate` does not
  — it returns on document commit. If you're porting a script across,
  add an explicit `Wait` after every `Navigate`.
- **`LoadHTML` is one-shot per call.** Multiple navigations to the
  same URL need multiple `LoadHTML` calls. If you forget, the second
  navigation hits the real network — and on a non-existent URL that
  means a real failure.
- **URL match is exact.** The URL you pass to `LoadHTML` has to match
  what the page actually requests, character for character —
  trailing-slash, query-string and fragment differences all miss. Use
  the URL the page is going to request *literally*.
- **`LoadHTML` doesn't bypass anti-bot.** Once the synthetic response
  is served, the resulting page is a normal page in a normal browser
  — same fingerprint, same cookies, same JS execution. Use it for
  fixtures, not as a circumvention layer.

## See also

- [Waiting](/docs/guides/waiting) — the `Navigate → Wait → act` pair lives here.
- [Network](/docs/guides/network) — for general request interception, header rewriting and blocklists; `LoadHTML` is the focused single-shot variant of the same machinery.
- API reference: [Go `Navigate` / `LoadHTML`](/docs/api-reference/go#Navigate) · [TS `navigate` / `loadHTML`](/docs/api-reference/ts#navigate).

→ Continue: [Interaction](/docs/guides/interaction)
