<!--
  url: https://browserscale.cloud/docs/guides/cookies
  title: Cookies, storage & sessions
  description: Bring a user back across runs: cookies and storage as data, a signed-in login as one portable object, a pinned machine identity and reattaching to live sessions.
-->

# Cookies, storage & sessions

The headline question this guide answers is: **how do I make the next
session look like the same user as the last one?** Four pieces of state
control that — the cookies the browser is carrying, the localStorage the
sites have written, the account the browser itself is signed in to, and
the machine the browser presents — and all of them are designed to
survive across rentals.

> **TL;DR**
>
> - Cookies live in the browser context. `GetCookies` / `SetCookies` / `ClearCookies` are the entire surface.
> - The cookie identity is the `(name, domain, path)` tuple. `SetCookies` upserts on that key.
> - localStorage has the same three-call surface: `GetStorage` / `SetStorage` / `ClearStorage`, grouped by origin. `SetStorage` accepts exactly what `GetStorage` returns, so a dump replays verbatim.
> - A signed-in login travels as one object: `GetAuthSession` / `SetAuthSession` export the browser's signed-in account together with its device-bound sessions (DBSC), and bring them up signed in inside a fresh context.
> - Every session is rented with a **fingerprint id** — either one you pin via `WithFingerprint(...)` / `withFingerprint(...)`, or one the server picks based on your country code. Read it back with `browser.Fingerprint()` (Go) / `browser.getFingerprint()` (TS).
> - To come back as the same user across runs: store the fingerprint id + cookies + storage (+ auth session) on first rental, replay all of it on the next.
> - `ConnectSession` (Go) / `createWebSocketBrowser` (TS) reattach to an existing rental from another process. They do *not* extend the rental lifetime.

## The cookie surface

There are exactly three calls, and they all operate on the whole
browser context (across every tab and frame). No flags, no filters —
read everything, write everything, clear everything.

**Go:**

```go
// Read all cookies.
cookies, _ := browser.GetCookies(ctx)
for _, c := range cookies {
    fmt.Println(c.Name, "=", c.Value, "@", c.Domain, c.Path)
}

// Write or upsert cookies.
secure := true
httpOnly := true
sameSite := "Lax"
_ = browser.SetCookies(ctx, []browserscale.CookieParam{
    {
        Name:     "session",
        Value:    "abc123",
        Domain:   "example.com",
        Path:     "/",
        Secure:   &secure,
        HTTPOnly: &httpOnly,
        SameSite: &sameSite,
    },
    {Name: "locale", Value: "en-US", Domain: "example.com", Path: "/"},
})

// Wipe all cookies (no per-cookie deletion).
_ = browser.ClearCookies(ctx)
```

**TypeScript:**

```ts
const cookies = await browser.getCookies();
for (const c of cookies) {
    console.log(c.name, "=", c.value, "@", c.domain, c.path);
}

await browser.setCookies([
    {
        name: "session",
        value: "abc123",
        domain: "example.com",
        path: "/",
        secure: true,
        httpOnly: true,
        sameSite: "Lax",
    },
    { name: "locale", value: "en-US", domain: "example.com", path: "/" },
]);

await browser.clearCookies();
```

A `CookieParam` includes the common identity fields `name`, `value`,
`domain`, and `path`, plus optional browser cookie attributes such as
`secure`, `httpOnly`, `sameSite`, `expires`, `priority`, `sourceScheme`,
`sourcePort`, and `partitionKey`. The server uses the `(name, domain, path)`
tuple as the cookie's identity — calling `SetCookies` with the same tuple
**overwrites** the existing cookie rather than adding a duplicate.

## The storage surface

localStorage gets the same three-call treatment: read it, write it,
clear it — grouped by origin. The storage database is read and written
directly in the browser process, so **no page needs to be open** for any
of these calls.

**Go:**

```go
// Read localStorage for every origin (pass an origin to filter).
storage, _ := browser.GetStorage(ctx, "")
for _, e := range storage {
    for _, item := range e.Items {
        fmt.Println(e.Origin, item.Key, "=", item.Value)
    }
}

// Write entries. The structure is exactly what GetStorage returns,
// so a dump can be fed back verbatim. Existing keys are overwritten.
_ = browser.SetStorage(ctx, []browserscale.StorageOriginEntry{
    {
        Origin: "https://example.com",
        Items: []browserscale.StorageItem{
            {Key: "token", Value: "abc123"},
            {Key: "theme", Value: "dark"},
        },
    },
})

// Wipe one origin — or pass "" to wipe everything.
_ = browser.ClearStorage(ctx, "https://example.com")
```

**TypeScript:**

```ts
// Read localStorage for every origin (pass an origin to filter).
const storage = await browser.getStorage();
for (const e of storage) {
    for (const { key, value } of e.items) {
        console.log(e.origin, key, "=", value);
    }
}

// Write entries. The structure is exactly what getStorage() returns,
// so a dump can be fed back verbatim. Existing keys are overwritten.
await browser.setStorage([
    {
        origin: "https://example.com",
        items: [
            { key: "token", value: "abc123" },
            { key: "theme", value: "dark" },
        ],
    },
]);

// Wipe one origin — or call with no argument to wipe everything.
await browser.clearStorage("https://example.com");
```

Two scope notes: only **first-party localStorage** is covered —
`sessionStorage` is per-tab and never included — and pages that are
already open will not observe `SetStorage` writes until they reload,
because their renderer cache is not invalidated.

## Persisting cookies between runs

The simplest "make me look logged in next time" recipe: do the login
once, dump the cookie jar to disk, restore it on the next rental.

**Go:**

```go
// First run: log in, then persist.
_, _ = browser.Navigate(ctx, "https://example.com/login", 0)
_, _ = browser.Wait(ctx, browserscale.CSS("#email"))
_, _ = browser.Fill(ctx, browserscale.CSS("#email"), "alice@example.com")
_, _ = browser.Fill(ctx, browserscale.CSS("#password"), "hunter2")
_, _ = browser.Click(ctx, browserscale.CSS("button[type=submit]"))

cookies, _ := browser.GetCookies(ctx)
buf, _ := json.Marshal(cookies)
_ = os.WriteFile("cookies.json", buf, 0o600)

// Next run (same script or a different process): restore.
buf, _ := os.ReadFile("cookies.json")
var cookies []browserscale.CookieParam
_ = json.Unmarshal(buf, &cookies)
_ = browser.SetCookies(ctx, cookies)
_, _ = browser.Navigate(ctx, "https://example.com/dashboard", 0)
```

**TypeScript:**

```ts
// First run.
await browser.navigate("https://example.com/login");
await browser.wait(css("#email"));
await browser.fill(css("#email"), "alice@example.com");
await browser.fill(css("#password"), "hunter2");
await browser.click(css("button[type=submit]"));

const cookies = await browser.getCookies();
await fs.writeFile("cookies.json", JSON.stringify(cookies), { mode: 0o600 });

// Next run.
const restored = JSON.parse(await fs.readFile("cookies.json", "utf8"));
await browser.setCookies(restored);
await browser.navigate("https://example.com/dashboard");
```

Two things this *doesn't* solve on its own:

- The new session ships with a different **fingerprint** — same
  cookies on a brand-new browser identity is the classic "weird
  session" signal that triggers re-auth on cautious sites.
- The new session is on a different **exit IP** — a session cookie
  followed by an IP swap can also force a re-login.

Both are fixable. The fingerprint side is further down in this guide;
the IP side comes down to reusing the same proxy (or pinning a
country code if you're using managed proxies — see
[`BrowserConfig`](/docs/api-reference/go#NewBrowserConfig) /
[TS](/docs/api-reference/ts#BrowserConfig)).

## A signed-in login as one object

Cookies carry a site's session. Some logins live one level deeper: in
the account the browser itself is signed in to, and in **device-bound
sessions** (DBSC) that tie a site's session to a key held by that
browser. Copying cookies alone does not bring those back — the site
sees a session whose device key is missing and asks for the password
again.

`GetAuthSession` exports both as one portable `AuthSession`: the
signed-in account with its refresh token, the binding key and device
id that must travel with it, and every DBSC session of the context. It
reads state in the browser process, so no page needs to be open, and
returns nothing when the context has neither a sign-in nor DBSC
sessions. `SetAuthSession` imports it into a fresh context, which then
comes up signed in.

**Go:**

```go
// First run: sign in, then export.
auth, err := browser.GetAuthSession(ctx)
if err != nil {
    log.Fatal(err)
}
if auth != nil {
    buf, _ := json.Marshal(auth)
    _ = os.WriteFile("auth.json", buf, 0o600)
}

// Next run, on a fresh rental: import before navigating.
buf, _ := os.ReadFile("auth.json")
var saved browserscale.AuthSession
_ = json.Unmarshal(buf, &saved)
_ = browser.SetAuthSession(ctx, saved)
_, _ = browser.Navigate(ctx, "https://mail.google.com", 0)
```

**TypeScript:**

```ts
// First run: sign in, then export.
const auth = await browser.getAuthSession();
if (auth) await fs.writeFile("auth.json", JSON.stringify(auth), { mode: 0o600 });

// Next run, on a fresh rental: import before navigating.
const saved = JSON.parse(await fs.readFile("auth.json", "utf8"));
await browser.setAuthSession(saved);
await browser.navigate("https://mail.google.com");
```

Call `SetAuthSession` before the first navigation, and pair it with
`SetCookies` / `SetStorage` to restore the whole persona. Treat the
file like a password: the refresh token in it signs the account in.

## Session lifecycle in one paragraph

Quick refresher (covered in depth in
[Core concepts](/docs/concepts)): `RentBrowser` rents a session for
the duration in your config and hands you a `CloudBrowser` handle.
The session lives until the rental timer runs out or you call
`StopBrowser` / `Close`. From a different process, `ConnectSession`
(Go) / `createWebSocketBrowser` (TS) attaches to an existing session by
its id and transport URL — useful when you persisted those across a
restart. Closing an *attached* handle closes only the connection, the
underlying rental keeps running until the rent timer expires or
someone calls the standalone `browserscale.StopBrowser(apiKey, sessionId)`.

That gives you the second persistence axis: you can pause your code,
restart your process, and reattach to the same browser without ever
losing cookies, navigation state, or fingerprint.

## Fingerprint persistency

This is where browserscale differs from spinning up a fresh headless Chrome
every run. Each rental is provisioned with a **fingerprint** — the
machine the browser presents: user agent, platform, cores and memory,
the GPU it reports, screen size, language and timezone — and that
fingerprint has a **stable id** you can hold on to and replay.

Rendering itself is never faked. Sessions run on real consumer GPUs,
so canvas, WebGL and audio output is what that hardware actually
produces; there is no spoofing layer for a deeper check to unmask.

### Default: server picks per country

If you don't pass `WithFingerprint` / `withFingerprint`, the server
picks one for you based on the rental's country code. Different
rentals → different fingerprints, every run.

### Pinning: read it, store it, send it back

After every rental you can read which fingerprint the session is
wearing:

**Go:**

```go
browser, _ := browserscale.RentBrowser(ctx, cfg)
fmt.Println("fingerprint id:", browser.Fingerprint())
```

**TypeScript:**

```ts
const browser = await rentBrowser(cfg);
console.log("fingerprint id:", browser.getFingerprint());
```

Hold on to that string and feed it back into the next rental to get
the exact same browser identity again:

**Go:**

```go
cfg := browserscale.NewBrowserConfig(apiKey, 600, "", 0, "", "").
    WithCountryCode("DE").
    WithFingerprint(savedFingerprint)
browser, _ := browserscale.RentBrowser(ctx, cfg)
```

**TypeScript:**

```ts
const cfg = new BrowserConfig(apiKey, 600, "", 0, "", "")
    .withCountryCode("DE")
    .withFingerprint(savedFingerprint);
const browser = await rentBrowser(cfg);
```

What "same identity" gets you in practice:

- **UA string** stays identical (Chrome version, OS, build).
- **`navigator.*`** properties match (platform, hardwareConcurrency,
  deviceMemory, plugins, etc.) — in the main document, in every frame
  and in every worker.
- **The reported GPU** stays the same.
- **Screen + viewport defaults** stay the same.
- **Locale + timezone** carry across when you keep the same
  `WithCountryCode` / `WithTimezone`.

To the target site, that's a returning user, not a new browser
install on the same account.

## Stable identity across runs — putting it together

Cookies + storage + fingerprint + matching geography is the full "this
is the same user" combo. The first run captures everything; later runs
restore it:

**Go:**

```go
// ── First run ──
cfg := browserscale.NewBrowserConfig(apiKey, 600, "", 0, "", "").
    WithCountryCode("DE")
browser, _ := browserscale.RentBrowser(ctx, cfg)

_, _ = browser.Navigate(ctx, "https://example.com/login", 0)
// … perform login …

cookies, _ := browser.GetCookies(ctx)
storage, _ := browser.GetStorage(ctx, "")
identity := struct {
    Fingerprint string                    \`json:"fingerprint"\`
    Country     string                    \`json:"country"\`
    Cookies     []browserscale.CookieParam         \`json:"cookies"\`
    Storage     []browserscale.StorageOriginEntry  \`json:"storage"\`
}{
    Fingerprint: browser.Fingerprint(),
    Country:     browser.CountryCode(),
    Cookies:     cookies,
    Storage:     storage,
}
buf, _ := json.Marshal(identity)
_ = os.WriteFile("identity.json", buf, 0o600)
_ = browser.Close()

// ── Later run (same or different process) ──
buf, _ := os.ReadFile("identity.json")
var saved struct {
    Fingerprint string
    Country     string
    Cookies     []browserscale.CookieParam
    Storage     []browserscale.StorageOriginEntry
}
_ = json.Unmarshal(buf, &saved)

cfg = browserscale.NewBrowserConfig(apiKey, 600, "", 0, "", "").
    WithCountryCode(saved.Country).
    WithFingerprint(saved.Fingerprint)
browser, _ = browserscale.RentBrowser(ctx, cfg)
_ = browser.SetCookies(ctx, saved.Cookies)
_ = browser.SetStorage(ctx, saved.Storage)
_, _ = browser.Navigate(ctx, "https://example.com/dashboard", 0)
```

**TypeScript:**

```ts
// ── First run ──
const cfg = new BrowserConfig(apiKey, 600, "", 0, "", "")
    .withCountryCode("DE");
let browser = await rentBrowser(cfg);

await browser.navigate("https://example.com/login");
// … perform login …

const identity = {
    fingerprint: browser.getFingerprint(),
    country:     "DE",
    cookies:     await browser.getCookies(),
    storage:     await browser.getStorage(),
};
await fs.writeFile("identity.json", JSON.stringify(identity), { mode: 0o600 });
await browser.stopBrowser();

// ── Later run ──
const saved = JSON.parse(await fs.readFile("identity.json", "utf8"));
const reCfg = new BrowserConfig(apiKey, 600, "", 0, "", "")
    .withCountryCode(saved.country)
    .withFingerprint(saved.fingerprint);
browser = await rentBrowser(reCfg);
await browser.setCookies(saved.cookies);
await browser.setStorage(saved.storage);
await browser.navigate("https://example.com/dashboard");
```

If you're routing through your own proxy, persist
`(proxyHost, proxyPort, proxyUsername, proxyPassword)` the same way
— a stable exit-IP region on top of stable cookies and a stable
fingerprint is the cleanest "returning user" signal you can send. If
the flow signs in to the browser itself, add the
[auth session](#a-signed-in-login-as-one-object) to the same file and
import it before the first navigation.

## Reattaching to a live session

For long-running flows that outlive your process — overnight crawls,
human-in-the-loop pauses, reconnecting after a script crash — store
the `sessionId` and `grpcUrl` and reattach later:

**Go:**

```go
// First process: rent, persist the address.
browser, _ := browserscale.RentBrowser(ctx, cfg)
_ = os.WriteFile("session.json", []byte(fmt.Sprintf(
    \`{"sessionId":%q,"grpcUrl":%q}\`,
    browser.SessionId(), browser.GrpcUrl(),
)), 0o600)
// Do NOT call Close — the rental should keep running.

// Later process: reattach to the same session.
buf, _ := os.ReadFile("session.json")
var s struct{ SessionId, GrpcUrl string }
_ = json.Unmarshal(buf, &s)
browser, _ = browserscale.ConnectSession(ctx, s.GrpcUrl, apiKey, s.SessionId)
// browser drives the same context as before — same cookies, same fingerprint,
// same open page.
```

**TypeScript:**

```ts
// First process.
const browser = await rentBrowser(cfg);
await fs.writeFile("session.json", JSON.stringify({
    sessionId: browser.getSessionId(),
}), { mode: 0o600 });
// Do NOT call stopBrowser — keep the rental alive.

// Later process. The TS SDK doesn't expose the rented transport URL
// on the CloudBrowser handle, so you need to know your WebSocket URL
// out-of-band (typically a known cluster endpoint).
const s = JSON.parse(await fs.readFile("session.json", "utf8"));
const browser2 = createWebSocketBrowser(WS_URL, s.sessionId, apiKey);
// browser2 drives the same context as before.
```

A reattached handle is a thin wrapper: closing it only closes the
transport, not the rental. To actually release the rental from a
process that only has the id, use the standalone
`browserscale.StopBrowser(ctx, apiKey, sessionId)` (Go) /
`stopBrowser(apiKey, sessionId)` (TS).

## Gotchas

- **Cookies are upserted, not appended.** `SetCookies` with an
  existing `(name, domain, path)` tuple replaces the value — handy
  for refreshing a session cookie, surprising if you expected a
  duplicate entry. There is no per-cookie delete; clear the whole
  jar with `ClearCookies` if you need to start over.
- **Open pages don't see `SetStorage` writes.** Storage is written
  directly in the browser process; a page that was already open keeps
  serving its cached values until it reloads. Write first, navigate
  after — the recipes above do it in that order for a reason.
- **`sessionStorage` is out of scope.** `GetStorage` / `SetStorage` /
  `ClearStorage` cover first-party localStorage only; sessionStorage
  is per-tab and is not captured or replayed.
- **Fingerprint ids are server-side handles.** They're meaningful
  as long as the server still has that fingerprint in its catalog.
  If you persist an id for months and later get a "fingerprint not
  found" error on rental, fall back to letting the server pick a new
  one and persist the new id.
- **Don't mix and match identity signals.** Replaying cookies and
  a US fingerprint while exiting through a DE IP is a louder signal
  than running with all-defaults. If you persist one of the three
  (fingerprint / cookies / IP region), persist all three.
- **Reattaching is not the same as resuming a stopped session.**
  `ConnectSession` / `createWebSocketBrowser` assumes the rental is
  still alive on the server. Once the rent duration expires or
  someone explicitly calls `StopBrowser`, the context is gone — its
  cookies, fingerprint and open pages disappear with it.
- **Closing an attached handle doesn't release the rental.** That's
  intentional — multiple processes can attach to the same session.
  Use the standalone `StopBrowser(ctx, apiKey, sessionId)` when you
  do want to release from outside the renting process.

## See also

- [Core concepts](/docs/concepts) — the rent → drive → stop lifecycle.
- [Quickstart](/docs/quickstart) — `BrowserConfig` parameters in context.
- API reference: [Go `GetCookies` / `GetStorage` / `RentBrowser`](/docs/api-reference/go#GetCookies) · [TS `getCookies` / `getStorage` / `rentBrowser`](/docs/api-reference/ts#getCookies).
- Auth sessions: [Go `GetAuthSession`](/docs/api-reference/go#GetAuthSession) · [TS `getAuthSession`](/docs/api-reference/ts#getAuthSession) and the [`AuthSession`](/docs/api-reference/go#AuthSession) type.

→ Continue: [Captchas](/docs/guides/captchas)
