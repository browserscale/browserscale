<!--
  url: https://browserscale.cloud/docs/guides/captchas
  title: Captchas
  description: Passive anti-bot challenges, interactive captchas, and the SolveCaptcha integration for the ones that need a human solver.
-->

# Captchas

Bot defences on the open web come in two flavours: **passive** checks
that score every visitor in the background, and **interactive**
challenges that ask the user to do something (check a box, drag a
slider, pick the bicycles). browserscale handles those two cases very
differently, and getting that distinction right saves you a lot of
unnecessary code.

> **TL;DR**
>
> - **Passive challenges and anti-bot scoring** — browserscale already passes these without any code on your side. Don't call `SolveCaptcha` for them; there is nothing to solve.
> - **Interactive challenges** — that's what `SolveCaptcha` is for. It detects, solves, and wires the result back into the page server-side.
> - `SolveCaptcha(timeoutMs, retryAmount)` (Go) / `solveCaptcha({ timeoutMs, retryAmount })` (TS). Both default to `0` → server default (60 s budget, no retries).
> - Returns an empty string on success — the solution is applied in-page, no follow-up call needed.
> - Supports the common interactive challenge types you run into in the wild. The exact list moves over time; ping us on [Discord](https://discord.gg/wrc) for the current set.
> - `retryAmount` does **not** apply to every challenge type — some invalidate the page after a failed attempt and a retry is impossible.

## Passive vs interactive — when *not* to call `SolveCaptcha`

The thing most users get wrong on day one is calling `SolveCaptcha`
defensively on every page load "just in case". You don't need to.

**Passive challenges** are everything the page does behind the scenes
to decide whether you're a bot: fingerprint analysis, behavioural
scoring, automation-flag detection (`navigator.webdriver`,
CDP-leak probes), TLS/JA3 fingerprinting, headless-browser tells,
proxy-quality checks. There is nothing on screen for you to interact
with — the page just decides whether to serve you content or block
you. browserscale's native pipeline (real browsers, real fingerprints, real
residential-grade exit IPs, no CDP-attached automation) is already
designed to pass these. **No code required.** If you find a site
that blocks browserscale on passive checks alone, that's a bug we want to
hear about — open a ticket or message us on
[Discord](https://discord.gg/wrc).

**Interactive challenges** are the ones you've seen: a checkbox
("I'm not a robot"), an image grid, a slider puzzle, a rotating
image to align, an audio transcription fallback. The page has put a
widget on screen and is waiting for a human (or a solver) to feed it
back a token. That's the case `SolveCaptcha` is built for.

The decision rule is simple:

| You see… | Call `SolveCaptcha`? |
| --- | --- |
| Page loads normally, content is there. | No — passive check already passed. |
| Page is blocked or redirected to "verify you're human" but with no widget. | No — usually a passive check browserscale will pass; check your country/fingerprint config first. |
| A visible challenge widget (checkbox, image grid, slider, etc.). | Yes — call `SolveCaptcha`. |
| You're not sure. | Try without first. Add `SolveCaptcha` only if the flow doesn't progress. |

## The basic call

`SolveCaptcha` scans the page for the first supported interactive
challenge, solves it server-side, wires the result back into the
page, and returns. Most challenges don't surface a token to the
caller — the empty string back means "done, keep going."

**Go:**

```go
if _, err := browser.SolveCaptcha(ctx, 0, 2); err != nil {
    log.Fatal(err)
}
// Captcha solved; the page is ready to proceed.
```

**TypeScript:**

```ts
await browser.solveCaptcha({ retryAmount: 2 });
// Captcha solved; the page is ready to proceed.
```

Two parameters, both optional:

| Parameter | Default | What it does |
| --- | --- | --- |
| `timeoutMs` | `0` → server default of 60 000 ms | How long to wait for a supported challenge to appear *and* be solved. Passing `0` is fine for most cases; raise it for slow-loading sites. |
| `retryAmount` | `0` (no retries) | If the first solve attempt fails, retry this many times before giving up. **Only meaningful for challenge types that can be retried** (see below). |

### What you get back

The return value is an empty string on success and on failure — the
*string* doesn't tell you anything. Check the `err` (Go) or rely on
the rejected promise (TS) to detect failure. For most challenge
types the solution is invisibly wired into the page (form field,
cookie, JS variable), and the next interaction just works.

A non-empty string is returned only for the small subset of
challenges that surface a token you might want to forward elsewhere
(e.g. into a separate API call). Most callers can ignore the return
value entirely.

## Retries, and why they don't apply everywhere

`retryAmount` controls how many times the *solve* step is repeated
on failure, not how many times the *detect* step is repeated. The
useful distinction is:

- **Retriable challenges** — the widget stays on the page after a
  failed solve, so another attempt can be made against the same
  instance. Set `retryAmount` to `1`–`3`.
- **One-shot challenges** — the widget invalidates itself after a
  failed attempt (rotates the puzzle, refreshes the token, or
  outright blocks further attempts). For these, `retryAmount` is
  effectively ignored — you get one shot, period.

There isn't a clean way to know in advance which bucket a given
challenge falls into; what we can tell you is that **`retryAmount`
is a maximum, not a guarantee**. Treat any specific value above zero
as best-effort. If a flow keeps failing on retries, drop the retry
count back to `0`, let `SolveCaptcha` error cleanly, and handle
the retry at the script level (re-navigate, fresh attempt).

## Which challenges are supported?

`SolveCaptcha` covers the common interactive challenge types you run
into in the wild. We deliberately don't enumerate the supported
providers here — the list shifts as both sides evolve, and the most
accurate answer is always the current one. **For the up-to-date
list, ping us on [Discord](https://discord.gg/wrc).** If you've got
a site that uses a challenge we don't yet support, send us the URL
and we'll usually have it queued within a release or two.

## Gotchas

- **Don't call `SolveCaptcha` defensively.** If the page loads and
  your interactions work, there's nothing to solve. Adding a call
  "just in case" only costs you 60 s of timeout per page.
- **An empty result string is not an error.** Both success and
  failure return `""` — only the error tells you whether it worked.
- **`retryAmount` is a best-effort cap.** One-shot challenges
  invalidate themselves after a failed attempt and retries can't
  recover. Handle retry at the script level (re-navigate, try
  again) for those.
- **Passive failure looks different.** If a site silently shows you
  a "we couldn't verify you" page with no widget, `SolveCaptcha`
  won't help — there's no challenge to solve. The fix is upstream:
  check your country code and proxy quality.
- **A solve can take real time.** Interactive challenges aren't
  instant — image-grid puzzles in particular can take 10–30 s
  server-side. Account for that in any per-action timeout you've
  configured elsewhere in your flow.
- **The session keeps running on failure.** A failed `SolveCaptcha`
  returns an error but doesn't tear down the session — you can
  re-navigate, call it again with different parameters, or fall
  back to a manual workflow.

## See also

- [Loading pages](/docs/guides/loading) — first action you'll typically take before evaluating whether a challenge is on screen.
- [Discord](https://discord.gg/wrc) — for the current list of supported challenge types.
- API reference: [Go `SolveCaptcha`](/docs/api-reference/go#SolveCaptcha) · [TS `solveCaptcha`](/docs/api-reference/ts#solveCaptcha).

→ Continue: [Shadow DOM & canvas](/docs/guides/shadow-canvas)
