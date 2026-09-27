<!--
  url: https://browserscale.cloud/docs/api-reference/go
  title: Go SDK Reference
  description: Complete reference for the github.com/browserscale/browserscale-go module: context-first methods, Method / MethodWith pairs, error codes, and runnable examples.
-->

# Go SDK Reference

Auto-generated from Go doc-comments. Every method takes `ctx context.Context` as its first argument and returns an explicit `error`; both are elided from the signatures below for brevity.

## CloudBrowser

CloudBrowser is the SDK-side handle for an active browserscale browser session.

One CloudBrowser corresponds to exactly one browser context, which is implicitly bound to its primary page server-side. The proto's page_id field is currently ignored server-side, so the SDK never sets it.

### `AcceptLanguage() → string`

*method on `CloudBrowser`*

AcceptLanguage returns the Accept-Language header value the session was provisioned with.

**Returns:** `string`

### `AddReaction(match *Locator) → string`

*method on `CloudBrowser`*

AddReaction registers a one-shot "reaction": a background poller (one shared loop per page) watches for the match locator and, as soon as it matches, clicks it with the full smart-click machinery (scroll, human path, occlusion gate, evade) — then removes itself. The poller yields to any in-flight input action and only fires while the pointer is idle, so a reaction naturally slots into the gaps of a retrying foreground action (e.g. it dismisses a newsletter modal blocking a CloudBrowser.Click, after which the click's own retry succeeds). Reactions are scoped to the page and torn down automatically when the page/session ends.

match must be a CSS or JS Locator — Node and At are rejected. Use Locator.InAllFrames to watch every frame and Locator.Visible(false) to opt out of the default visibility gate.

**Parameters:**
- `match` (`*Locator`) — the CSS/JS locator to watch for

**Returns:** `string` — the reactionId (pass to CloudBrowser.RemoveReaction)

**Throws:**
- `INVALID_LOCATOR` — match is nil, has no selector/JS expression, or is

```go
// Auto-dismiss a consent button whenever it appears, in any frame.
id, err := browser.AddReaction(ctx, browserscale.CSS("button#accept").InAllFrames())
if err != nil {
    log.Fatal(err)
}
_ = id
```

**See also:** CloudBrowser.AddReactionWith for a different click target, button,

### `AddReactionWith(match *Locator, opts ReactionOpts) → string`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.AddReaction`*

AddReactionWith is the customizable variant of CloudBrowser.AddReaction.

match must be a CSS or JS Locator — Node and At are rejected. Use Locator.InAllFrames to watch every frame and Locator.Visible(false) to opt out of the default visibility gate.

**Parameters:**
- `match` (`*Locator`) — the CSS/JS locator to watch for
- `opts` (`ReactionOpts`) — reaction customization; see ReactionOpts

**Returns:** `string` — the reactionId (pass to CloudBrowser.RemoveReaction)

**Throws:**
- `INVALID_LOCATOR` — match is nil, has no selector/JS expression, or is

```go
// Watch for a newsletter modal, but click its close "X" instead.
id, err := browser.AddReactionWith(ctx,
    browserscale.CSS("#newsletter-modal"),
    browserscale.ReactionOpts{On: browserscale.CSS(".modal-close")},
)
```

**See also:** CloudBrowser.AddReactionWith for a different click target, button,

### `ApiKey() → string`

*method on `CloudBrowser`*

ApiKey returns the API key used to rent this session.

**Returns:** `string`

### `CaptureNetwork(opts NetworkCaptureOptions, onExchange NetworkExchangeHandler) → *NetworkCapture`

*method on `CloudBrowser`*

CaptureNetwork starts capturing the session's network traffic and returns a live view of it.

Every request matching opts.Patterns is reported once it completes, and "every request" is literal: capture sits in the browser process rather than in a page, so cross-process iframes, workers and service workers are included, the headers are the ones actually put on the wire (Cookie and Sec-* included), and each hop of a redirect chain arrives as its own exchange. Requests are never paused, so the page loads at full speed.

This call returns as soon as the capture is running; onExchange then fires in the background while you drive the browser. The capture is armed only after the subscription exists, so nothing that happens after this call returns is missed. Call NetworkCapture.Stop when done — it disarms the capture server-side, which a cancelled context alone does not.

**Parameters:**
- `opts` (`NetworkCaptureOptions`) — which requests to capture and whether to keep bodies
- `onExchange` (`NetworkExchangeHandler`) — called per exchange; see NetworkExchangeHandler for the

**Returns:** `*NetworkCapture` — *NetworkCapture handle for stopping the capture and inspecting how

**Throws:**
- `UNKNOWN_ERROR` — onExchange is nil, or the capture could not be started

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

### `ClearCookies()`

*method on `CloudBrowser`*

ClearCookies deletes every cookie in the browser context.

**Throws:**
- `UNKNOWN_ERROR` — the cookies could not be cleared

```go
_ = browser.ClearCookies(ctx)
```

### `ClearStorage(origin string)`

*method on `CloudBrowser`*

ClearStorage deletes localStorage in the browser context.

**Parameters:**
- `origin` (`string`) — if non-empty, only this origin's storage is deleted (e.g. "https://example.com"); empty string deletes all origins

**Throws:**
- `UNKNOWN_ERROR` — the storage could not be cleared

```go
// Wipe one origin.
_ = browser.ClearStorage(ctx, "https://example.com")

// Wipe everything.
_ = browser.ClearStorage(ctx, "")
```

### `Click(target *Locator) → *ElementResult`

*method on `CloudBrowser`*

Click triggers a single left mouse click on the given target.

The browser scrolls the element into view if needed, moves the cursor along a human-like path, then dispatches a full mouseDown+mouseUp at a randomized point inside the element's bounding rect.

**Parameters:**
- `target` (`*Locator`) — locator describing what to click; At is also valid

**Returns:** `*ElementResult` — *ElementResult with success, resolved frameId, backendNodeId,

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
res, err := browser.Click(ctx, browserscale.CSS("button.submit"))
if err != nil {
    var ce *browserscale.ClickError
    if errors.As(err, &ce) {
        log.Printf("blocked by %s (%s)", ce.Occluder.TagName, ce.Code)
    }
    log.Fatal(err)
}
```

**See also:** ClickError for the occlusion-failure detail · CloudBrowser.ClickWith for right-click, double-click,

### `ClickWith(target *Locator, opts ClickOpts) → *ElementResult`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.Click`*

ClickWith is the customizable variant of CloudBrowser.Click.

The browser scrolls the element into view if needed, moves the cursor along a human-like path, then dispatches a full mouseDown+mouseUp at a randomized point inside the element's bounding rect.

**Parameters:**
- `target` (`*Locator`) — locator describing what to click; At is also valid
- `opts` (`ClickOpts`) — click customization; see ClickOpts

**Returns:** `*ElementResult` — *ElementResult with success, resolved frameId, backendNodeId,

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
// Right double-click on a context menu trigger.
_, err := browser.ClickWith(ctx, browserscale.CSS("li.menu"), browserscale.ClickOpts{
    Button:     "right",
    ClickCount: 2,
})
```

**See also:** ClickError for the occlusion-failure detail · CloudBrowser.ClickWith for right-click, double-click,

### `Close()`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.StopBrowser`*

Close is the defer-friendly alias for CloudBrowser.StopBrowser that uses a background context.

Useful when a session id was persisted across processes and the rental outlived the original handle. Only calls the rent stop endpoint; there is no gRPC connection to close in this form.

**Throws:**
- `UNKNOWN_ERROR` — the stop API rejected the request

```go
browser, err := browserscale.RentBrowser(ctx, cfg)
if err != nil { log.Fatal(err) }
defer browser.Close()
```

### `CloseConn()`

*method on `CloudBrowser`*

CloseConn closes only the gRPC connection, leaving the server-side session running.

Use this to detach without releasing the rental — the common case when you attached with ConnectSession to act on a session owned elsewhere, or when a short-lived handle should not outlive its work but the session must. Contrast with CloudBrowser.Close / CloudBrowser.StopBrowser, which also release the rental via the stop endpoint.

**Throws:**
- `UNKNOWN_ERROR` — the gRPC connection could not be closed

```go
browser, err := browserscale.ConnectSession(ctx, grpcUrl, apiKey, sessionId)
if err != nil { log.Fatal(err) }
defer browser.CloseConn() // detach; the session keeps running
```

### `CountryCode() → string`

*method on `CloudBrowser`*

CountryCode returns the ISO-3166 country code the server allocated for this session (drives geo-IP and locale defaults).

**Returns:** `string`

### `DragBy(target *Locator, offsetX float64, offsetY float64) → *DragResult`

*method on `CloudBrowser`*

DragBy picks up the target and drops it at an offset relative to the pickup point.

The browser presses the left mouse button at a pickup point inside the element, drags along a human-like path to (pickupX+offsetX, pickupY+offsetY), then releases. At is not a valid target — drag needs a real element.

**Parameters:**
- `target` (`*Locator`) — locator describing the element to pick up
- `offsetX` (`float64`) — horizontal distance to drag, in CSS pixels
- `offsetY` (`float64`) — vertical distance to drag, in CSS pixels

**Returns:** `*DragResult` — *DragResult with the resolved frameId, backendNodeId and the

**Throws:**
- `UNKNOWN_ERROR` — the drag could not be performed

```go
_, err := browser.DragBy(ctx, browserscale.CSS(".slider .handle"), 120, 0)
if err != nil {
    log.Fatal(err)
}
```

### `DragTo(target *Locator, absoluteX float64, absoluteY float64) → *DragResult`

*method on `CloudBrowser`*

DragTo picks up the target and drops it at absolute root-viewport coordinates.

Same gesture as CloudBrowser.DragBy, but the drop destination is in page coordinates rather than relative to the pickup point.

**Parameters:**
- `target` (`*Locator`) — locator describing the element to pick up
- `absoluteX` (`float64`) — horizontal drop coordinate in the root viewport
- `absoluteY` (`float64`) — vertical drop coordinate in the root viewport

**Returns:** `*DragResult` — *DragResult with the resolved frameId, backendNodeId and the

**Throws:**
- `UNKNOWN_ERROR` — the drag could not be performed

```go
_, err := browser.DragTo(ctx, browserscale.CSS(".card"), 800, 400)
if err != nil {
    log.Fatal(err)
}
```

### `Evaluate(expression string) → *EvaluateResult`

*method on `CloudBrowser`*

Evaluate runs a JavaScript expression in the page's main frame.

The expression's return value is JSON-serialized server-side and parsed eagerly into EvaluateResult.Value. When the expression returns a DOM element the EvaluateResult.Value is left empty and the element metadata (BackendNodeId, IsVisible, Bounds) is populated instead — use Node(id) in subsequent calls to act on it.

**Parameters:**
- `expression` (`string`) — JavaScript expression evaluated in the main frame

**Returns:** `*EvaluateResult` — *EvaluateResult with either Value (for non-Element returns) or

**Throws:**
- `UNKNOWN_ERROR` — the expression threw or could not be compiled

```go
res, err := browser.Evaluate(ctx, "document.title")
if err != nil {
    log.Fatal(err)
}
fmt.Println(res.Value)
```

### `EvaluateInFrame(frameId string, expression string) → *EvaluateResult`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.Evaluate`*

EvaluateInFrame runs a JavaScript expression in the given frame.

Same semantics as CloudBrowser.Evaluate but targets a specific frame instead of the main frame. Useful for evaluating inside OOPIFs (out-of- process iframes) found via CloudBrowser.GetPages.

**Parameters:**
- `frameId` (`string`) — id of the frame to evaluate in; empty falls back to the main frame
- `expression` (`string`) — JavaScript expression evaluated in the main frame

**Returns:** `*EvaluateResult` — *EvaluateResult with either Value (for non-Element returns) or

**Throws:**
- `UNKNOWN_ERROR` — the expression threw or could not be compiled

```go
pages, _ := browser.GetPages(ctx)
iframeId := pages[0].FrameTree.Children[0].FrameId
_, _ = browser.EvaluateInFrame(ctx, iframeId, "location.href")
```

### `Fill(target *Locator, text string) → *ElementResult`

*method on `CloudBrowser`*

Fill clicks the target and types text into it, appending to any existing content.

The browser scrolls the element into view, moves the cursor along a human-like path, clicks to focus, then types the text character-by- character with QWERTZ keyboard simulation and human-like timing.

To overwrite the field instead of appending, use CloudBrowser.FillWith with ClearFirst: true.

At is not a valid target — Fill requires an actual element.

**Parameters:**
- `target` (`*Locator`) — locator describing the input element
- `text` (`string`) — text to type into the element

**Returns:** `*ElementResult` — *ElementResult with success, resolved frameId, backendNodeId

**Throws:**
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
res, err := browser.Fill(ctx, browserscale.CSS("input[name=email]"), "user@example.com")
if err != nil {
    var fe *browserscale.FillError
    if errors.As(err, &fe) && fe.ClickError != nil {
        log.Printf("blocked by %s", fe.ClickError.Occluder.TagName)
    }
    log.Fatal(err)
}
_ = res
```

**See also:** CloudBrowser.FillWith for clearing existing content or · FillError for the focus-failure detail

### `FillWith(target *Locator, text string, opts FillOpts) → *ElementResult`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.Fill`*

FillWith is the customizable variant of CloudBrowser.Fill.

The browser scrolls the element into view, moves the cursor along a human-like path, clicks to focus, then types the text character-by- character with QWERTZ keyboard simulation and human-like timing.

To overwrite the field instead of appending, use CloudBrowser.FillWith with ClearFirst: true.

At is not a valid target — Fill requires an actual element.

**Parameters:**
- `target` (`*Locator`) — locator describing the input element
- `text` (`string`) — text to type into the element
- `opts` (`FillOpts`) — fill customization; see FillOpts

**Returns:** `*ElementResult` — *ElementResult with success, resolved frameId, backendNodeId

**Throws:**
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
// Wipe the field first, then type fresh content.
_, err := browser.FillWith(ctx, browserscale.CSS("input[name=email]"), "user@example.com", browserscale.FillOpts{
    ClearFirst: true,
})
```

**See also:** CloudBrowser.FillWith for clearing existing content or · FillError for the focus-failure detail

### `Fingerprint() → string`

*method on `CloudBrowser`*

Fingerprint returns the browser fingerprint id in use for this session.

**Returns:** `string`

### `FollowScript(runId string, onEvent ScriptEventHandler) → *ScriptFollow`

*method on `CloudBrowser`*

FollowScript watches script output in a session without starting anything.

For the case CloudBrowser.StartScript cannot cover: a run somebody else launched, or one this process started before it restarted. Several readers can watch the same session, each with its own buffer.

Only output produced from now on arrives — lines printed before the subscription existed are not kept. A run that has already finished is therefore invisible here; CloudBrowser.ListScriptRuns is how you tell that apart from a run that is merely quiet.

**Parameters:**
- `runId` (`string`) — run to follow, or "" to follow every run in the session
- `onEvent` (`ScriptEventHandler`) — called per event; see ScriptEventHandler for the ordering

**Returns:** `*ScriptFollow` — *ScriptFollow handle for stopping the subscription

**Throws:**
- `UNKNOWN_ERROR` — onEvent is nil, or the subscription could not be opened

```go
follow, err := browser.FollowScript(ctx, runId, func(ev browserscale.ScriptEvent) {
    if ev.Log != nil { fmt.Println(ev.Log.Message) }
})
if err != nil { log.Fatal(err) }
defer follow.Stop()
```

### `GetAuthSession() → *AuthSession`

*method on `CloudBrowser`*

GetAuthSession exports the signed-in primary account and DBSC sessions of this browser context. Reads state in the browser process — no page needed.

Returns nil, nil when the context has neither a signed-in account nor DBSC sessions.

**Returns:** `*AuthSession` — *AuthSession, or nil when there is nothing to export

**Throws:**
- `UNKNOWN_ERROR` — the auth session could not be read

```go
auth, err := browser.GetAuthSession(ctx)
if err != nil {
    log.Fatal(err)
}
if auth == nil {
    log.Println("no auth/DBSC state")
    return
}
// persist auth, then later SetAuthSession on a fresh rent
```

### `GetCookies() → []CookieParam`

*method on `CloudBrowser`*

GetCookies returns all cookies currently stored in this session's browser context.

**Returns:** `[]CookieParam` — []CookieParam, one per cookie in the context

**Throws:**
- `UNKNOWN_ERROR` — the cookies could not be read

```go
cookies, err := browser.GetCookies(ctx)
if err != nil {
    log.Fatal(err)
}
for _, c := range cookies {
    fmt.Println(c.Name, "=", c.Value)
}
```

### `GetDOM(frameId string, depth int32) → string`

*method on `CloudBrowser`*

GetDOM returns a JSON string in CDP DOM.Node shape for the requested frame.

The shape matches Chrome DevTools' Protocol DOM.Node — useful for piping into agent loops or visualizers that already speak CDP. For a much smaller agent-oriented payload, prefer CloudBrowser.GetObservation instead.

**Parameters:**
- `frameId` (`string`) — id of the frame to dump; empty targets the main frame
- `depth` (`int32`) — tree depth: -1 for the full tree, 0 for root only, N for

**Returns:** `string` — JSON string in CDP DOM.Node shape

**Throws:**
- `UNKNOWN_ERROR` — the DOM could not be retrieved

```go
tree, err := browser.GetDOM(ctx, "", -1)
if err != nil {
    log.Fatal(err)
}
fmt.Println(tree)
```

### `GetDomChildren(backendNodeId int32, frameId string, depth int32) → *DomChildren`

*method on `CloudBrowser`*

GetDomChildren fetches a node's children and starts reporting changes inside them. DomMirror.Expand calls this and folds the result into the tree.

On an <iframe>/<frame>/<object> the one child is the document it hosts, and this call is what starts mirroring that frame.

**Parameters:**
- `backendNodeId` (`int32`) — the node to open
- `frameId` (`string`) — the frame its id belongs to; empty targets the main frame
- `depth` (`int32`) — levels below the node; 0 uses the server default of 1

**Returns:** `*DomChildren` — *DomChildren with the child list as JSON and the sequence it is

**Throws:**
- `UNKNOWN_ERROR` — the children could not be read

### `GetDOMHash(frameId string) → string`

*method on `CloudBrowser`*

GetDOMHash returns sha256:8 of the full-tree DOM JSON for cheap polling-based change detection.

Computing a hash is much cheaper than transferring the full tree — pair this with CloudBrowser.GetDOM only when the hash differs from your last snapshot.

**Parameters:**
- `frameId` (`string`) — id of the frame to hash; empty targets the main frame

**Returns:** `string` — 16-char hex string (the first 8 bytes of sha256 of the DOM JSON)

**Throws:**
- `UNKNOWN_ERROR` — the hash could not be computed

```go
hash, err := browser.GetDOMHash(ctx, "")
if err != nil {
    log.Fatal(err)
}
if hash != lastHash {
    // DOM changed → re-fetch
}
```

### `GetDomRevision(frameId string) → uint64`

*method on `CloudBrowser`*

GetDomRevision returns a frame's mutation counter, incremented on every change the document sees. O(1) in the browser and the change detector to poll if you are not consuming mirror events.

Prefer it over CloudBrowser.GetDOMHash, which serializes the whole tree just to hash it. The two answer different questions: a hash compares content, a revision only says whether this document moved since you last asked. The counter is meaningful only within the current document.

**Parameters:**
- `frameId` (`string`) — the frame to ask; empty targets the main frame

**Returns:** `uint64` — uint64 monotonic counter

**Throws:**
- `UNKNOWN_ERROR` — the revision could not be read

### `GetObservation() → string`

*method on `CloudBrowser`*

GetObservation returns a compact, frame-aware view of the visible page — the first thing to reach for on an unfamiliar page, and the cheapest way to re-read the current state afterwards.

Each frame opens with header lines carrying the URL, the title and the scroll position, then one line per visible element:

input#email47 type="email" name="loginId" value="a@b.com" required click "E-Mail"

It spans every frame, pierces open and closed shadow roots, enumerates <select> options, and reports live form state: value= is what is typed in right now (passwords as a length), checked= for boxes. The trailing quoted string is always the label or text, never the value, so an empty and a prefilled field stay distinguishable. Because the headers already carry URL, title and scroll offset, this replaces the usual handful of CloudBrowser.Evaluate probes after each step.

On what to do with the result: backendNodeId (the 47 above) is a handle for this session and can be passed straight to click/fill via Node. It does not survive a new document, so for anything you write into a script, target with CSS or JS instead — those calls return the backendNodeId they resolved to, which lets you confirm the durable anchor hits the element you saw.

**Returns:** `string` — the observation in the requested format, ready to hand to a model

**Throws:**
- `UNKNOWN_ERROR` — the observation could not be produced

```go
obs, err := browser.GetObservation(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println(obs)
```

### `GetObservationWith(opts ObservationOpts) → string`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.GetObservation`*

GetObservationWith is the customizable variant of CloudBrowser.GetObservation.

Each frame opens with header lines carrying the URL, the title and the scroll position, then one line per visible element:

input#email47 type="email" name="loginId" value="a@b.com" required click "E-Mail"

It spans every frame, pierces open and closed shadow roots, enumerates <select> options, and reports live form state: value= is what is typed in right now (passwords as a length), checked= for boxes. The trailing quoted string is always the label or text, never the value, so an empty and a prefilled field stay distinguishable. Because the headers already carry URL, title and scroll offset, this replaces the usual handful of CloudBrowser.Evaluate probes after each step.

On what to do with the result: backendNodeId (the 47 above) is a handle for this session and can be passed straight to click/fill via Node. It does not survive a new document, so for anything you write into a script, target with CSS or JS instead — those calls return the backendNodeId they resolved to, which lets you confirm the durable anchor hits the element you saw.

**Parameters:**
- `opts` (`ObservationOpts`) — observation customization; see ObservationOpts

**Returns:** `string` — the observation in the requested format, ready to hand to a model

**Throws:**
- `UNKNOWN_ERROR` — the observation could not be produced

```go
// Only what is on screen right now, as structured JSON.
obs, err := browser.GetObservationWith(ctx, browserscale.ObservationOpts{
    Format:       "json",
    ViewportOnly: true,
})

// Re-read just one form after the first full look.
obs, err = browser.GetObservationWith(ctx, browserscale.ObservationOpts{
    Selector: "form#register",
})
```

### `GetPages() → []*PageInfo`

*method on `CloudBrowser`*

GetPages returns all open pages (tabs and popups) for this session's browser context.

Each PageInfo carries the page's URL, title, viewport and a full nested frame tree (out-of-process iframes are children of the page's main frame).

**Returns:** `[]*PageInfo` — []*PageInfo for every page currently open in the context

**Throws:**
- `UNKNOWN_ERROR` — the pages could not be enumerated

```go
pages, err := browser.GetPages(ctx)
if err != nil {
    log.Fatal(err)
}
for _, p := range pages {
    fmt.Println(p.Url, p.Title)
}
```

### `GetSelection() → string`

*method on `CloudBrowser`*

GetSelection returns the current text selection.

Walks every frame and returns the first non-empty selection found — useful for "copy what the user highlighted" flows. Returns an empty string when nothing is selected anywhere.

**Returns:** `string` — the selected text, or "" when nothing is selected

**Throws:**
- `UNKNOWN_ERROR` — the selection could not be read

```go
sel, err := browser.GetSelection(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println("user selected:", sel)
```

### `GetStorage(origin string) → []StorageOriginEntry`

*method on `CloudBrowser`*

GetStorage returns the localStorage contents of this session's browser context, grouped by origin.

The storage database is read directly in the browser process, so no page needs to be open. Only first-party localStorage is included — sessionStorage is per-tab and not covered.

**Parameters:**
- `origin` (`string`) — if non-empty, only this origin is returned (e.g. "https://example.com"); empty string returns all origins

**Returns:** `[]StorageOriginEntry` — []StorageOriginEntry, one per origin with localStorage data

**Throws:**
- `UNKNOWN_ERROR` — the storage could not be read

```go
storage, err := browser.GetStorage(ctx, "")
if err != nil {
    log.Fatal(err)
}
for _, e := range storage {
    for _, item := range e.Items {
        fmt.Println(e.Origin, item.Key, "=", item.Value)
    }
}
```

### `GetStreamConfig() → []IceServer`

*method on `CloudBrowser`*

GetStreamConfig returns the ICE servers (TURN URL + short-lived credentials) to put in your RTCPeerConnection BEFORE creating the offer, so it can gather relay candidates.

Live streaming is a two-step, client-offerer handshake: call GetStreamConfig, build your peer with the returned servers, create an offer, then pass its SDP to CloudBrowser.StartStream and apply the returned answer.

**Returns:** `[]IceServer` — the ICE servers for the client RTCPeerConnection

**Throws:**
- `UNKNOWN_ERROR` — TURN is not configured on the server

```go
ice, err := browser.GetStreamConfig(ctx)
if err != nil { log.Fatal(err) }
// configure your RTCPeerConnection with ice, then create an offer …
```

### `GrpcUrl() → string`

*method on `CloudBrowser`*

GrpcUrl returns the gRPC endpoint the session is connected to.

**Returns:** `string`

### `HighlightNode(backendNodeId int32, frameId string)`

*method on `CloudBrowser`*

HighlightNode paints a debug overlay over the node identified by backendNodeId.

Useful for visual debugging of agent flows — the overlay stays until the next call. Pass backendNodeId <= 0 to clear any current highlights.

**Parameters:**
- `backendNodeId` (`int32`) — id of the node to highlight, or <= 0 to clear
- `frameId` (`string`) — id of the frame the node lives in; empty targets the main frame

**Throws:**
- `UNKNOWN_ERROR` — the highlight could not be applied

```go
if err := browser.HighlightNode(ctx, res.BackendNodeId, res.FrameId); err != nil {
    log.Fatal(err)
}
```

### `InsertText(text string)`

*method on `CloudBrowser`*

InsertText pastes text at the current caret using IME-style input.

No individual key events are dispatched; the entire string is committed at once via Input.insertText. Whatever element currently has focus receives the text. Use CloudBrowser.Click or CloudBrowser.Fill first if you need a specific element to be focused.

**Parameters:**
- `text` (`string`) — the text to insert at the caret

**Throws:**
- `UNKNOWN_ERROR` — the text could not be inserted

```go
if err := browser.InsertText(ctx, "hello world"); err != nil {
    log.Fatal(err)
}
```

### `InspectAtPosition(x float64, y float64) → *InspectResult`

*method on `CloudBrowser`*

InspectAtPosition hit-tests at the viewport-relative (x, y) and returns the topmost element under that point.

Mirrors what the live-UI overlay does on hover. Elements with pointer-events:none are skipped — the result is the actual click target, not the visually-topmost node.

**Parameters:**
- `x` (`float64`) — viewport-relative x in CSS pixels
- `y` (`float64`) — viewport-relative y in CSS pixels

**Returns:** `*InspectResult` — *InspectResult with the resolved backendNodeId, frameId, tag

**Throws:**
- `UNKNOWN_ERROR` — the hit-test failed

```go
res, err := browser.InspectAtPosition(ctx, 200, 300)
if err != nil {
    log.Fatal(err)
}
fmt.Println(res.TagName, res.TextContent)
```

### `ListReactions() → []ReactionInfo`

*method on `CloudBrowser`*

ListReactions returns the still-pending reactions registered for the current page. Reactions that have already fired (one-shot) are not included.

**Returns:** `[]ReactionInfo` — the pending reactions for the page

```go
pending, err := browser.ListReactions(ctx)
for _, r := range pending {
    log.Printf("reaction %s watching %s%s", r.ReactionID, r.MatchSelector, r.MatchJsExpression)
}
```

### `ListScriptRuns() → []ScriptRunInfo`

*method on `CloudBrowser`*

ListScriptRuns reports the scripts still running in the session.

Only runs in flight — a finished run is reported once on the event stream and then forgotten, so this is not a history. Its use is finding work this caller did not start: a script a previous process left behind, which CloudBrowser.StopScripts needs an id to name.

**Returns:** `[]ScriptRunInfo` — []ScriptRunInfo one entry per run still executing

**Throws:**
- `UNKNOWN_ERROR` — the session could not be queried

```go
runs, err := browser.ListScriptRuns(ctx)
if err != nil { log.Fatal(err) }
for _, run := range runs {
    fmt.Println(run.RunId, run.Running)
}
```

### `LoadHTML(url string, html string, headers []Header, statusCode int32)`

*method on `CloudBrowser`*

LoadHTML serves a synthetic response for the next navigation to url.

Registers a one-shot interceptor that intercepts the next request to url and replies with the supplied html and headers instead of going to the network. Useful for snapshotted pages, test fixtures, and offline replays. Pair with CloudBrowser.Navigate to trigger the load.

**Parameters:**
- `url` (`string`) — the URL pattern that, when navigated to, returns the html
- `html` (`string`) — the response body to serve
- `headers` (`[]Header`) — extra response headers (Content-Type is set automatically)
- `statusCode` (`int32`) — HTTP status code to serve; 0 means 200

**Throws:**
- `UNKNOWN_ERROR` — the interceptor could not be installed

```go
_ = browser.LoadHTML(ctx, "https://example.com", "<h1>hi</h1>", nil, 0)
_, _ = browser.Navigate(ctx, "https://example.com", 0)
```

### `MirrorDom(opts DomMirrorOptions, onChange DomChangeHandler, onResync DomResyncHandler) → *DomMirror`

*method on `CloudBrowser`*

MirrorDom starts mirroring the session's page and returns a live copy of its DOM.

One mirror covers the whole page as ONE tree. An <iframe> is an ordinary element whose single child is the document it hosts; expanding it fetches that document and starts mirroring the frame, however deeply nested and whether or not it is cross-origin. Unlike the inlining CloudBrowser.GetDOM does, these regions stay live — and frames nobody opened cost nothing.

This replaces polling CloudBrowser.GetDOMHash and re-fetching GetDOM: the browser reports changes to the part you actually expanded instead of re-serializing the document so you can hash it.

The subscription is established before the snapshot is taken, so no change between the two is lost. Call DomMirror.Stop when done — it stops the mirror server-side, which a cancelled context alone does not.

**Parameters:**
- `opts` (`DomMirrorOptions`) — initial depth and whether to pierce shadow roots
- `onChange` (`DomChangeHandler`) — called after every change, including the first snapshot;
- `onResync` (`DomResyncHandler`) — called when the copy had to be rebuilt; may be nil

**Returns:** `*DomMirror` — *DomMirror holding the tree

**Throws:**
- `UNKNOWN_ERROR` — onChange is nil, or the mirror could not be started

```go
mirror, err := browser.MirrorDom(ctx, browserscale.DomMirrorOptions{Pierce: true},
    func(m *browserscale.DomMirror) {
        render(m.Root())
    }, nil)
if err != nil {
    log.Fatal(err)
}
defer mirror.Stop(ctx)

body := mirror.Node(mirror.MainFrameId(), bodyId)
_ = mirror.Expand(ctx, body, 0)
```

### `ModifyRequest(urlPattern string, body string, timeoutMs float64, mods []HeaderModification) → *InterceptedRequest`

*method on `CloudBrowser`*

ModifyRequest waits for the next request whose URL matches urlPattern, applies the supplied header modifications (and optional body replacement), then forwards the modified request.

One-shot: consumes the first matching request. Pass nil/empty mods to leave headers untouched and only override the body.

**Parameters:**
- `urlPattern` (`string`) — URL wildcard to wait for
- `body` (`string`) — replacement request body; empty leaves the original body
- `timeoutMs` (`float64`) — per-call timeout in milliseconds; 0 uses the server default
- `mods` (`[]HeaderModification`) — HeaderModification entries; see HeaderModification for the fields

**Returns:** `*InterceptedRequest` — *InterceptedRequest carrying the method/URL/headers/body that

**Throws:**
- `UNKNOWN_ERROR` — no matching request appeared within the timeout

```go
req, err := browser.ModifyRequest(ctx, "*/api/me", "", 5000, []browserscale.HeaderModification{
    {Action: browserscale.HeaderModificationAdd, Name: "X-Trace", Value: "abc123"},
    {Action: browserscale.HeaderModificationRemove, Name: "Cookie"},
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("forwarded headers:", req.Headers)
```

### `MoveTo(target *Locator) → *ElementResult`

*method on `CloudBrowser`*

MoveTo moves the mouse cursor over the given target.

The browser scrolls the target into view first if necessary, then animates the cursor along a human-like path to the element's random center area (or to the viewport coordinate when target is At).

**Parameters:**
- `target` (`*Locator`) — locator describing where to move; At is also valid

**Returns:** `*ElementResult` — *ElementResult with the resolved frameId, backendNodeId,

**Throws:**
- `UNKNOWN_ERROR` — the move could not be completed

```go
_, err := browser.MoveTo(ctx, browserscale.CSS("nav .menu"))
if err != nil {
    log.Fatal(err)
}
```

### `Navigate(url string, timeoutMs float64) → *NavigateResult`

*method on `CloudBrowser`*

Navigate navigates the page to url.

Returns once the primary main-frame navigation commits (the response is received and a new document is selected), before DOMContentLoaded or load fire. Cross-origin redirects are followed.

**Parameters:**
- `url` (`string`) — destination URL
- `timeoutMs` (`float64`) — per-call timeout in milliseconds; 0 uses the server default

**Returns:** `*NavigateResult` — *NavigateResult with the final resolved URL and the frameId of

**Throws:**
- `UNKNOWN_ERROR` — the navigation failed or timed out

```go
_, err := browser.Navigate(ctx, "https://example.com", 0)
if err != nil {
    log.Fatal(err)
}
```

### `PressKey(key string, code string, modifiers int32, location int32)`

*method on `CloudBrowser`*

PressKey fires a single key-down event.

Only the keydown half is dispatched — pair with CloudBrowser.ReleaseKey for a full press cycle. The event targets whichever element currently has focus.

**Parameters:**
- `key` (`string`) — DOM KeyboardEvent.key value (e.g. "Enter", "a", "ArrowLeft")
- `code` (`string`) — DOM KeyboardEvent.code value (e.g. "Enter", "KeyA"); empty falls back to key
- `modifiers` (`int32`) — bit-flag combination: Alt=1, Ctrl=2, Meta=4, Shift=8
- `location` (`int32`) — DOM KeyboardEvent.location: 0=standard, 1=left, 2=right, 3=numpad

**Throws:**
- `UNKNOWN_ERROR` — the event could not be dispatched

```go
// Ctrl+A
_ = browser.PressKey(ctx, "a", "KeyA", 2, 0)
_ = browser.ReleaseKey(ctx, "a", "KeyA", 2, 0)
```

### `ReadCanvas(target *Locator) → *ReadCanvasResult`

*method on `CloudBrowser`*

ReadCanvas reads the pixels of a <canvas> element directly in the renderer, bypassing the origin-clean (tainted) security check and without executing any page JavaScript — so cross-origin/tainted canvases (common in captchas) read fine where a normal toDataURL / getImageData would throw a SecurityError.

**Parameters:**
- `target` (`*Locator`) — locator for the <canvas>; CSS, JS or Node

**Returns:** `*ReadCanvasResult` — *ReadCanvasResult with the base64 image in DataBase64, the canvas

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty, uses At(x,y), or has multiple targets
- `PAGE_NOT_ALIVE` — the page has been closed
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
res, err := browser.ReadCanvas(ctx, browserscale.CSS("#game canvas"))
if err != nil {
    log.Fatal(err)
}
img, _ := base64.StdEncoding.DecodeString(res.DataBase64)
os.WriteFile("canvas.png", img, 0o644)
```

**See also:** CloudBrowser.ReadCanvasWith for format, quality, or a sub-rectangle

### `ReadCanvasWith(target *Locator, opts ReadCanvasOpts) → *ReadCanvasResult`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.ReadCanvas`*

ReadCanvasWith is the customizable variant of CloudBrowser.ReadCanvas.

**Parameters:**
- `target` (`*Locator`) — locator for the <canvas>; CSS, JS or Node
- `opts` (`ReadCanvasOpts`) — format, quality, sub-rectangle and frame override; see ReadCanvasOpts

**Returns:** `*ReadCanvasResult` — *ReadCanvasResult with the base64 image in DataBase64, the canvas

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty, uses At(x,y), or has multiple targets
- `PAGE_NOT_ALIVE` — the page has been closed
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
// Read the left half of the canvas as JPEG at quality 80.
res, err := browser.ReadCanvasWith(ctx, browserscale.CSS("canvas"),
    browserscale.ReadCanvasOpts{Format: "jpeg", Quality: 80, SW: 150, SH: 300})
```

**See also:** CloudBrowser.ReadCanvasWith for format, quality, or a sub-rectangle

### `ReleaseDomSubtree(backendNodeId int32, frameId string)`

*method on `CloudBrowser`*

ReleaseDomSubtree stops reporting changes inside a node, and inside any frame below it. DomMirror.Collapse calls this.

Skipping it is not an error, it is a slow leak: the browser's revealed set only grows, and eventually it is no longer filtering anything.

**Parameters:**
- `backendNodeId` (`int32`) — the node to close
- `frameId` (`string`) — the frame its id belongs to; empty targets the main frame

**Throws:**
- `UNKNOWN_ERROR` — the subtree could not be released

### `ReleaseKey(key string, code string, modifiers int32, location int32)`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.PressKey`*

ReleaseKey fires a single key-up event.

Mirror of CloudBrowser.PressKey. Same parameter semantics; use this to close a press cycle that was started with PressKey.

**Parameters:**
- `key` (`string`) — DOM KeyboardEvent.key value (e.g. "Enter", "a", "ArrowLeft")
- `code` (`string`) — DOM KeyboardEvent.code value (e.g. "Enter", "KeyA"); empty falls back to key
- `modifiers` (`int32`) — bit-flag combination: Alt=1, Ctrl=2, Meta=4, Shift=8
- `location` (`int32`) — DOM KeyboardEvent.location: 0=standard, 1=left, 2=right, 3=numpad

**Throws:**
- `UNKNOWN_ERROR` — the event could not be dispatched

```go
_ = browser.PressKey(ctx, "Shift", "ShiftLeft", 0, 1)
_ = browser.ReleaseKey(ctx, "Shift", "ShiftLeft", 0, 1)
```

### `RemoveReaction(reactionID string) → bool`

*method on `CloudBrowser`*

RemoveReaction removes a pending reaction by id. It returns false if the reaction had already fired (one-shot) or was never registered.

**Parameters:**
- `reactionID` (`string`) — id returned by CloudBrowser.AddReaction

**Returns:** `bool` — true if a pending reaction with this id existed and was removed

```go
removed, err := browser.RemoveReaction(ctx, id)
```

### `RevealDomNode(backendNodeId int32, frameId string) → *DomPath`

*method on `CloudBrowser`*

RevealDomNode returns the chain from the main document down to a node, each ancestor with its own children, crossing into frames where it has to and starting the ones it passes through. DomMirror.Reveal calls this and splices it in.

**Parameters:**
- `backendNodeId` (`int32`) — the node to reach
- `frameId` (`string`) — the frame its id belongs to; empty targets the main frame

**Returns:** `*DomPath` — *DomPath with the ancestor chain as JSON and the sequence it is

**Throws:**
- `UNKNOWN_ERROR` — the path could not be built

### `RunScript(source string) → *ScriptResult`

*method on `CloudBrowser`*

RunScript runs source in the session's browser and waits for it to finish.

The script runs beside the browser, in a V8 isolate of its own rather than in the page, and reaches the document through the engine: a cross-origin iframe is read as plain `contentDocument` with no frame ids anywhere, values come back as live objects it can assign to rather than snapshots, an element can be handed straight to `browser.click`, and the page sees nothing injected. Steps cost microseconds rather than network round trips, so work that is chatty by nature — polling for a selector, walking a list, following pagination — is affordable there. A guide for it is still to come.

This blocks for as long as the script runs, and cannot be bounded: the run id needed to cancel only arrives with the reply. Cancelling ctx abandons the wait but not the run. Use CloudBrowser.StartScript when the script may outlive the caller's patience, or CloudBrowser.StopScripts to abandon what this session is running.

**Parameters:**
- `source` (`string`) — JavaScript to execute; its return value comes back as JSON

**Returns:** `*ScriptResult` — *ScriptResult with the return value and the script's whole console

**Throws:**
- `UNKNOWN_ERROR` — the script could not be delivered to the browser

```go
result, err := browser.RunScript(ctx, `
    await browser.navigate("https://example.com");
    const items = [];
    for (const el of await browser.getDOM().querySelectorAll("h1")) {
        items.push(el.textContent);
    }
    return items;
`)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Success, result.Result)
```

### `Screenshot(format string, quality int32) → *ScreenshotResult`

*method on `CloudBrowser`*

Screenshot captures a single image of the page's current frame and returns it as base64-encoded image bytes.

The capture uses a one-shot surface copy (the same mechanism as CDP Page.captureScreenshot), so it is independent of any active live stream and works with both GPU (hardware) and software compositing.

**Parameters:**
- `format` (`string`) — "png" (default), "jpeg", or "webp"; pass "" for PNG
- `quality` (`int32`) — encode quality 0-100 for "jpeg"/"webp" (ignored for

**Returns:** `*ScreenshotResult` — *ScreenshotResult with the base64 image in DataBase64 and the

**Throws:**
- `UNKNOWN_ERROR` — the screenshot could not be captured

```go
shot, err := browser.Screenshot(ctx, "png", 0)
if err != nil {
    log.Fatal(err)
}
img, _ := base64.StdEncoding.DecodeString(shot.DataBase64)
os.WriteFile("page.png", img, 0o644)
```

### `ScrollTo(target *Locator) → *ElementResult`

*method on `CloudBrowser`*

ScrollTo scrolls the given element into view.

Whatever scroll container is closest to the element does the scrolling — nested scroll containers and out-of-process iframe chains are walked automatically. At is not a valid target here; scrolling needs a real element.

**Parameters:**
- `target` (`*Locator`) — locator describing the element to bring into view;

**Returns:** `*ElementResult` — *ElementResult with the resolved frameId, backendNodeId,

**Throws:**
- `UNKNOWN_ERROR` — the element could not be scrolled into view

```go
_, err := browser.ScrollTo(ctx, browserscale.CSS("#footer"))
if err != nil {
    log.Fatal(err)
}
```

### `SelectByIndex(target *Locator, index int32) → *SelectOptionResult`

*method on `CloudBrowser`*

SelectByIndex picks the <option> at the zero-based index inside the targeted <select> element.

Sets the option as selected on the targeted <select>, then fires the standard input + change events (unless suppressed via CloudBrowser.SelectByIndexWith with SelectOpts.NoEvents).

At is not a valid target — Select requires an actual <select> element.

**Parameters:**
- `target` (`*Locator`) — locator describing the <select> element
- `index` (`int32`) — zero-based option index

**Returns:** `*SelectOptionResult` — *SelectOptionResult with the resolved selectedIndex,

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `SELECT_FAILED` — the option could not be selected
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
_, err := browser.SelectByIndex(ctx, browserscale.CSS("select#country"), 2)
```

**See also:** CloudBrowser.SelectByIndexWith for suppressing events or · CloudBrowser.SelectByValue, CloudBrowser.SelectByText

### `SelectByIndexWith(target *Locator, index int32, opts SelectOpts) → *SelectOptionResult`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.SelectByIndex`*

SelectByIndexWith is the customizable variant of CloudBrowser.SelectByIndex.

Sets the option as selected on the targeted <select>, then fires the standard input + change events (unless suppressed via CloudBrowser.SelectByIndexWith with SelectOpts.NoEvents).

At is not a valid target — Select requires an actual <select> element.

**Parameters:**
- `target` (`*Locator`) — locator describing the <select> element
- `index` (`int32`) — zero-based option index
- `opts` (`SelectOpts`) — select customization; see SelectOpts

**Returns:** `*SelectOptionResult` — *SelectOptionResult with the resolved selectedIndex,

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `SELECT_FAILED` — the option could not be selected
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
// Pick the option silently, no input/change events.
_, err := browser.SelectByIndexWith(ctx, browserscale.CSS("select#hidden"), 0, browserscale.SelectOpts{
    NoEvents: true,
})
```

**See also:** CloudBrowser.SelectByIndexWith for suppressing events or · CloudBrowser.SelectByValue, CloudBrowser.SelectByText

### `SelectByText(target *Locator, text string) → *SelectOptionResult`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.SelectByIndex`*

SelectByText picks the <option> whose visible (trimmed) text matches the given string exactly.

Sets the option as selected on the targeted <select>, then fires the standard input + change events (unless suppressed via CloudBrowser.SelectByIndexWith with SelectOpts.NoEvents).

At is not a valid target — Select requires an actual <select> element.

**Parameters:**
- `target` (`*Locator`) — locator describing the <select> element
- `text` (`string`) — the visible option text to match

**Returns:** `*SelectOptionResult` — *SelectOptionResult with the resolved selectedIndex,

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `SELECT_FAILED` — the option could not be selected
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
_, err := browser.SelectByText(ctx, browserscale.CSS("select#country"), "Germany")
```

**See also:** CloudBrowser.SelectByIndexWith for suppressing events or · CloudBrowser.SelectByValue, CloudBrowser.SelectByText

### `SelectByTextWith(target *Locator, text string, opts SelectOpts) → *SelectOptionResult`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.SelectByText`*

SelectByTextWith is the customizable variant of CloudBrowser.SelectByText.

Sets the option as selected on the targeted <select>, then fires the standard input + change events (unless suppressed via CloudBrowser.SelectByIndexWith with SelectOpts.NoEvents).

At is not a valid target — Select requires an actual <select> element.

**Parameters:**
- `target` (`*Locator`) — locator describing the <select> element
- `text` (`string`) — the visible option text to match
- `opts` (`SelectOpts`) — select customization; see SelectOpts

**Returns:** `*SelectOptionResult` — *SelectOptionResult with the resolved selectedIndex,

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `SELECT_FAILED` — the option could not be selected
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
_, err := browser.SelectByTextWith(ctx, browserscale.CSS("select#country"), "Germany", browserscale.SelectOpts{
    NoEvents: true,
})
```

**See also:** CloudBrowser.SelectByIndexWith for suppressing events or · CloudBrowser.SelectByValue, CloudBrowser.SelectByText

### `SelectByValue(target *Locator, value string) → *SelectOptionResult`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.SelectByIndex`*

SelectByValue picks the <option> whose `value` attribute matches the given string exactly.

Sets the option as selected on the targeted <select>, then fires the standard input + change events (unless suppressed via CloudBrowser.SelectByIndexWith with SelectOpts.NoEvents).

At is not a valid target — Select requires an actual <select> element.

**Parameters:**
- `target` (`*Locator`) — locator describing the <select> element
- `value` (`string`) — the `value` attribute to match

**Returns:** `*SelectOptionResult` — *SelectOptionResult with the resolved selectedIndex,

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `SELECT_FAILED` — the option could not be selected
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
_, err := browser.SelectByValue(ctx, browserscale.CSS("select#country"), "DE")
```

**See also:** CloudBrowser.SelectByIndexWith for suppressing events or · CloudBrowser.SelectByValue, CloudBrowser.SelectByText

### `SelectByValueWith(target *Locator, value string, opts SelectOpts) → *SelectOptionResult`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.SelectByValue`*

SelectByValueWith is the customizable variant of CloudBrowser.SelectByValue.

Sets the option as selected on the targeted <select>, then fires the standard input + change events (unless suppressed via CloudBrowser.SelectByIndexWith with SelectOpts.NoEvents).

At is not a valid target — Select requires an actual <select> element.

**Parameters:**
- `target` (`*Locator`) — locator describing the <select> element
- `value` (`string`) — the `value` attribute to match
- `opts` (`SelectOpts`) — select customization; see SelectOpts

**Returns:** `*SelectOptionResult` — *SelectOptionResult with the resolved selectedIndex,

**Throws:**
- `ELEMENT_NOT_FOUND` — no element matched the locator
- `FRAME_NOT_FOUND` — the requested frame does not exist
- `INVALID_LOCATOR` — target is empty or has multiple targets set
- `PAGE_NOT_ALIVE` — the page has been closed
- `SELECT_FAILED` — the option could not be selected
- `TIMEOUT` — the operation exceeded the server-side timeout

```go
_, err := browser.SelectByValueWith(ctx, browserscale.CSS("select#country"), "DE", browserscale.SelectOpts{
    NoEvents: true,
})
```

**See also:** CloudBrowser.SelectByIndexWith for suppressing events or · CloudBrowser.SelectByValue, CloudBrowser.SelectByText

### `SessionId() → string`

*method on `CloudBrowser`*

SessionId returns the unique server-assigned id for this browser session.

**Returns:** `string`

### `SetAuthSession(session AuthSession)`

*method on `CloudBrowser`*

SetAuthSession imports an auth session so the context comes up signed in (and syncing if SyncConsent) with its DBSC sessions restored.

Call before navigating. Pair with SetCookies / SetStorage to fully restore a persona.

**Parameters:**
- `session` (`AuthSession`) — session as returned by GetAuthSession

**Throws:**
- `UNKNOWN_ERROR` — the auth session could not be written

```go
_ = browser.SetAuthSession(ctx, *saved)
_ = browser.Navigate(ctx, "https://mail.google.com")
```

### `SetBlockList(patterns []string)`

*method on `CloudBrowser`*

SetBlockList replaces the session's URL blocklist.

Any request whose URL matches one of the supplied patterns is blocked before it leaves the browser. Patterns are simple URL wildcards (`*` matches any character span). Pass a nil/empty slice to clear the blocklist and let everything through.

**Parameters:**
- `patterns` (`[]string`) — URL wildcards to block; nil or empty clears the list

**Throws:**
- `UNKNOWN_ERROR` — the blocklist could not be applied

```go
_ = browser.SetBlockList(ctx, []string{
    "*.doubleclick.net/*",
    "*googletagmanager.com*",
})
```

### `SetCookies(cookies []CookieParam)`

*method on `CloudBrowser`*

SetCookies writes the supplied cookies into the browser context.

Existing cookies with the same (name, domain, path) tuple are overwritten. Pass an empty slice for a no-op.

**Parameters:**
- `cookies` (`[]CookieParam`) — cookies to write; empty slice is a no-op

**Throws:**
- `UNKNOWN_ERROR` — the cookies could not be written

```go
secure := true
httpOnly := true
sameSite := "Lax"

_ = browser.SetCookies(ctx, []browserscale.CookieParam{
    {
        Name:     "auth",
        Value:    "tok",
        Domain:   "example.com",
        Path:     "/",
        Secure:   &secure,
        HTTPOnly: &httpOnly,
        SameSite: &sameSite,
    },
})
```

### `SetProxy(proxyHost string, proxyPort int32, proxyUsername string, proxyPassword string)`

*method on `CloudBrowser`*

SetProxy changes the runtime proxy for this session.

Takes effect for new requests immediately; in-flight requests keep their original routing. Pass an empty proxyHost to clear the proxy and route directly.

**Parameters:**
- `proxyHost` (`string`) — upstream proxy host; empty disables the proxy
- `proxyPort` (`int32`) — upstream proxy port; ignored when proxyHost is empty
- `proxyUsername` (`string`) — proxy auth user (empty for unauthenticated proxies)
- `proxyPassword` (`string`) — proxy auth password (empty for unauthenticated proxies)

**Throws:**
- `UNKNOWN_ERROR` — the proxy could not be applied

```go
_ = browser.SetProxy(ctx, "proxy.example.com", 8080, "user", "pass")
```

### `SetStaticPaths(blobName string, patterns []string)`

*method on `CloudBrowser`*

SetStaticPaths configures the session to serve cached static responses for requests matching the given patterns from blobName.

Useful for replaying frozen page assets (HTML/JS/CSS/images) without hitting the origin every time. The cache backend itself (blob storage, CDN, …) is configured server-side. Pass an empty patterns slice to disable caching for this session.

**Parameters:**
- `blobName` (`string`) — server-side identifier of the snapshot to serve from
- `patterns` (`[]string`) — URL wildcards to redirect to the cache; nil/empty disables

**Throws:**
- `UNKNOWN_ERROR` — the static paths could not be configured

```go
_ = browser.SetStaticPaths(ctx, "snap-2026-05", []string{"*.example.com/*"})
```

### `SetStorage(storage []StorageOriginEntry)`

*method on `CloudBrowser`*

SetStorage writes localStorage entries into the browser context, grouped by origin.

Accepts the same structure GetStorage returns, so a dump can be fed back verbatim. Existing keys are overwritten. Works without any open page; pages that are already open will not observe the writes until they reload.

**Parameters:**
- `storage` (`[]StorageOriginEntry`) — entries to write, grouped by origin

**Throws:**
- `UNKNOWN_ERROR` — the storage could not be written

```go
_ = browser.SetStorage(ctx, []browserscale.StorageOriginEntry{
    {
        Origin: "https://example.com",
        Items: []browserscale.StorageItem{
            {Key: "token", Value: "abc123"},
            {Key: "theme", Value: "dark"},
        },
    },
})
```

### `SolveCaptcha(timeoutMs int32, retryAmount int32) → string`

*method on `CloudBrowser`*

SolveCaptcha detects and solves the first supported bot-challenge it finds anywhere on the page.

Detection covers the common challenge types you run into in the wild. The challenge is completed in-page server-side (the resulting token / bypass cookies are wired into the page automatically), so callers can ignore the returned string.

**Parameters:**
- `timeoutMs` (`int32`) — how long to wait for a captcha to appear, in
- `retryAmount` (`int32`) — number of retries on a failed solve before giving up

**Returns:** `string` — empty string on success — the solution is applied server-side

**Throws:**
- `UNKNOWN_ERROR` — no captcha appeared within timeoutMs, or the

```go
if _, err := browser.SolveCaptcha(ctx, 0, 2); err != nil {
    log.Fatal(err)
}
```

### `StartDomMirror(opts DomMirrorOptions) → *DomSnapshot`

*method on `CloudBrowser`*

StartDomMirror starts (or restarts) the page's mirror and returns the main document, without subscribing to changes. CloudBrowser.MirrorDom is what you normally want; this is the raw command.

Calling it again restarts the mirror, which is also the recovery path after a resync. Subscribe before calling it: changes between the snapshot and the subscription are not replayed.

**Parameters:**
- `opts` (`DomMirrorOptions`) — initial depth and whether to pierce shadow roots

**Returns:** `*DomSnapshot` — *DomSnapshot with the main document, its frame and the baseline

**Throws:**
- `UNKNOWN_ERROR` — the mirror could not be started

### `StartNetworkCapture(opts NetworkCaptureOptions)`

*method on `CloudBrowser`*

StartNetworkCapture arms a capture without subscribing to it.

Use it when the reader lives somewhere else — another process, or a later ConnectSession against the same session. Most callers want CloudBrowser.CaptureNetwork instead, which arms and subscribes together. Calling this again replaces the running capture.

**Parameters:**
- `opts` (`NetworkCaptureOptions`) — which requests to capture and whether to keep bodies

**Throws:**
- `UNKNOWN_ERROR` — the capture could not be started

### `StartScript(source string, onEvent ScriptEventHandler) → *ScriptRun`

*method on `CloudBrowser`*

StartScript launches source in the session's browser and returns as soon as the run is under way.

The counterpart to CloudBrowser.RunScript, for scripts that are not worth waiting on: a watcher that runs for the life of the session, work that should survive this process. Output arrives at onEvent while the caller gets on with something else, and ScriptRun.Wait collects the outcome if it is wanted.

Subscribing has to happen before the launch, because a detached run's output is not kept anywhere — the browser rejects a start with nobody listening rather than discard the script's log and result. This call does both in that order, so nothing the script prints is missed.

**Parameters:**
- `source` (`string`) — JavaScript to execute
- `onEvent` (`ScriptEventHandler`) — called per log line and once for the outcome; see

**Returns:** `*ScriptRun` — *ScriptRun handle for awaiting or cancelling the run

**Throws:**
- `UNKNOWN_ERROR` — onEvent is nil, or the run could not be started

```go
run, err := browser.StartScript(ctx, source, func(ev browserscale.ScriptEvent) {
    if ev.Log != nil {
        fmt.Println(ev.Log.Level, ev.Log.Message)
    }
})
if err != nil {
    log.Fatal(err)
}
outcome, err := run.Wait(ctx)
```

### `StartStream(offerSDP string) → StreamAnswer`

*method on `CloudBrowser`*

StartStream answers your WebRTC SDP offer and starts streaming the page as a video track. The browser is the answerer; you are the offerer (see CloudBrowser.GetStreamConfig for the credentials to build the offer).

**Parameters:**
- `offerSDP` (`string`) — your RTCPeerConnection's SDP offer

**Returns:** `StreamAnswer` — the SDP answer plus the viewport to map input coordinates into

**Throws:**
- `UNKNOWN_ERROR` — the offer was empty, TURN is unconfigured, or the

```go
stream, err := browser.StartStream(ctx, offer.SDP)
if err != nil { log.Fatal(err) }
// peer.SetRemoteDescription({type: "answer", sdp: stream.AnswerSDP}) …
```

### `StopDomMirror()`

*method on `CloudBrowser`*

StopDomMirror stops the page's mirror, every frame of it. Idempotent.

**Throws:**
- `UNKNOWN_ERROR` — the mirror could not be stopped

### `StopNetworkCapture() → bool`

*method on `CloudBrowser`*

StopNetworkCapture disarms the session's capture.

**Returns:** `bool` — bool reporting whether a capture was running, and an error

**Throws:**
- `UNKNOWN_ERROR` — the capture could not be stopped

### `StopScripts(runId string) → int`

*method on `CloudBrowser`*

StopScripts cancels runs in the session and reports how many it ended.

An empty runId cancels every run in the session, which is the only form available to a caller that never learned an id — notably one abandoning a blocking CloudBrowser.RunScript.

**Parameters:**
- `runId` (`string`) — run to cancel, or "" for all of them

**Returns:** `int` — int how many runs were cancelled; 0 when the id named nothing in

**Throws:**
- `UNKNOWN_ERROR` — the cancel could not be delivered

```go
_, err := browser.StopScripts(ctx, "") // abandon everything running
```

### `StopStream()`

*method on `CloudBrowser`*

StopStream tears down the live video stream for the session's page. It is safe to call even if no stream is running.

**Throws:**
- `UNKNOWN_ERROR` — the stream could not be stopped

```go
if err := browser.StopStream(ctx); err != nil { log.Fatal(err) }
```

### `StreamNetworkExchanges(onExchange NetworkExchangeHandler) → *NetworkCapture`

*method on `CloudBrowser`*

StreamNetworkExchanges subscribes to the session's capture without arming one, for reading a capture that CloudBrowser.StartNetworkCapture armed elsewhere. Several readers can watch the same capture, each with its own buffer.

Stopping the returned view detaches this reader and leaves the capture running, since other readers may still be attached.

**Parameters:**
- `onExchange` (`NetworkExchangeHandler`) — called per exchange; see NetworkExchangeHandler for the

**Returns:** `*NetworkCapture` — *NetworkCapture attached to whatever capture is running; onExchange

**Throws:**
- `UNKNOWN_ERROR` — onExchange is nil, or the subscription could not be opened

### `Timezone() → string`

*method on `CloudBrowser`*

Timezone returns the IANA timezone the session was provisioned with (e.g. "Europe/Berlin").

**Returns:** `string`

### `Type(text string, clearFirst bool)`

*method on `CloudBrowser`*

Type types text into the currently focused element as a per-key stream of real keyboard events (keyDown/char/keyUp with the context's QWERTZ/QWERTY layout and human cadence) — unlike CloudBrowser.InsertText, a single IME-style commit with no key events.

Type is intentionally UNtargeted and loose: it does not locate or focus any element and does NOT pin focus, so the page is free to route keys and move focus between fields mid-stream — ideal for one-time-code / OTP inputs that auto-advance to the next box on each digit. To type one specific field that must stay focused for the whole value, use CloudBrowser.Fill instead (strict, target-bound, per-key focus-verified).

Nothing is focused for you: CloudBrowser.Click (or Fill) the field first, or otherwise ensure focus, before calling Type.

**Parameters:**
- `text` (`string`) — the text to type as real key events
- `clearFirst` (`bool`) — when true, clears the focused field (Ctrl+A, Delete) first

**Throws:**
- `UNKNOWN_ERROR` — the page/context was torn down mid-stream

```go
// OTP field that auto-advances across boxes.
_, _ = browser.Click(ctx, browserscale.CSS("input.otp-0"))
if err := browser.Type(ctx, "123456", false); err != nil {
    log.Fatal(err)
}
```

### `Wait(args ...WaitArg) → *WaitResult`

*method on `CloudBrowser`*

Wait blocks until any of the supplied locators matches.

Pass one or more Locators (built with CSS, JS, …) plus optional wait-level arguments such as Timeout. When several locators are supplied, the first one to match wins; the others are abandoned.

Defaults applied automatically: - timeout: DefaultWaitTimeoutMs (30s) — override with Timeout - per-locator visible/steady: DefaultVisible (true) and DefaultSteadyMs (500) for CSS and JS locators. For JS expressions returning a non-Element value (bool/string/number/object) both flags are no-ops. Override with Locator.Visible / Locator.Steady on individual locators.

Node and At are not valid wait conditions — they only make sense as action targets — and produce an error at send time.

**Parameters:**
- `args` (`...WaitArg`) — one or more Locators plus optional wait-level options;

**Returns:** `*WaitResult` — *WaitResult for the first matching condition (carries the

```go
// Wait for either a success banner or a JS condition, max 5s.
res, err := browser.Wait(ctx,
    browserscale.CSS(".success"),
    browserscale.JS("window.__ready === true"),
    browserscale.Timeout(5000),
)
if err != nil {
    var we *browserscale.WaitError
    if errors.As(err, &we) {
        for _, c := range we.Conditions {
            log.Printf("condition %d: %s", c.Index, c.State)
        }
    }
    log.Fatal(err)
}
_ = res
```

**See also:** WaitError for the timeout detail

### `WaitForAnyRequest(timeoutMs float64, patterns []RequestPattern) → int32, *InterceptedRequest`

*method on `CloudBrowser`*

WaitForAnyRequest blocks until the next request whose URL matches one of the supplied patterns is observed.

Returns the matched pattern's index and the captured request. When patternsi.Abort is true the request is dropped with an empty 200 response instead of being sent to the network.

**Parameters:**
- `timeoutMs` (`float64`) — per-call timeout in milliseconds; 0 uses the server default
- `patterns` (`[]RequestPattern`) — one or more URL patterns (with optional Abort flags)

**Returns:** `int32, *InterceptedRequest` — int32 index of the matched pattern, *InterceptedRequest with

**Throws:**
- `UNKNOWN_ERROR` — the wait timed out or no patterns were supplied

```go
idx, req, err := browser.WaitForAnyRequest(ctx, 5000, []browserscale.RequestPattern{
    {URL: "*/api/login"},
})
if err != nil {
    log.Fatal(err)
}
_ = idx
fmt.Println(req.Method, req.Url)
```

### `WaitForAnyResponse(timeoutMs float64, patterns []RequestPattern) → int32, *InterceptedResponse`

*method on `CloudBrowser`*
*inherits from `CloudBrowser.WaitForAnyRequest`*

WaitForAnyResponse blocks until the next response whose URL matches one of the supplied patterns is observed.

Same shape as CloudBrowser.WaitForAnyRequest but on the response phase. When patternsi.Abort is true the page receives an empty 200 instead of the real response.

**Parameters:**
- `timeoutMs` (`float64`) — per-call timeout in milliseconds; 0 uses the server default
- `patterns` (`[]RequestPattern`) — one or more URL patterns (with optional Abort flags)

**Returns:** `int32, *InterceptedResponse` — int32 index of the matched pattern, *InterceptedResponse with

**Throws:**
- `UNKNOWN_ERROR` — the wait timed out or no patterns were supplied

```go
idx, resp, err := browser.WaitForAnyResponse(ctx, 5000, []browserscale.RequestPattern{
    {URL: "*/api/login"},
})
if err != nil {
    log.Fatal(err)
}
_ = idx
fmt.Println(resp.StatusCode)
```

## Locator

Locator is the universal "what element / what condition" type. It is used both as a wait condition (passed to Wait) and as a target for element actions (passed to Click, Fill, etc.).

Not every field is meaningful in every context: - selector / jsExpression  → both wait and actions - backendNodeId            → actions only (Wait rejects it) - visible / steadyTime     → wait only (silently ignored by actions) - x / y                    → actions only (Wait rejects it) - frameId                  → both, may be overridden by call-level browserscale.InFrame() / browserscale.InAllFrames() options

Use the CSS / JS / Node / At constructors instead of building this struct by hand.

### `InAllFrames() → *Locator`

*method on `Locator`*

InAllFrames scopes this Locator to every frame.

Equivalent to `.InFrame(AllFrames)`. Use this when an element might appear inside any of several frames and you do not want to enumerate them.

**Returns:** `*Locator` — the same Locator for chaining

```go
_, _ = browser.Wait(ctx, browserscale.CSS("button.consent").InAllFrames())
```

### `InFrame(id string) → *Locator`

*method on `Locator`*

InFrame scopes this Locator to a specific frameId.

Use the frameId from a previous result or CloudBrowser.GetPages to target elements inside a known iframe.

**Parameters:**
- `id` (`string`) — id of the frame to scope to

**Returns:** `*Locator` — the same Locator for chaining

```go
pages, _ := browser.GetPages(ctx)
iframeId := pages[0].FrameTree.Children[0].FrameId
_, _ = browser.Click(ctx, browserscale.CSS("button").InFrame(iframeId))
```

### `Steady(ms float64) → *Locator`

*method on `Locator`*

Steady requires the element to keep a stable position and size for at least ms milliseconds before the wait matches.

Pass 0 to disable the default DefaultSteadyMs (500). Has no effect for JS expressions that return a non-Element value, nor when the Locator is used as an action target.

**Parameters:**
- `ms` (`float64`) — steady-state duration in milliseconds; 0 disables

**Returns:** `*Locator` — the same Locator for chaining

```go
_, _ = browser.Wait(ctx, browserscale.CSS(".banner").Steady(0))
```

### `Visible(v bool) → *Locator`

*method on `Locator`*

Visible enforces or disables the visibility check for this Locator's wait condition.

Pass false to opt out of the default DefaultVisible (true). Has no effect when the Locator is used as an action target — actions never check visibility before dispatching.

**Parameters:**
- `v` (`bool`) — true to require visibility, false to skip the check

**Returns:** `*Locator` — the same Locator for chaining

```go
_, _ = browser.Wait(ctx, browserscale.CSS("#hidden").Visible(false))
```

## BrowserConfig

BrowserConfig holds all parameters for renting a browser session. Use NewBrowserConfig with the required fields, then chain optional setters.

### `UnstableWithFakeGpu(renderer string, vendor string, extensions []string) → *BrowserConfig`

*method on `BrowserConfig`*

UnstableWithFakeGpu overrides WebGL UNMASKED_RENDERER_WEBGL, UNMASKED_VENDOR_WEBGL and getSupportedExtensions().

Unstable API — likely to be reshaped or removed without notice. Use only when you have a specific WebGL-fingerprint requirement.

**Parameters:**
- `renderer` (`string`) — value to return for UNMASKED_RENDERER_WEBGL
- `vendor` (`string`) — value to return for UNMASKED_VENDOR_WEBGL
- `extensions` (`[]string`) — list returned by getSupportedExtensions()

**Returns:** `*BrowserConfig` — the modified *BrowserConfig for chaining

```go
cfg.UnstableWithFakeGpu("ANGLE", "Google Inc.", []string{"OES_texture_float"})
```

### `UnstableWithGpuEnabled(enabled bool) → *BrowserConfig`

*method on `BrowserConfig`*

UnstableWithGpuEnabled restricts the rental to hosts that render on a physical GPU instead of the software renderer.

Unstable API — do not build on it. It exists to compare GPU-backed hosts against software rendering while that rollout is in progress; once every host is GPU-backed the flag becomes meaningless and is removed. Note that it narrows the pool: the rental fails rather than falling back to a software-rendered host, so it can report no capacity while ordinary rentals still succeed.

**Parameters:**
- `enabled` (`bool`) — true to require a GPU-backed host

**Returns:** `*BrowserConfig` — the modified *BrowserConfig for chaining

```go
cfg := browserscale.NewBrowserConfig(apiKey, 600, "", 0, "", "").UnstableWithGpuEnabled(true)
```

### `WithCountryCode(countryCode string) → *BrowserConfig`

*method on `BrowserConfig`*

WithCountryCode sets the geo-IP country code for the rented session.

Drives both the assigned exit-IP region and the locale defaults (Accept- Language, timezone fallback) when those are not overridden separately.

**Parameters:**
- `countryCode` (`string`) — ISO-3166 country code (e.g. "DE", "US")

**Returns:** `*BrowserConfig` — the modified *BrowserConfig for chaining

```go
cfg := browserscale.NewBrowserConfig(apiKey, 600, "", 0, "", "").WithCountryCode("DE")
```

### `WithFingerprint(fingerprint string) → *BrowserConfig`

*method on `BrowserConfig`*

WithFingerprint pins a specific browser fingerprint id for the session.

When omitted the server picks a fingerprint based on the country code. Pass a known id (e.g. one returned by a previous rental) to keep fingerprints stable across sessions.

**Parameters:**
- `fingerprint` (`string`) — server-side fingerprint id

**Returns:** `*BrowserConfig` — the modified *BrowserConfig for chaining

```go
cfg := browserscale.NewBrowserConfig(apiKey, 600, "", 0, "", "").WithFingerprint("fp_abc123")
```

### `WithTimezone(timezone string) → *BrowserConfig`

*method on `BrowserConfig`*

WithTimezone sets the IANA timezone for the rented session.

**Parameters:**
- `timezone` (`string`) — IANA timezone (e.g. "Europe/Berlin")

**Returns:** `*BrowserConfig` — the modified *BrowserConfig for chaining

```go
cfg := browserscale.NewBrowserConfig(apiKey, 600, "", 0, "", "").WithTimezone("Europe/Berlin")
```

## BrowserInfo

BrowserInfo describes one running session as ListBrowsers reports it.

**Fields:**
- `SessionId` (`string`)
- `GrpcUrl` (`string`) — GrpcUrl is the endpoint this session is driven from — the same one rent returned. It is what makes a listed id usable: pass it to ConnectSession, or call BrowserInfo.Connect.
- `StartTime` (`int64`) — StartTime is unix seconds.
- `RentDuration` (`int`) — RentDuration is the rental length in seconds; 0 means unlimited.
- `RemainingSeconds` (`*int`) *(optional)* — RemainingSeconds counts down to the end of the rental, and is nil for an unlimited one.
- `CountryCode` (`string`)
- `Timezone` (`string`)
- `ProxyHost` (`string`)
- `PublicIp` (`string`) — PublicIp is the address the session egresses from.
- `GpuIndex` (`*int`) *(optional)* — GpuIndex is the physical card the session renders on, nil on a software-rendered host.

### `Connect(apiKey string) → *CloudBrowser`

*method on `BrowserInfo`*

Connect attaches to this listed session over gRPC.

Shorthand for ConnectSession with the URL and id already in hand. The returned handle owns no rental, so CloudBrowser.CloseConn detaches without ending the session — which is usually what you want for a session you found rather than rented.

**Parameters:**
- `apiKey` (`string`) — API key the session was rented with

**Returns:** `*CloudBrowser` — *CloudBrowser attached to the session

**Throws:**
- `UNKNOWN_ERROR` — the gRPC connection could not be opened

```go
browsers, _ := browserscale.ListBrowsers(ctx, apiKey)
browser, err := browsers[0].Connect(ctx, apiKey)
if err != nil { log.Fatal(err) }
defer browser.CloseConn() // detach; the session keeps running
```

## DomMirror

DomMirror is a live copy of a page's DOM, across every frame in it, returned by CloudBrowser.MirrorDom.

The browser sends the top of the tree once and from then on only what changed in the part you expanded, so a page that churns inside a collapsed subtree costs one number per batch instead of a re-serialized document.

It is one tree. An <iframe> is an element whose one child is the document it hosts; expanding it fetches that document and starts mirroring the frame, collapsing it stops again, and a frame navigating arrives as its owner's child being replaced.

Node ids restart per frame, so a node's address is the pair (DomNode.FrameId, DomNode.BackendNodeId) and never the id alone.

Every method is safe to call from any goroutine.

### `Collapse(node *DomNode)`

*method on `DomMirror`*

Collapse stops reporting changes inside a node, called when the user closes it. The node itself stays in the tree and keeps reporting its child count. A child frame below it stops being mirrored too.

**Parameters:**
- `node` (`*DomNode`) — the node to close

**Throws:**
- `UNKNOWN_ERROR` — the subtree could not be released

### `Expand(node *DomNode, depth int32)`

*method on `DomMirror`*

Expand fetches a node's children and starts reporting changes inside them. This is what a tree view calls when the user opens a node.

On an <iframe> the one child is the document it hosts, and this call is what starts mirroring that frame. Nothing about the result says a process boundary was crossed; it is a child list like any other.

A node that left the tree while the call was in flight is not an error and changes nothing.

**Parameters:**
- `node` (`*DomNode`) — the node to open, from DomMirror.Root or DomMirror.Node
- `depth` (`int32`) — levels below the node; 0 uses the server default of 1

**Throws:**
- `UNKNOWN_ERROR` — the children could not be read

### `FrameIds() → []string`

*method on `DomMirror`*

FrameIds returns every frame with a document in the tree, main frame first. A frame whose <iframe> has not been expanded is not mirrored and not listed.

**Returns:** `[]string`

### `IsExpanded(node *DomNode) → bool`

*method on `DomMirror`*

IsExpanded reports whether this node's children are known. Changes inside a node that is not expanded arrive only as an updated ChildNodeCount.

**Parameters:**
- `node` (`*DomNode`)

**Returns:** `bool`

### `MainFrameId() → string`

*method on `DomMirror`*

MainFrameId returns the page's main frame.

**Returns:** `string`

### `Resync()`

*method on `DomMirror`*

Resync throws away the local copy of the whole page and fetches a fresh one. It happens automatically whenever the browser says the copy is void, so you rarely need to call it.

**Throws:**
- `UNKNOWN_ERROR` — the page could not be re-read; the mirror then ends

### `Reveal(backendNodeId int32, frameId string) → []*DomNode`

*method on `DomMirror`*

Reveal brings a node into the tree together with its ancestors and their siblings, and starts reporting changes along that path.

Use it to focus a node you do not hold — an CloudBrowser.InspectAtPosition hit, say. You cannot walk up to it yourself: it is not in your tree, so there is nothing to walk from.

The node may be in a frame nobody opened, and that works: the chain comes back crossing the frame boundaries it has to, and those frames start being mirrored, exactly as if you had expanded your way there by hand.

**Parameters:**
- `backendNodeId` (`int32`) — the node to reach
- `frameId` (`string`) — the frame its id belongs to; empty targets the main frame

**Returns:** `[]*DomNode` — []*DomNode the ancestor chain, the main document first, or nil if the

**Throws:**
- `UNKNOWN_ERROR` — the path could not be built

### `Root() → *DomNode`

*method on `DomMirror`*

Root returns the main frame's document, or nil before the first snapshot arrived.

The result is an immutable snapshot: the mirror will not modify the nodes it hands out, so it stays consistent to walk while the page keeps changing.

**Returns:** `*DomNode`

### `Seq() → uint64`

*method on `DomMirror`*

Seq returns the page sequence of the last change applied. One clock for the whole page: a change in an out-of-process iframe and one in the main document are ordered against each other.

**Returns:** `uint64`

## MoveError

MoveError is returned as the error from CloudBrowser.MoveTo when the target could not be located. A move has no occlusion notion, so this is the only semantic failure. Implements the error interface; recover with errors.As.

**Fields:**
- `Code` (`string`) — Code is currently always "not_found".
- `Message` (`string`) — Message is a human-readable description.

### `Error() → string`

*method on `MoveError`*

Error implements the error interface.

**Returns:** `string`

## ScriptFollow

ScriptFollow is a read-only view of script output, returned by CloudBrowser.FollowScript.

### `Dropped() → uint64`

*method on `ScriptFollow`*

Dropped reports how many events the server discarded because this reader fell behind.

**Returns:** `uint64`

### `Err()`

*method on `ScriptFollow`*

Err reports why the subscription ended, or nil while it is still open and after a clean stop.

### `Stop()`

*method on `ScriptFollow`*

Stop ends the subscription. Idempotent, and safe to defer. It never cancels a run: other readers, and the script itself, are unaffected.

Once it returns, the handler is no longer running and everything it wrote is visible to the calling goroutine.

## ScriptRun

ScriptRun is a script running in the background, returned by CloudBrowser.StartScript. Its output is delivered to the handler passed there; this handle exists to wait for the outcome and to cancel the run.

### `Detach()`

*method on `ScriptRun`*

Detach stops reading this run's output without cancelling the run. The script keeps going with nobody watching, which is what makes a detached run outlive the process that started it.

### `RunId() → string`

*method on `ScriptRun`*

RunId is the id the browser gave this run. Pass it to CloudBrowser.StopScripts to cancel the run from elsewhere, or to CloudBrowser.FollowScript to watch it from another process.

**Returns:** `string`

## Functions

### `At(x float64, y float64) → *Locator`

*function*

At targets viewport coordinates instead of an element.

Useful for clicking inside a canvas, hovering decorative regions, or dispatching events at synthetic positions. Action-only — using it in CloudBrowser.Wait returns an error at send time. Note that only Click and MoveTo accept At; Scroll, Drag, Fill and Select all require a real element.

**Parameters:**
- `x` (`float64`) — viewport-relative x in CSS pixels
- `y` (`float64`) — viewport-relative y in CSS pixels

**Returns:** `*Locator` — *Locator usable only as an action target

```go
// Click at canvas-relative coordinates.
_, _ = browser.Click(ctx, browserscale.At(120, 240))
```

### `ConnectSession(grpcUrl string, apiKey string, sessionId string) → *CloudBrowser`

*function*

ConnectSession attaches to an already-running session via gRPC.

Use this when you have a session id and gRPC URL from a previous RentBrowser (for example stored across process restarts). Unlike RentBrowser this does not call the rent API — the session must already exist server-side.

**Parameters:**
- `grpcUrl` (`string`) — the session's gRPC endpoint as returned by CloudBrowser.GrpcUrl,
- `apiKey` (`string`) — API key authorizing access to the session
- `sessionId` (`string`) — id of the existing session to attach to

**Returns:** `*CloudBrowser` — *CloudBrowser attached to the existing session; the returned

**Throws:**
- `UNKNOWN_ERROR` — the gRPC connection could not be opened

```go
browser, err := browserscale.ConnectSession(ctx, "grpcs://api.browserscale.cloud:443", apiKey, sessionId)
if err != nil {
    log.Fatal(err)
}
defer browser.Close()
```

### `CSS(selector string) → *Locator`

*function*

CSS waits for / targets an element matching the given CSS selector.

When used in CloudBrowser.Wait, the returned Locator carries the SDK defaults DefaultVisible (true) and DefaultSteadyMs (500). Override per call with Locator.Visible / Locator.Steady (use `.Steady(0)` to disable the steady check).

When used as an action target (Click, etc.) the visible/steady fields are ignored — there are no corresponding fields on the action requests.

**Parameters:**
- `selector` (`string`) — CSS selector matching the element

**Returns:** `*Locator` — *Locator usable as a wait condition or as an action target

```go
// As a wait condition.
_, _ = browser.Wait(ctx, browserscale.CSS("button.submit"))
// As an action target.
_, _ = browser.Click(ctx, browserscale.CSS("button.submit"))
```

### `JS(expression string) → *Locator`

*function*

JS waits for / targets the result of a JavaScript expression.

Same wait defaults as CSS (DefaultVisible=true, DefaultSteadyMs=500); these only apply when the expression returns a DOM Element. For non-Element truthy values (boolean, string, number, plain object) both fields are no-ops and the condition matches as soon as the value is truthy.

Use Locator.Visible(false) / Locator.Steady(0) on the returned Locator to opt out.

**Parameters:**
- `expression` (`string`) — JavaScript expression evaluated in the target frame

**Returns:** `*Locator` — *Locator usable as a wait condition or as an action target

```go
_, _ = browser.Wait(ctx, browserscale.JS("window.__ready === true"))
```

### `ListBrowsers(apiKey string) → []BrowserInfo`

*function*

ListBrowsers reports the sessions an API key currently holds.

Use it to recover session ids the process lost — after a restart, or from a different machine entirely. Without it a rental is only reachable through the handle that created it, so a crash between rent and stop leaves a paid session running with nothing able to name it.

Only live sessions are listed; a stopped one is gone, not reported as ended.

**Parameters:**
- `apiKey` (`string`) — API key whose sessions to list

**Returns:** `[]BrowserInfo` — []BrowserInfo oldest first, empty when the key holds none

**Throws:**
- `UNKNOWN_ERROR` — the list API rejected the request

```go
browsers, err := browserscale.ListBrowsers(ctx, apiKey)
if err != nil {
    log.Fatal(err)
}
for _, b := range browsers {
    fmt.Println(b.SessionId, b.CountryCode)
}
```

### `NewBrowserConfig(apiKey string, rentDuration int, proxyHost string, proxyPort int, proxyUsername string, proxyPassword string) → *BrowserConfig`

*function*

NewBrowserConfig returns a BrowserConfig populated with the required rental fields. Optional fields are configured via the chainable With… setters before passing the config to RentBrowser.

**Parameters:**
- `apiKey` (`string`) — API key authenticating the rental
- `rentDuration` (`int`) — lifetime of the session in seconds
- `proxyHost` (`string`) — upstream proxy host (empty string disables the proxy)
- `proxyPort` (`int`) — upstream proxy port (ignored when proxyHost is empty)
- `proxyUsername` (`string`) — proxy auth user (empty for unauthenticated proxies)
- `proxyPassword` (`string`) — proxy auth password (empty for unauthenticated proxies)

**Returns:** `*BrowserConfig` — *BrowserConfig ready to be customized further or passed to RentBrowser

```go
cfg := browserscale.NewBrowserConfig("sk_…", 600, "", 0, "", "").
    WithCountryCode("DE").
    WithTimezone("Europe/Berlin")
```

### `Node(backendNodeId int32) → *Locator`

*function*

Node targets an element by its DevTools backendNodeId.

Use this when you already have a backendNodeId from a previous result (e.g. WaitResult or EvaluateResult) and want to act on the exact same element without re-resolving by selector. Action-only — using it in CloudBrowser.Wait returns an error at send time.

**Parameters:**
- `backendNodeId` (`int32`) — DevTools backendNodeId of the target element

**Returns:** `*Locator` — *Locator usable only as an action target

```go
res, _ := browser.Click(ctx, browserscale.CSS("button.open"))
_, _ = browser.Click(ctx, browserscale.Node(res.BackendNodeId))
```

### `Ptr(v T) → *T`

*function*

Ptr returns a pointer to v. It is a convenience for the SDK's optional pointer fields where a zero value is meaningful and must be distinguished from "unset" — e.g. FillOpts.TimeoutMs: browserscale.Ptr(0.0) makes Fill one-shot, whereas a nil field takes the server default.

**Parameters:**
- `v` (`T`)

**Returns:** `*T`

### `RentBrowser(config *BrowserConfig) → *CloudBrowser`

*function*

RentBrowser rents a new browser session and returns a connected handle.

Calls the browserscale rent endpoint with the supplied BrowserConfig, opens a gRPC connection to the assigned session host, and returns a ready-to-use CloudBrowser. On any failure the partially-rented session is best-effort released.

**Parameters:**
- `config` (`*BrowserConfig`) — rental parameters built with NewBrowserConfig

**Returns:** `*CloudBrowser` — *CloudBrowser ready to drive the rented session; call

**Throws:**
- `UNKNOWN_ERROR` — the rent API rejected the request or the gRPC

```go
cfg := browserscale.NewBrowserConfig("sk_…", 600, "", 0, "", "")
browser, err := browserscale.RentBrowser(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer browser.Close()
```

### `SetApiEndpoint(endpoint string)`

*function*

SetApiEndpoint overrides the HTTP rent/stop endpoint.

Defaults to `https://api.browserscale.cloud`. Call this before any RentBrowser/StopBrowser call if you need to point at a private browserscale deployment.

**Parameters:**
- `endpoint` (`string`) — base URL of the rent/stop service, with no trailing slash

```go
browserscale.SetApiEndpoint("https://browserscale.internal.example.com")
```

### `StopAllBrowsers(apiKey string) → int`

*function*

StopAllBrowsers releases every session an API key holds.

The blunt instrument, for cleaning up after a run that leaked sessions — a crashed worker pool, an interrupted test. It ends sessions this process never created, including ones another machine is using, so it is not a way to tidy up "my" sessions in a shared account.

Unused credits are refunded per session, as with StopBrowser.

**Parameters:**
- `apiKey` (`string`) — API key whose sessions to release

**Returns:** `int` — int how many sessions were stopped

**Throws:**
- `UNKNOWN_ERROR` — the stop API rejected the request

```go
stopped, err := browserscale.StopAllBrowsers(ctx, apiKey)
```

### `StopBrowser(apiKey string, sessionId string)`

*function*

StopBrowser releases a session without needing a CloudBrowser handle.

Useful when a session id was persisted across processes and the rental outlived the original handle. Only calls the rent stop endpoint; there is no gRPC connection to close in this form.

**Parameters:**
- `apiKey` (`string`) — API key the session was rented with
- `sessionId` (`string`) — id of the session to release

**Throws:**
- `UNKNOWN_ERROR` — the stop API rejected the request

```go
_ = browserscale.StopBrowser(context.Background(), apiKey, sessionId)
```

### `Timeout(ms float64) → WaitArg`

*function*

Timeout overrides the CloudBrowser.Wait timeout.

When omitted, DefaultWaitTimeoutMs (30s) is used. Pass once per Wait call as one of the variadic arguments.

**Parameters:**
- `ms` (`float64`) — timeout in milliseconds

**Returns:** `WaitArg` — a WaitArg suitable for passing to Wait

```go
_, _ = browser.Wait(ctx, browserscale.CSS("#done"), browserscale.Timeout(5000))
```

## Types

### `AuthSession`

*struct*

AuthSession is a portable snapshot of a context's signed-in Google account and/or DBSC sessions. All fields are optional so a context that only has DBSC (no primary account) or only a sign-in (no DBSC) round-trips.

Pair with CloudBrowser.GetCookies / CloudBrowser.SetCookies and CloudBrowser.GetStorage / CloudBrowser.SetStorage to fully move a persona between fresh contexts. Call CloudBrowser.SetAuthSession before navigating.

**Fields:**
- `GaiaID` (`*string`) *(optional)*
- `Email` (`*string`) *(optional)*
- `RefreshToken` (`*string`) *(optional)*
- `WrappedBindingKey` (`*string`) *(optional)*
- `SigninScopedDeviceID` (`*string`) *(optional)*
- `SyncConsent` (`*bool`) *(optional)*
- `DbscSessions` (`[]DbscSession`)

### `ClickError`

*struct*

ClickError is returned as the error from CloudBrowser.Click / CloudBrowser.ClickWith when the click did not land (the element was occluded and the point could not be reached). It implements the error interface, so the ordinary `res, err := browser.Click(...)` shape keeps working; recover the structured detail with errors.As:

res, err := browser.Click(ctx, browserscale.CSS("#buy")) var ce *browserscale.ClickError if errors.As(err, &ce) { // ce.Code, ce.Message, ce.Occluder describe the blocker }

**Fields:**
- `Code` (`string`) — Code is a machine-stable failure code, e.g. "occluded_no_reachable_point" (target fully covered, no exposed part reachable) or "occluded_after_evade" (a reposition was tried but the target was still covered).
- `Message` (`string`) — Message is a human-readable description.
- `Occluder` (`*OccluderInfo`) *(optional)* — Occluder is the intercepting element (present for occlusion codes).
- `EvadeAttempted` (`bool`) — EvadeAttempted reports whether a pointer reposition was tried before giving up.

### `ClickOpts`

*struct*

ClickOpts customizes a CloudBrowser.ClickWith call. Zero/empty values mean "use the server default".

**Fields:**
- `InFrame` (`string`) — InFrame overrides the locator's own frame. Empty = use the locator's frame (or the main frame if none). Pass a specific frameId, or AllFrames, to search elsewhere.
- `Button` (`string`) — Button is the mouse button to use. Valid: "left" (default), "right", "middle".
- `ClickCount` (`int32`) — ClickCount controls single/double-click. 0 or 1 = single click (default), 2 = double-click.
- `Action` (`string`) — Action selects the mouse phase. "" or "click" = full mouseDown+mouseUp (default). "press" only dispatches mouseDown. "release" only dispatches mouseUp at the current cursor position.

### `CookieParam`

*struct*

CookieParam is one cookie returned by GetCookies / passed to SetCookies. Name, Value, Domain, and Path are the common required identity fields; optional attributes mirror the browser's CookieParam shape: URL, Secure, HTTPOnly, SameSite, Expires, Priority, SourceScheme, SourcePort, and PartitionKey.

**Fields:**
- `Name` (`string`)
- `Value` (`string`)
- `URL` (`*string`) *(optional)*
- `Domain` (`string`)
- `Path` (`string`)
- `Secure` (`*bool`) *(optional)*
- `HTTPOnly` (`*bool`) *(optional)*
- `SameSite` (`*string`) *(optional)*
- `Expires` (`*float64`) *(optional)*
- `Priority` (`*string`) *(optional)*
- `SourceScheme` (`*string`) *(optional)*
- `SourcePort` (`*int`) *(optional)*
- `PartitionKey` (`*CookiePartitionKey`) *(optional)*

### `CookiePartitionKey`

*struct*

CookiePartitionKey describes CHIPS partitioning metadata for partitioned cookies.

**Fields:**
- `TopLevelSite` (`string`)
- `HasCrossSiteAncestor` (`bool`)

### `DbscSession`

*struct*

DbscSession is one Device Bound Session Credentials entry.

**Fields:**
- `Site` (`string`) — Site is the serialized schemeful site key, e.g. "https://google.com".
- `Session` (`string`) — Session is base64 of the serialized DBSC Session proto (includes the wrapped binding key; portable under WRC's software key provider).

### `DomChangeHandler`

*type alias*

`type DomChangeHandler = func(...)`

DomChangeHandler is called after the mirrored tree changed, including once for the opening snapshot. Read the new state from DomMirror.Root.

Calls are serialized on a goroutine the SDK owns, so the handler needs no locking of its own, and it is safe to call back into the mirror from it. Calls are also coalesced: several changes in quick succession may produce a single call, which always sees the newest tree. Treat it as "something moved, re-read the root" rather than as one call per edit.

### `DomChildren`

*struct*

DomChildren is the reply to CloudBrowser.GetDomChildren.

**Fields:**
- `Children` (`string`) — Children is a JSON array of DOM.Node. Empty for an id the mirror never handed out, which is also what a stale id from before a resync looks like.
- `Seq` (`uint64`) — Seq is the page sequence this payload is valid as of.

### `DomMirrorOptions`

*struct*

DomMirrorOptions configures CloudBrowser.MirrorDom and CloudBrowser.StartDomMirror.

**Fields:**
- `Depth` (`int32`) — Depth is how many levels to serialize up front. 0 uses the server default of 2 — #document → <html> → <head>/<body>, enough to draw a collapsed tree. -1 walks everything and gives up what the mirror is for.
- `Pierce` (`bool`) — Pierce descends into author shadow roots. Fixed for the life of the mirror.

### `DomNode`

*struct*

DomNode is a node in the mirrored page, in CDP's DOM.Node shape — the same shape CloudBrowser.GetDOM returns, with two additions the mirror needs and a caller usually wants anyway: DomNode.FrameId and DomNode.ContentFrameId.

An <iframe> is an ordinary element here. The document it hosts is its one entry in Children, present once the element has been expanded, and nothing about walking the tree has to know a process boundary runs through it.

Nodes are immutable once handed out. The mirror applies a change by replacing the nodes from the root down to the one that moved, so a tree you took from DomMirror.Root stays a consistent snapshot while the mirror moves on, and unchanged subtrees keep their identity. Do not modify them.

**Fields:**
- `NodeId` (`int32`)
- `BackendNodeId` (`int32`)
- `NodeType` (`int32`)
- `NodeName` (`string`)
- `LocalName` (`string`)
- `NodeValue` (`string`)
- `Attributes` (`[]string`) — Attributes is flat name, value, name, value, ..., as CDP sends it.
- `ChildNodeCount` (`int32`) — ChildNodeCount is the total children in the page, whether or not they are in Children. An <iframe> reports 1: the document it hosts.
- `Children` (`[]*DomNode`) — Children is non-nil once the node has been expanded — empty and non-nil for a node that is expanded and has none.
- `ShadowRoots` (`[]*DomNode`) — ShadowRoots holds author shadow roots, when the mirror was started with DomMirrorOptions.Pierce.
- `FrameId` (`string`) — FrameId is the frame this node lives in. Always set. Together with BackendNodeId this is the node's address: node ids are handed out per renderer and restart per frame, so two frames can and do use the same one, and the id on its own is ambiguous across a page.
- `ContentFrameId` (`string`) — ContentFrameId is set on an element that hosts a frame (<iframe>, <frame>, <object>): the frame it hosts, which is a different frame from FrameId and is the one its child document's ids belong to.

### `DomPath`

*struct*

DomPath is the reply to CloudBrowser.RevealDomNode.

**Fields:**
- `Path` (`string`) — Path is a JSON array of DOM.Node, the main document first, each carrying one level of children, crossing into frames at the document nodes along it. Empty if the node is not on the page.
- `Seq` (`uint64`) — Seq is the page sequence this payload is valid as of.

### `DomResyncHandler`

*type alias*

`type DomResyncHandler = func(...)`

DomResyncHandler is called when the mirror had to be rebuilt, after the new tree is already in place. Rebuilding is automatic; this exists to tell a user why their expanded nodes collapsed.

The reason is one of "documentReplaced", "overflow", "rendererGone", "slowReader" or "manual", and is worth treating as an open set.

### `DomSnapshot`

*struct*

DomSnapshot is the opening snapshot: the main frame's document. Child frames are not in it — their documents are fetched by expanding the <iframe> elements that host them, which is also what starts mirroring them.

**Fields:**
- `Root` (`string`) — Root is the main frame's document as CDP DOM.Node-shaped JSON, the same format GetDOM returns.
- `FrameId` (`string`) — FrameId is the page's main frame.
- `Seq` (`uint64`) — Seq is the page sequence this snapshot is the baseline for. Every update after it carries a higher one.

### `DragError`

*struct*

DragError is returned as the error from CloudBrowser.Drag variants when the source element could not be acquired/pressed. Drag picks up the source with the same smart click as CloudBrowser.Click, so a pre-drag failure is a click failure: Code/Message mirror it and the full click diagnostics live under ClickError. Implements the error interface; recover with errors.As.

**Fields:**
- `Code` (`string`) — Code is mirrored from the underlying click failure: "not_found", "occluded_no_reachable_point" or "occluded_after_evade".
- `Message` (`string`) — Message is a human-readable description (mirrors ClickError.Message).
- `ClickError` (`*ClickError`) *(optional)* — ClickError is the underlying click-core failure at the source pickup.

### `DragResult`

*struct*

DragResult is the outcome of a CloudBrowser.Drag gesture: the resolved source element and the start/end coordinates of the performed drag.

**Fields:**
- `Success` (`bool`)
- `FrameId` (`string`)
- `BackendNodeId` (`int32`)
- `StartX` (`float64`)
- `StartY` (`float64`)
- `EndX` (`float64`)
- `EndY` (`float64`)

### `ElementRef`

*struct*

ElementRef is a lightweight descriptor of an element — enough to identify it (and decide what to do) without another DOM round-trip. It names the element that stole focus in a FillError focus-loss failure.

**Fields:**
- `BackendNodeId` (`int32`) — BackendNodeId is the element's stable backend node id.
- `TagName` (`string`) — TagName is the upper-case tag name, e.g. "INPUT", "BUTTON", "DIV".
- `Id` (`string`) — Id is the id attribute, if present.
- `Name` (`string`) — Name is the name attribute, if present.
- `ClassName` (`string`) — ClassName is the class attribute, if present.
- `InputType` (`string`) — InputType is the <input> type, if the element is an <input>.
- `Text` (`string`) — Text is a whitespace-collapsed textContent/value snippet (max 120 chars).
- `Editable` (`bool`) — Editable is true when the element is itself an editable text sink (input / textarea / contenteditable).

### `ElementResult`

*struct*

ElementResult is the outcome of an element interaction such as CloudBrowser.Click, CloudBrowser.Fill or CloudBrowser.ScrollTo: the resolved element plus the root-relative coordinates the action was performed at.

**Fields:**
- `Success` (`bool`)
- `FrameId` (`string`)
- `BackendNodeId` (`int32`)
- `IsVisible` (`bool`)
- `Bounds` (`Rect`)
- `RootX` (`float64`)
- `RootY` (`float64`)

### `EvaluateResult`

*struct*

EvaluateResult carries the outcome of a JS evaluate call.

If the expression returned a DOM element, BackendNodeId/IsVisible/Bounds are populated and Value is nil. Otherwise Value holds the parsed JSON value (string/number/bool/[]any/mapstringany/nil). On parse failure Value falls back to the raw server string so the caller is never empty- handed.

**Fields:**
- `Value` (`any`)
- `BackendNodeId` (`int32`)
- `IsVisible` (`bool`)
- `Bounds` (`Rect`)

### `FillError`

*struct*

FillError is returned as the error from CloudBrowser.Fill / CloudBrowser.FillWith when the field could not be focused/typed. Fill focuses the field with the exact same smart click as CloudBrowser.Click, so a pre-typing failure is a click failure: Code/Message mirror it and the full click diagnostics live under ClickError. It implements the error interface, so the ordinary `res, err := browser.Fill(...)` shape keeps working; recover the detail with errors.As:

res, err := browser.Fill(ctx, browserscale.CSS("#email"), "a@b.com") var fe *browserscale.FillError if errors.As(err, &fe) && fe.ClickError != nil { // fe.ClickError.Occluder describes the blocker }

**Fields:**
- `Code` (`string`) — Code is the machine-stable failure code. Click-phase codes ("not_found", "occluded_no_reachable_point", "occluded_after_evade") mirror the underlying focus click, with diagnostics under ClickError. The focus codes are "focus_stolen" (another element took focus — FocusedElement names it; Fill is strictly target-bound and will not type into the thief) and "focus_lost" (focus left the target and nothing is focused). For untargeted stream typing that lets focus move (e.g. OTP), use Type.
- `Message` (`string`) — Message is a human-readable description (mirrors ClickError.Message).
- `ClickError` (`*ClickError`) *(optional)* — ClickError is the underlying click-core failure (locate or occlusion) that prevented focusing/typing. Present for the click-phase codes; absent for "focus_stolen"/"focus_lost".
- `FocusedBackendNodeId` (`int32`) — FocusedBackendNodeId is the node that held focus when Fill gave up (0 if nothing was focused), for the "focus_stolen"/"focus_lost" codes.
- `FocusedElement` (`*ElementRef`) *(optional)* — FocusedElement describes the element that grabbed focus instead of the target ("focus_stolen"), so you can act on it (e.g. a consent button).
- `TargetEditable` (`*bool`) *(optional)* — TargetEditable and TargetValueLength report the fill target's own state at the point of failure (the focus codes): whether it is still an editable text sink and its current text length. Both nil when not reported.
- `TargetValueLength` (`*int`) *(optional)*

### `FillOpts`

*struct*

FillOpts customizes a CloudBrowser.FillWith call. Zero/empty values mean "use the server default".

**Fields:**
- `InFrame` (`string`) — InFrame overrides the locator's own frame. Empty = use the locator's frame (or the main frame if none). Pass a specific frameId, or AllFrames, to search elsewhere.
- `ClearFirst` (`bool`) — ClearFirst, when true, wipes the field's existing content with Ctrl+A, Delete before typing. Default (false) appends to whatever is already there.
- `TimeoutMs` (`*float64`) *(optional)* — TimeoutMs bounds focus acquisition (locate, scroll, settle, un-occlude) in ms, mirroring the click timeout. nil = server default (5000). It is a pointer because 0 is meaningful: browserscale.Ptr(0.0) makes Fill one-shot (no retry).
- `SteadyMs` (`*float64`) *(optional)* — SteadyMs is the settle window in ms before the focus click, mirroring the click steady-time. nil = server default (750); browserscale.Ptr(0.0) skips settling.

### `FrameInfo`

*struct*

FrameInfo describes a single frame within a page's frame tree.

**Fields:**
- `FrameId` (`string`)
- `Url` (`string`)
- `IsOOPIF` (`bool`)
- `HasJSContext` (`bool`)
- `IsLoading` (`bool`)
- `IsVisible` (`bool`)
- `AbsoluteRect` (`Rect`)
- `RelativeRect` (`Rect`)
- `Children` (`[]*FrameInfo`)

### `Header`

*struct*

Header is a single HTTP header (name/value pair) on an intercepted request or response.

**Fields:**
- `Name` (`string`)
- `Value` (`string`)

### `HeaderModification`

*struct*

HeaderModification is one entry passed to CloudBrowser.ModifyRequest. Build it as a plain struct literal.

**Fields:**
- `Action` (`HeaderModificationAction`) — Action selects what happens: HeaderModificationAdd inserts a new header, HeaderModificationEdit replaces an existing header's value, HeaderModificationRemove drops the header.
- `Name` (`string`) — Name is the header name the action applies to.
- `Value` (`string`) — Value is the header value for add/edit; ignored for remove.
- `Before` (`string`) — Before positions an "add" immediately before the named existing header; otherwise the header is appended at the end. Ignored for edit/remove.
- `After` (`string`) — After positions an "add" immediately after the named existing header. Mirror of Before; ignored for edit/remove.

### `HeaderModificationAction`

*type alias*

`type HeaderModificationAction = string`

HeaderModificationAction is the verb of a HeaderModification. Matches the add/edit/remove action strings; use the HeaderModificationXxx constants.

### `IceServer`

*struct*

IceServer is one entry for a WebRTC RTCPeerConnection's ICE configuration: a TURN (or STUN) URL plus the short-lived credentials to authenticate with it. Pass these to your peer before creating the SDP offer.

**Fields:**
- `URLs` (`[]string`) — URLs are the ICE server URLs (e.g. "turn:relay.example.com:3478?transport=udp").
- `Username` (`string`) — Username is the short-lived TURN REST username (empty for plain STUN).
- `Credential` (`string`) — Credential is the short-lived TURN REST credential (empty for plain STUN).

### `InspectResult`

*struct*

InspectResult describes the topmost element hit at viewport-relative (x, y). BackendNodeId == 0 means nothing was found at that position.

**Fields:**
- `BackendNodeId` (`int32`)
- `FrameId` (`string`)
- `TagName` (`string`)
- `TextContent` (`string`)
- `IsVisible` (`bool`)
- `Bounds` (`Rect`)

### `InterceptedRequest`

*struct*

InterceptedRequest describes an outgoing request captured by CloudBrowser.WaitForAnyRequest.

**Fields:**
- `Method` (`string`)
- `Url` (`string`)
- `Headers` (`[]Header`)
- `Body` (`string`)
- `ResourceType` (`string`)

### `InterceptedResponse`

*struct*

InterceptedResponse describes a network response captured by CloudBrowser.WaitForAnyResponse.

**Fields:**
- `Url` (`string`)
- `StatusCode` (`int32`)
- `Headers` (`[]Header`)
- `Body` (`string`)

### `NavigateResult`

*struct*

NavigateResult reports where a CloudBrowser.Navigate call ended up after redirects.

**Fields:**
- `FrameId` (`string`)
- `Url` (`string`)

### `NetworkBodies`

*type alias*

`type NetworkBodies = string`

NetworkBodies selects how much of a response body network capture keeps.

### `NetworkCapture`

*struct*

NetworkCapture is a running capture, returned by CloudBrowser.CaptureNetwork. Exchanges are delivered to the handler passed there; this handle only exists to stop the capture and to report how it went.

### `NetworkCaptureOptions`

*struct*

NetworkCaptureOptions configures CloudBrowser.CaptureNetwork.

There is deliberately no byte-cap option: buffer sizes bound memory on a machine shared with other sessions, so the server owns them.

**Fields:**
- `Patterns` (`[]string`) — Patterns are URL wildcards to capture; nil captures every request the session makes. Prefix a pattern with "!" to exclude it, which is the short way to say "everything except this".
- `Bodies` (`NetworkBodies`) — Bodies selects response-body capture. Empty means NetworkBodiesNone.
- `BodyPatterns` (`[]string`) — BodyPatterns narrows body capture to a subset of the captured requests; nil applies Bodies to all of them. Use it to log every request but only keep the payloads you care about.

### `NetworkExchange`

*struct*

NetworkExchange is one request together with the response it received, as reported by CloudBrowser.CaptureNetwork.

A redirect chain arrives as one exchange per hop: the hops share ChainId and count up RedirectIndex, so a 302 and the request it points at are two exchanges, each with its own headers and status.

**Fields:**
- `RequestId` (`string`) — RequestId is unique per hop.
- `ChainId` (`string`) — ChainId is shared by every hop of one redirect chain.
- `RedirectIndex` (`int32`) — RedirectIndex is 0 for the original request and counts up once per redirect followed.
- `FrameId` (`string`) — FrameId is the frame that issued the request; empty for worker traffic.
- `IsOOPIF` (`bool`) — IsOOPIF reports whether that frame runs in its own process. Capture happens in the browser process, so cross-process iframes are included.
- `ResourceType` (`NetworkResourceType`)
- `Method` (`string`)
- `Url` (`string`)
- `InitiatorUrl` (`string`) — InitiatorUrl is the origin that started the request; empty when the browser itself did.
- `RequestHeaders` (`[]Header`)
- `RequestHeadersAreWire` (`bool`) — RequestHeadersAreWire reports whether RequestHeaders are the bytes actually sent — Cookie, User-Agent and Sec-* included — rather than what the page asked for before the network stack filled in the rest.
- `RequestBody` (`[]byte`) — RequestBody holds an inline body only. File and streamed uploads set RequestBodyTruncated instead of appearing here.
- `RequestBodyTruncated` (`bool`)
- `HasResponse` (`bool`) — HasResponse is false when the request failed before any response arrived; Error then says why.
- `StatusCode` (`int32`)
- `StatusText` (`string`)
- `MimeType` (`string`)
- `Protocol` (`string`) — Protocol is the negotiated ALPN protocol, e.g. "h2" or "http/1.1".
- `RemoteAddress` (`string`)
- `ServedFrom` (`NetworkServedFrom`)
- `ResponseHeaders` (`[]Header`)
- `ResponseHeadersAreWire` (`bool`)
- `ResponseBody` (`[]byte`) — ResponseBody is populated only when body capture was requested for this URL and applied; check ResponseBodyCaptured to tell an empty body from an uncaptured one. Binary content does not survive the browser boundary intact — see NetworkBodiesAll.
- `ResponseBodyTruncated` (`bool`)
- `ResponseBodyCaptured` (`bool`)
- `EncodedDataLength` (`int64`)
- `Error` (`string`) — Error is the net error name (e.g. "net::ERR_ABORTED"), empty on success.

### `NetworkExchangeHandler`

*type alias*

`type NetworkExchangeHandler = func(...)`

NetworkExchangeHandler is called once per completed exchange.

Calls are sequential and in the order the browser finished the requests, so the hops of a redirect chain arrive in order and the handler needs no locking of its own. It runs on a goroutine the SDK owns, not the caller's.

Blocking here stalls the capture: the server buffers a bounded amount per reader and then drops its oldest entries, which NetworkCapture.Dropped reports. Hand slow work (disk, HTTP, a database) to another goroutine.

### `NetworkResourceType`

*type alias*

`type NetworkResourceType = string`

NetworkResourceType is the kind of load an exchange belongs to. It is a string rather than a closed enum so an exchange from a newer browser still round-trips instead of decoding to an empty value; compare against the NetworkResource* constants.

### `NetworkServedFrom`

*type alias*

`type NetworkServedFrom = string`

NetworkServedFrom says where an exchange's response came from.

### `ObservationOpts`

*struct*

ObservationOpts customizes a CloudBrowser.GetObservationWith call. Zero/empty values mean "use the server default".

**Fields:**
- `Format` (`string`) — Format is "text" (default) for the compact line format meant to be handed to a model as-is, or "json" for the structured form. Only the requested representation is built, so asking for one does not cost the other.
- `MaxElementsPerFrame` (`int32`) — MaxElementsPerFrame caps emitted elements per frame. 0 = server default (800). This is a safety net against runaway documents; MaxTotalTokens is the limit that normally binds.
- `MaxTextLength` (`int32`) — MaxTextLength caps human-readable strings (labels, text, values) in characters. 0 = server default (300). Identifier-like attributes (type, name, role) have their own fixed, shorter cap and are unaffected.
- `MaxTotalTokens` (`int32`) — MaxTotalTokens budgets the whole page in estimated tokens rather than characters, because the same character count is worth roughly four times as many tokens in CJK text as in ASCII. 0 = server default (8000). Frames are visited in tree order and each gets whatever is left.
- `IncludeBounds` (`bool`) — IncludeBounds adds bounds="x,y,w,h" to every row. Off by default; bounds cost about as much as the rest of a row and are rarely needed, since elements are addressed by backendNodeId.
- `ViewportOnly` (`bool`) — ViewportOnly limits the walk to elements intersecting the frame's current viewport. Off by default.
- `BackendNodeId` (`int32`) — Subtree scope — set exactly one of BackendNodeId, Selector or JSExpression to observe only that element's subtree (follow-up looks at a form then cost the form, not the ads around it). Omit all three for the whole page. Child iframes reached inside the scope are still visited.
- `Selector` (`string`)
- `JSExpression` (`string`)
- `InFrame` (`string`) — InFrame looks up the scope root: empty = main frame, a frameId, or AllFrames. Ignored when observing the whole page.

### `OccluderInfo`

*struct*

OccluderInfo describes the element that intercepted a click — the element sitting on top of the target at the intended click point. Coordinates are in root-viewport CSS pixels. Populated on ClickError for occlusion failures so the caller can locate and clear the blocker (e.g. find its close button).

**Fields:**
- `BackendNodeId` (`int32`)
- `FrameId` (`string`)
- `TagName` (`string`)
- `Id` (`string`)
- `ClassName` (`string`)
- `Text` (`string`)
- `Bounds` (`Rect`)
- `PointerEvents` (`string`) — PointerEvents is the blocker's computed pointer-events keyword (e.g. "auto", "none", "all"). Lets you tell an invisible pass-through layer from one that genuinely swallows the click.
- `Visibility` (`string`) — Visibility is the blocker's computed visibility keyword ("visible", "hidden", "collapse").
- `Opacity` (`float64`) — Opacity is the blocker's computed opacity (0..1). 0 means visually invisible but it may still intercept clicks depending on PointerEvents.
- `ZIndex` (`string`) — ZIndex is the blocker's computed effective z-index as a string ("0" when auto / not stacked).
- `HittableWhileInvisible` (`bool`) — HittableWhileInvisible is true when the blocker intercepts clicks even while invisible (computed pointer-events in {all, painted, fill, stroke}): a real click is swallowed even at visibility:hidden / opacity:0. When false and the element is invisible, a real click would fall through.
- `Position` (`string`) — Position is the computed position keyword. "fixed"/"sticky" means the blocker is pinned (by itself or an ancestor) and stays put no matter where the pointer goes — clear it by scrolling the target out from under it; ordinary overlays often collapse once the pointer leaves.

### `PageInfo`

*struct*

PageInfo describes an open page (tab or popup) inside a browser context.

**Fields:**
- `PageId` (`string`)
- `BrowserContextId` (`string`)
- `Url` (`string`)
- `Title` (`string`)
- `Viewport` (`Rect`)
- `FrameTree` (`FrameInfo`)

### `ReactionInfo`

*struct*

ReactionInfo describes a still-pending reaction, as returned by CloudBrowser.ListReactions. One-shot reactions that have already fired are gone and never appear here.

**Fields:**
- `ReactionID` (`string`) — ReactionID is the stable id assigned by AddReaction (pass to RemoveReaction).
- `MatchSelector` (`string`) — MatchSelector is set if the reaction matches by CSS selector.
- `MatchJsExpression` (`string`) — MatchJsExpression is set if the reaction matches by JS expression.
- `ActionSelector` (`string`) — ActionSelector is set if the click target differs from the matched element.
- `ActionJsExpression` (`string`) — ActionJsExpression is set if the click target differs from the matched element.
- `FrameID` (`string`) — FrameID is the frame scope: "" for the main frame, a specific frameId, or AllFrames.
- `Visible` (`bool`) — Visible reports whether the match additionally requires visibility.

### `ReactionOpts`

*struct*

ReactionOpts customizes CloudBrowser.AddReactionWith. Zero/empty values mean "use the server default".

**Fields:**
- `On` (`*Locator`) *(optional)* — On overrides the click target. Nil = click the matched element itself. Provide a CSS or JS Locator to click a different element, resolved in the matched element's frame (e.g. a modal's close "X"). Node/At locators are rejected.
- `Button` (`string`) — Button is the mouse button for the click. Valid: "left" (default), "right", "middle".
- `ClickCount` (`int32`) — ClickCount controls single/double-click. 0 or 1 = single click (default), 2 = double-click.
- `IntervalMs` (`float64`) — IntervalMs is the poll cadence in milliseconds for the shared page loop. 0 = server default (300ms).

### `ReadCanvasOpts`

*struct*

ReadCanvasOpts customizes a CloudBrowser.ReadCanvasWith call. Zero/empty values mean "use the server default".

**Fields:**
- `InFrame` (`string`) — InFrame overrides the locator's own frame. Empty = use the locator's frame (or the main frame if none). Pass a specific frameId, or AllFrames, to search elsewhere.
- `Format` (`string`) — Format is the output encoding. "" or "png" (default), "jpeg", "webp", or "rgba" for the raw unpremultiplied RGBA pixel buffer.
- `Quality` (`int32`) — Quality is the encode quality 0-100 for "jpeg"/"webp" (ignored otherwise). 0 = server default (90).
- `SX` (`int32`) — SX, SY, SW, SH is an optional sub-rectangle in canvas pixels (mirrors getImageData(sx, sy, sw, sh)). The full canvas is read when SW/SH <= 0.
- `SY` (`int32`) — SX, SY, SW, SH is an optional sub-rectangle in canvas pixels (mirrors getImageData(sx, sy, sw, sh)). The full canvas is read when SW/SH <= 0.
- `SW` (`int32`) — SX, SY, SW, SH is an optional sub-rectangle in canvas pixels (mirrors getImageData(sx, sy, sw, sh)). The full canvas is read when SW/SH <= 0.
- `SH` (`int32`) — SX, SY, SW, SH is an optional sub-rectangle in canvas pixels (mirrors getImageData(sx, sy, sw, sh)). The full canvas is read when SW/SH <= 0.

### `ReadCanvasResult`

*struct*

ReadCanvasResult is the pixel readback of a <canvas>, returned by CloudBrowser.ReadCanvas. DataBase64 holds the encoded image bytes (PNG by default) or the raw RGBA buffer when Opts.Format == "rgba". OriginClean reports whether the canvas was untainted (informational; the read succeeds either way).

**Fields:**
- `Success` (`bool`)
- `FrameId` (`string`)
- `BackendNodeId` (`int32`)
- `DataBase64` (`string`)
- `Width` (`int32`)
- `Height` (`int32`)
- `OriginClean` (`bool`)

### `Rect`

*struct*

Rect describes a position and size in CSS pixels.

**Fields:**
- `X` (`float64`)
- `Y` (`float64`)
- `Width` (`float64`)
- `Height` (`float64`)

### `RequestPattern`

*struct*

RequestPattern matches a URL pattern in WaitForAnyRequest/Response. Set Abort to true to drop the request with an empty 200 response instead of letting it through to the network.

**Fields:**
- `URL` (`string`)
- `Abort` (`bool`)

### `ScreenshotResult`

*struct*

ScreenshotResult is a single captured image of the page, returned by CloudBrowser.Screenshot. DataBase64 holds the encoded image bytes (PNG by default); Width and Height are in physical pixels.

**Fields:**
- `DataBase64` (`string`)
- `Width` (`int32`)
- `Height` (`int32`)

### `ScriptEvent`

*struct*

ScriptEvent is one item on a session's script event stream. Exactly one of Log and Finished is set.

**Fields:**
- `RunId` (`string`) — RunId is the run that produced this event.
- `Log` (`*ScriptLogEntry`) *(optional)* — Log is a console line the script printed.
- `Finished` (`*ScriptFinished`) *(optional)* — Finished marks the end of the run. No further event for that run follows.

### `ScriptEventHandler`

*type alias*

`type ScriptEventHandler = func(...)`

ScriptEventHandler is called once per script event.

Calls are sequential and in the order the browser produced them, so a run's last log line always arrives before its Finished and the handler needs no locking of its own. It runs on a goroutine the SDK owns, not the caller's.

Blocking here stalls delivery: the server buffers a bounded number of events per reader and then drops its oldest, which ScriptRun.Dropped reports. Hand slow work to another goroutine.

### `ScriptFinished`

*struct*

ScriptFinished says how a run ended.

**Fields:**
- `Success` (`bool`) — Success is false when the script failed to compile or threw; Result then holds the message.
- `Result` (`string`) — Result is the return value as JSON, or the error message.
- `Stopped` (`bool`) — Stopped is true when the run was cancelled, or the session went away under it, rather than the script returning on its own.

### `ScriptLogEntry`

*struct*

ScriptLogEntry is one console.* call from a script.

**Fields:**
- `Level` (`string`) — Level is "info", "warning" or "error", from console.log / .warn / .error.
- `Message` (`string`) — Message holds the logged arguments, already stringified the way console does it.
- `Timestamp` (`time.Time`) — Timestamp is when the script printed the line, stamped in the browser.

### `ScriptResult`

*struct*

ScriptResult is the outcome of a blocking CloudBrowser.RunScript.

**Fields:**
- `Success` (`bool`) — Success is false when the script failed to compile or threw; Result then holds the message.
- `Result` (`string`) — Result is the return value as JSON, or "undefined" when the script returned nothing. On failure it is the error message.
- `RunId` (`string`) — RunId names the run. It arrives with the reply, so it is only useful after the fact — to match up log lines a separate follower already saw.
- `Log` (`[]ScriptLogEntry`) — Log is everything the script printed, in order.
- `Truncated` (`bool`) — Truncated is true when the script printed more than the reply holds, in which case Log is the tail of the output rather than all of it.

### `ScriptRunInfo`

*struct*

ScriptRunInfo is one run still in flight, as CloudBrowser.ListScriptRuns reports it.

**Fields:**
- `RunId` (`string`)
- `Running` (`time.Duration`) — Running is how long the run has been going.

### `ScrollError`

*struct*

ScrollError is returned as the error from CloudBrowser.ScrollTo when the target could not be located/scrolled. Implements the error interface; recover with errors.As.

**Fields:**
- `Code` (`string`) — Code is currently always "not_found".
- `Message` (`string`) — Message is a human-readable description.

### `SelectOptionError`

*struct*

SelectOptionError is returned as the error from CloudBrowser SelectByXxx calls when the option could not be selected. selectOption is programmatic (no pointer gate), so it only reports semantic failures. Implements the error interface; recover with errors.As.

**Fields:**
- `Code` (`string`) — Code is "not_found" (the <select> was not located) or "option_not_found" (no option matched the requested index/value/text).
- `Message` (`string`) — Message is a human-readable description.

### `SelectOptionResult`

*struct*

SelectOptionResult reports which <option> a SelectByXxx call ended up selecting.

**Fields:**
- `Success` (`bool`)
- `SelectedIndex` (`int32`)
- `SelectedValue` (`string`)
- `SelectedText` (`string`)

### `SelectOpts`

*struct*

SelectOpts customizes a SelectByXxxWith call. Zero/empty values mean "use the server default".

**Fields:**
- `InFrame` (`string`) — InFrame overrides the locator's own frame. Empty = use the locator's frame (or the main frame if none). Pass a specific frameId, or AllFrames, to search elsewhere.
- `NoEvents` (`bool`) — NoEvents picks the option silently without firing input/change events. Default (false) fires the standard events.

### `StorageItem`

*struct*

StorageItem is a single localStorage key/value pair.

**Fields:**
- `Key` (`string`)
- `Value` (`string`)

### `StorageOriginEntry`

*struct*

StorageOriginEntry groups the localStorage entries of one origin (e.g. "https://example.com"). GetStorage returns these and SetStorage accepts the same shape, so a dump can be fed back verbatim.

**Fields:**
- `Origin` (`string`)
- `Items` (`[]StorageItem`)

### `StreamAnswer`

*struct*

StreamAnswer is the browser's reply to a CloudBrowser.StartStream.

**Fields:**
- `AnswerSDP` (`string`) — SDP answer to apply as your peer's remote description.
- `Viewport` (`Rect`) — Root viewport in CSS pixels, the coordinate space the stream's input data channels expect. X/Y are always 0. Map your on-screen pointer positions into this space before sending them; the video may be displayed at any size. It arrives with the answer rather than from a separate GetPages so it cannot race the stream, and the browser pushes {"type":"viewport","width":W,"height":H} on the reliable "input" channel whenever it changes.

### `WaitArg`

*interface*

WaitArg is the marker interface for everything Wait accepts: a Locator (treated as a condition) or a wait-level option such as Timeout / InFrame / InAllFrames.

### `WaitConditionStatus`

*struct*

WaitConditionStatus is the per-condition diagnostic carried by WaitError when a CloudBrowser.Wait times out: one entry per condition (in the order they were passed) explaining why it never matched.

**Fields:**
- `Index` (`int32`) — Index into the condition list this entry describes.
- `State` (`string`) — State is the last observed state: "not_found", "found_hidden", "found_occluded" (only when the condition required visibility), or "pending_steady".
- `BackendNodeId` (`int32`) — BackendNodeId last seen for this condition (0 if never found).
- `FrameId` (`string`) — FrameId where it was last seen (empty if never found).
- `IsVisible` (`bool`) — IsVisible reports whether it was CSS-visible at the last observation.
- `Bounds` (`*Rect`) *(optional)* — Bounds is the last known rect in root-viewport coordinates (nil if never found).
- `Occluder` (`*OccluderInfo`) *(optional)* — Occluder is the intercepting element, present iff State == "found_occluded".

### `WaitError`

*struct*

WaitError is returned as the error from CloudBrowser.Wait when no condition matched before the deadline. It implements the error interface, so the ordinary `res, err := browser.Wait(...)` shape keeps working; recover the structured detail (including the per-condition breakdown) with errors.As:

res, err := browser.Wait(ctx, browserscale.CSS(".ready")) var we *browserscale.WaitError if errors.As(err, &we) { for _, c := range we.Conditions { log.Printf("condition %d: %s", c.Index, c.State) } }

**Fields:**
- `Code` (`string`) — Code is a machine-stable failure code, currently always "timeout".
- `Message` (`string`) — Message is a human-readable description.
- `Conditions` (`[]WaitConditionStatus`) — Conditions holds the per-condition status, same order/length as the conditions passed to Wait.

### `WaitResult`

*struct*

WaitResult is the outcome of a CloudBrowser.Wait / CloudBrowser.WaitForAny call: which condition matched (Index, in argument order) and where the matched element lives.

**Fields:**
- `Index` (`int32`)
- `FrameId` (`string`)
- `BackendNodeId` (`int32`)
- `IsVisible` (`bool`)
- `Bounds` (`Rect`)
