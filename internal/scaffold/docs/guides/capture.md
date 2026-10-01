<!--
  url: https://browserscale.cloud/docs/guides/capture
  title: Capturing traffic
  description: Stream every request a browserscale session completes — cross-process iframes, workers and each redirect hop included — without ever pausing the page.
-->

# Capturing traffic

The levers in the [Network guide](/docs/guides/network) all answer a
question about *one* request: catch the next login call, rewrite this
one header, drop that beacon. Capture answers a different question —
what did this session actually do? It is a log, not a trap. Nothing is
paused, nothing is rewritten, and you get every request the browser
completed, in the order it finished them.

> **TL;DR**
>
> - `CaptureNetwork` takes options and a handler, and calls the handler once per completed request for as long as the capture runs.
> - Capture happens in the **browser process**, so cross-process iframes, workers and service workers are all in the log, the headers are the ones that went on the wire, and each hop of a redirect is its own entry.
> - `Patterns` are the same URL wildcards as everywhere else. Prefix one with `!` to exclude it.
> - Bodies are off by default. `Bodies: "text"` keeps textual payloads; `"all"` also keeps binary ones, which do not survive the trip intact.
> - Never block in the handler. The server buffers a bounded amount per reader, then drops its oldest entries — `Dropped` tells you it happened.
> - `Stop` disarms the capture server-side. A cancelled context does not.

## Capture or intercept?

Both read traffic, but they are built for opposite situations. Reach
for the one that matches the question you are asking:

| | Capture | `WaitForAnyRequest` / `WaitForAnyResponse` |
| --- | --- | --- |
| Lifetime | Runs until you stop it | One-shot, consumes the next match |
| Delivery | Handler, called per request | Return value of a single call |
| Effect on the page | None — requests are never paused | Can pause, `Abort`, or rewrite |
| Sees | Every completed request, all frames and workers | The one match you armed for |
| Ordering worry | None, start it whenever | Must be armed before the trigger |
| Use it for | Auditing, debugging, harvesting API payloads, DevTools-style logs | Asserting one call happens, stubbing it, injecting a header |

The "arm before you trigger" rule that dominates the one-shot
interceptors does not apply here. A capture is armed before
`CaptureNetwork` returns, so anything that happens after it returns is
in the log — there is no window to race.

## Start a capture

Pass what you want captured and a function to receive it. The call
returns as soon as the capture is running; the handler then fires in
the background while you drive the browser:

**Go:**

```go
capture, err := browser.CaptureNetwork(ctx, browserscale.NetworkCaptureOptions{
    Patterns: []string{"*/api/*"},
    Bodies:   browserscale.NetworkBodiesText,
}, func(ex browserscale.NetworkExchange) {
    fmt.Println(ex.StatusCode, ex.Method, ex.Url)
})
if err != nil {
    log.Fatal(err)
}
defer capture.Stop(ctx)

_, _ = browser.Navigate(ctx, "https://example.com", 0)
```

**TypeScript:**

```ts
const capture = await browser.captureNetwork(
    { patterns: ["*/api/*"], bodies: "text" },
    (ex) => console.log(ex.statusCode, ex.method, ex.url),
);

try {
    await browser.navigate("https://example.com");
} finally {
    await capture.stop();
}
```

The returned handle is not where the data comes from — exchanges only
ever reach the handler. The handle exists to stop the capture and to
tell you how it went.

## Why the log is complete

Capture sits in the browser process, not injected into a page. That
single fact is what the rest of this guide keeps coming back to:

- **Cross-process iframes are included.** A cross-origin `iframe` runs
  in its own renderer, which a page-level hook cannot see into. Here it
  is just another `FrameId`, with `IsOOPIF` telling you it was
  out-of-process.
- **Workers and service workers are included.** Traffic a dedicated or
  service worker issues has no page to hook. Worker exchanges arrive
  with an empty `FrameId`.
- **The headers are the real ones.** `RequestHeadersAreWire` reports
  that you are looking at the bytes actually sent — `Cookie`,
  `User-Agent`, `Sec-*` and everything else the network stack filled
  in — rather than the header bag the page handed to `fetch()`.
- **Requests are never paused.** Nothing waits for your handler, so
  the page loads at full speed and timing-sensitive sites behave
  normally.
- **Served-from is visible.** `ServedFrom` distinguishes a real network
  fetch from an HTTP-cache hit, a service-worker response, and
  browserscale's own static cache from
  [`SetStaticPaths`](/docs/guides/network#cache-static-assets-with-setstaticpaths).

## Choosing what to capture

`Patterns` uses the same wildcard vocabulary as the rest of the network
API — `*` matches any span, no regex. Omit it entirely to capture
everything the session does.

The one addition is the `!` prefix, which excludes. It is the short way
to say "everything except this", without having to enumerate what you
do want:

**Go:**

```go
// Everything except image and font noise.
opts := browserscale.NetworkCaptureOptions{
    Patterns: []string{"*", "!*.png", "!*.jpg", "!*.woff2"},
}
```

**TypeScript:**

```ts
// Everything except image and font noise.
const opts = {
    patterns: ["*", "!*.png", "!*.jpg", "!*.woff2"],
};
```

Narrowing here is the cheapest optimisation available: an exchange that
never matches is never buffered, never serialized and never sent.

## Bodies

**An exchange never carries a body.** It carries a body *id*. The
browser keeps the bytes on its side while the capture runs, and you
pull the ones you want with `ReadNetworkBody`. A capture that logs
thousands of requests therefore costs you nothing for the payloads you
never look at, and the event stream stays fast regardless of body size.

Request bodies are always kept. Which response bodies are kept is
chosen with `Bodies`:

| Setting | Keeps |
| --- | --- |
| `"none"` *(default)* | No response bodies. Headers, status and sizes only. |
| `"text"` | Bodies whose MIME type is textual — JSON, HTML, CSS, JS, XML, plain text. |
| `"all"` | Every body regardless of type, images and binaries included. |

Bodies are stored byte for byte, after content decoding (gzip, brotli),
so a PNG read back with `"all"` is the PNG the page received.

`BodyPatterns` narrows body capture further, independently of which
requests get logged. This is the combination you usually want: log
everything, keep only the payloads you care about.

**Go:**

```go
capture, err := browser.CaptureNetwork(ctx, browserscale.NetworkCaptureOptions{
    Patterns:     []string{"*"},               // log every request
    Bodies:       browserscale.NetworkBodiesText,
    BodyPatterns: []string{"*/api/*"},         // but only keep API payloads
}, func(ex browserscale.NetworkExchange) {
    if ex.ResponseBodyId != "" {
        ids <- ex.ResponseBodyId // read it outside the handler
    }
})

// elsewhere
body, truncated, err := browser.ReadNetworkBody(ctx, id)
```

**TypeScript:**

```ts
const capture = await browser.captureNetwork({
    patterns: ["*"],             // log every request
    bodies: "text",
    bodyPatterns: ["*/api/*"],   // but only keep API payloads
}, (ex) => {
    if (ex.responseBodyId) ids.push(ex.responseBodyId); // read it outside the handler
});

// elsewhere
const { data, truncated } = await browser.readNetworkBody(id);
const json = JSON.parse(new TextDecoder().decode(data));
```

The id answers the question an empty body cannot: an exchange whose
body was kept has an id, even if the body is zero bytes long; an
exchange whose body was not kept has none. `ResponseBodySize` tells
you how much there is before you read it.

`ReadNetworkBody` assembles the whole body in memory. For large
payloads, `ReadNetworkBodyRange` reads a slice by offset and length and
reports the total size, so you can stream it to disk or only look at
the first kilobytes.

### How long bodies live

Bodies belong to the session. They stay readable after the capture
stops — so "record, stop, then inspect" works — and are deleted when
the session closes. Starting a new capture does not discard the bodies
of the previous one.

Storage is bounded per body and per session, and the server owns both
limits. A body larger than the per-body limit is kept up to it and
flagged `Truncated`. When a session reaches its total, the oldest
bodies of finished requests make room first; reading one of those
fails with `evicted`. If you capture for hours, read what you need as
you go instead of at the end.

Reading fails with a `CommandError` whose code says why:

| Code | Meaning |
| --- | --- |
| `not_found` | No body with this id exists in this session. |
| `evicted` | It existed, but was dropped to stay within the session's storage. |
| `unavailable` | The body could not be stored or read back. |

## What an exchange carries

One `NetworkExchange` is a request together with the response it
received. The fields worth knowing, grouped by what they answer:

**Which request is this?**

| Field | What it holds |
| --- | --- |
| `RequestId` | Unique per hop. |
| `ChainId` | Shared by every hop of one redirect chain. |
| `RedirectIndex` | `0` for the original request, `+1` per redirect followed. |
| `FrameId` | The frame that issued it. Empty for worker traffic. |
| `IsOOPIF` | Whether that frame runs in its own process. |
| `ResourceType` | `document`, `subframe`, `script`, `stylesheet`, `image`, `font`, `media`, `fetch`, `worker`, `manifest`, `object`, `csp-report`, `other`. |
| `InitiatorUrl` | The origin that started it; empty when the browser itself did. |

**What went out?**

| Field | What it holds |
| --- | --- |
| `Method`, `Url` | Verb and full URL. |
| `RequestHeaders` | Name/value pairs, order preserved. |
| `RequestHeadersAreWire` | Whether those are the bytes actually sent. |
| `RequestBodyId`, `RequestBodySize` | The kept request body, read with `ReadNetworkBody`. Empty id when there was none. |
| `RequestBodyTruncated` | Part of the body is missing: it hit the size limit, or it was a file or streamed upload, which are not kept. |

**What came back?**

| Field | What it holds |
| --- | --- |
| `HasResponse` | `false` when the request failed before any response arrived — then read `Error`. |
| `StatusCode`, `StatusText`, `MimeType` | The response line and content type. |
| `Protocol` | Negotiated ALPN protocol, e.g. `h2` or `http/1.1`. |
| `RemoteAddress` | Who answered. |
| `ServedFrom` | `network`, `cache`, `serviceWorker`, `wrcStaticCache`, `wrcSynthetic`. |
| `EncodedDataLength` | Bytes on the wire, not body size. `0` for a cache hit. |
| `ResponseBodyId`, `ResponseBodySize` | The kept response body, decoded, read with `ReadNetworkBody`. Empty id when it was not kept. |
| `ResponseBodyTruncated` | The kept body is shorter than what the page received: it hit the size limit, or the load ended early. |
| `Error` | Net error name, e.g. `net::ERR_ABORTED`. Empty on success. |

`ResourceType` is a string rather than a closed enum, so an exchange
from a newer browser carries its value through instead of decoding to
nothing. Compare against the `NetworkResource*` constants in Go, or the
string union in TypeScript. Note that `fetch()`, `XMLHttpRequest` and
`EventSource` all report `fetch` — they are indistinguishable at the
capture point.

## Redirect chains arrive as hops

A redirect is not one request with a final URL. It is a `301`/`302`
that really happened, followed by a request to wherever it pointed, and
capture reports both. The hops share `ChainId` and count up
`RedirectIndex`, so you can reassemble the chain or ignore it:

**Go:**

```go
chains := map[string][]browserscale.NetworkExchange{}
capture, _ := browser.CaptureNetwork(ctx, browserscale.NetworkCaptureOptions{
    Patterns: []string{"*/login*"},
}, func(ex browserscale.NetworkExchange) {
    chains[ex.ChainId] = append(chains[ex.ChainId], ex)
})

// ... drive the browser ...
_ = capture.Stop(ctx)

for id, hops := range chains {
    fmt.Printf("chain %s: %d hop(s)\\n", id, len(hops))
    for _, hop := range hops {
        fmt.Printf("  [%d] %d %s\\n", hop.RedirectIndex, hop.StatusCode, hop.Url)
    }
}
```

**TypeScript:**

```ts
const chains = new Map();
const capture = await browser.captureNetwork(
    { patterns: ["*/login*"] },
    (ex) => {
        const hops = chains.get(ex.chainId) ?? [];
        hops.push(ex);
        chains.set(ex.chainId, hops);
    },
);

// ... drive the browser ...
await capture.stop();

for (const [id, hops] of chains) {
    console.log(\`chain \${id}: \${hops.length} hop(s)\`);
    for (const hop of hops) {
        console.log(\`  [\${hop.redirectIndex}] \${hop.statusCode} \${hop.url}\`);
    }
}
```

Each hop carries its own headers and status, which is what makes a
capture usable for debugging an auth flow: the `Set-Cookie` on the
302 is right there instead of being lost behind the final response.

## The handler contract

Two guarantees and one rule.

**Calls are sequential and ordered.** The handler is called once per
exchange, in the order the browser finished the requests, never
concurrently. It needs no locking of its own — the map-building example
above is safe exactly because of this. In Go it runs on a goroutine the
SDK owns, not the caller's.

**Everything it wrote is visible after `Stop`.** Once `Stop` returns,
the handler is no longer running and its writes are visible to the
stopping goroutine. That is what makes "accumulate into a slice, read
it after `Stop`" correct without a mutex.

**Do not block.** Nothing waits for your handler, but the server keeps
buffering while it runs, and its per-reader buffer is bounded — once
full, it drops its **oldest** entries. Hand slow work to a queue of
your own:

**Go:**

```go
// The handler only hands off; the writer does the slow part.
rows := make(chan browserscale.NetworkExchange, 1024)
go func() {
    for ex := range rows {
        writeToDatabase(ex) // slow, and off the capture's goroutine
    }
}()

capture, _ := browser.CaptureNetwork(ctx, opts, func(ex browserscale.NetworkExchange) {
    select {
    case rows <- ex:
    default: // your own overflow policy, not the server's
    }
})
```

**TypeScript:**

```ts
// The handler only enqueues; the drain loop does the slow part.
const queue = [];
const capture = await browser.captureNetwork(opts, (ex) => {
    queue.push(ex);   // never await in here
});

void (async () => {
    while (true) {
        const ex = queue.shift();
        if (ex) await writeToDatabase(ex);
        else await new Promise((r) => setTimeout(r, 10));
    }
})();
```

Check `Dropped` when the capture ends. Anything above zero means the
log has holes, and the fix is one of: make the handler cheaper, or
narrow `Patterns`. Reading bodies counts as slow work — collect the
ids in the handler and read them elsewhere.

## Stopping, and how a capture ends

`Stop` does two things: it disarms the capture server-side and shuts
the local reader down. **A cancelled context only does the second**, so
a capture whose context expired is still armed in the browser — always
`Stop`.

Pass a live context to it. The one the capture was created with may
already be cancelled by the time you stop, and that context covers the
disarm call itself.

**Go:**

```go
capture, err := browser.CaptureNetwork(ctx, opts, handler)
if err != nil {
    log.Fatal(err)
}
defer capture.Stop(context.Background()) // live ctx, not the possibly-cancelled one

// ... drive the browser ...

if capture.Dropped() > 0 {
    log.Printf("log has holes: %d dropped", capture.Dropped())
}
```

**TypeScript:**

```ts
const capture = await browser.captureNetwork(opts, handler);
try {
    // ... drive the browser ...
} finally {
    await capture.stop();
}

if (capture.dropped > 0) {
    console.warn("log has holes:", capture.dropped);
}
```

To capture for as long as the session lives rather than around a piece
of your own code, `Wait` blocks until the capture ends for any reason
and reports why. `Err` answers the same question without blocking, and
is `nil` both while running and after a clean stop.

One trap worth naming: **to stop from inside the handler, call
`StopNetworkCapture` on the browser, not `Stop` on the handle.** `Stop`
waits for the handler to return, so calling it from the handler would
wait on itself until the context expires.

## Reading a capture from somewhere else

`CaptureNetwork` arms and subscribes together, which is what you want
when one piece of code does both. When the reader lives elsewhere — a
different process, another tab, a later `ConnectSession` against the
same session — the two halves are available separately:

**Go:**

```go
// Process A: arm it and walk away.
_ = browser.StartNetworkCapture(ctx, browserscale.NetworkCaptureOptions{
    Patterns: []string{"*/api/*"},
})

// Process B: attach to whatever is running.
capture, _ := browser.StreamNetworkExchanges(ctx, func(ex browserscale.NetworkExchange) {
    fmt.Println(ex.Method, ex.Url)
})
_ = capture.Wait()
```

**TypeScript:**

```ts
// Tab A: arm it and walk away.
await browser.startNetworkCapture({ patterns: ["*/api/*"] });

// Tab B: attach to whatever is running.
const capture = await browser.streamNetworkExchanges(
    (ex) => console.log(ex.method, ex.url),
);
await capture.wait();
```

Several readers can watch the same capture, each with its own buffer,
so one slow reader drops its own entries without affecting the others.
Stopping a handle that came from `StreamNetworkExchanges` only detaches
that reader — it never disarms a capture the others may still be
using. To disarm from a subscriber, call `StopNetworkCapture`, which
reports whether a capture was running at all.

Calling `StartNetworkCapture` again replaces the running capture rather
than adding a second one. There is one capture per session.

## Gotchas

- **A cancelled context leaves the capture armed.** Only `Stop` /
  `StopNetworkCapture` disarms it server-side.
- **`Dropped` above zero means missing entries, not reordered ones.**
  The order you receive is always the order the browser finished the
  requests; drops remove the oldest, they never shuffle.
- **Exchanges carry ids, not bodies.** Read the bytes with
  `ReadNetworkBody`; an empty id means the body was not kept.
- **Bodies die with the session.** Read them before closing it, and
  read early on long captures — the oldest go first once storage is
  full.
- **Bodies are decoded.** `ResponseBodySize` is the size after
  gzip/brotli, which is why it is usually larger than
  `EncodedDataLength`.
- **Worker traffic has no `FrameId`.** Do not key your log by frame
  and expect service-worker requests to land somewhere.
- **`fetch`, `XHR` and `EventSource` all report `fetch`.** The
  distinction does not exist at the capture point.
- **`EncodedDataLength` is wire bytes.** A cache hit reports `0`, and
  a compressed response reports less than its body length. It is not
  a body-size field.
- **Don't block the handler** — covered above, and the single most
  common cause of a log with holes.
- **Capture observes, it never intervenes.** If you need to change or
  drop a request, that is [`ModifyRequest` /
  `Abort`](/docs/guides/network#modify-a-request-before-it-goes-out).

## See also

- [Network](/docs/guides/network) — blocking, caching, and the one-shot interceptors that can modify traffic.
- [Frames & iframes](/docs/guides/frames) — what `FrameId` and `IsOOPIF` refer to.
- [Live DOM mirror](/docs/guides/dom-mirror) — the same streaming shape, applied to the document instead of the network.
- API reference: [Go `CaptureNetwork`](/docs/api-reference/go#CaptureNetwork) · [TS `captureNetwork`](/docs/api-reference/ts#captureNetwork).

→ Continue: [Cookies & sessions](/docs/guides/cookies)
