<!--
  url: https://browserscale.cloud/docs/guides/agentic-coding
  title: Agentic coding
  description: Let a coding agent build your automation: explore with the hosted MCP server, scaffold a runnable Go module with browserscale init, let the agent write the flow.
-->

# Agentic coding

There are two ways to put an AI coding agent to work on browserscale, and
they are meant to be used together. The **MCP server** hands the agent a
live cloud browser it can look at and poke at, one call at a time. The
**CLI** hands it a project to write into — a Go module that already
compiles, already runs workers in parallel and already knows how to rent a
browser.

Used on its own, MCP makes the agent a pair of hands: it clicks for you,
you pay tokens for every step, and when the conversation ends nothing is
left. Used together with the CLI, the agent explores with MCP and then
writes what it learned into code that runs without a model in the loop.
That is the workflow this page describes.

> **TL;DR**
>
> - **MCP endpoint:** `https://mcp.browserscale.cloud/mcp` (Streamable HTTP). Authenticate with an `Authorization: Bearer <your-api-key>` header.
> - `https://mcp.browserscale.cloud/` — without `/mcp` — is a human-readable list of the tools. **It is not the endpoint**; pointing a client at it will not work.
> - The API key belongs in your client config as a header. It is never a tool argument, so it never enters the model's context.
> - **CLI:** `npm i -g browserscale` (or `go install github.com/browserscale/browserscale@latest`), then `browserscale init` scaffolds a runnable Go module.
> - Open the generated folder in Cursor, Codex, Claude or any other coding agent: it ships an `AGENTS.md` and the whole SDK reference offline, so the agent works from real signatures instead of guessing.
> - `browserscale dev` rebuilds, restarts and streams the logs from one command — that is the loop the agent iterates in.

## The MCP server

browserscale runs a hosted MCP server, so there is nothing to install and
nothing to keep running.

| | |
| --- | --- |
| Endpoint | `https://mcp.browserscale.cloud/mcp` |
| Transport | Streamable HTTP (a remote MCP server, not stdio) |
| Auth | `Authorization: Bearer <your-api-key>` |
| Tool overview | [mcp.browserscale.cloud](https://mcp.browserscale.cloud/) — for humans, not for clients |

### Connecting a client

Nearly every MCP client wants the same three things: a name, the URL and a
map of headers. The file it lives in and the exact field names differ per
client, so check yours — but the shape is this:

```json
{
  "mcpServers": {
    "browserscale": {
      "url": "https://mcp.browserscale.cloud/mcp",
      "headers": {
        "Authorization": "Bearer sk_your_api_key"
      }
    }
  }
}
```

Two things worth being precise about:

- **Use the `/mcp` path.** The bare host serves a page that lists every
  tool with its full description. That page exists so you (or an agent
  reading the web) can see what the server offers without connecting. A
  client pointed at `https://mcp.browserscale.cloud/` will not speak MCP.
- **The key travels as a header, never as an argument.** No tool takes an
  `apiKey` parameter; the server reads it off the request. That is what
  keeps your key out of the model's context window and out of the
  conversation log.

### How sessions work

The server is stateless — it holds no session for you between calls, which
is why the session id travels with every call:

1. `rent` starts a browser and returns a `sessionId` and a `grpcUrl`.
2. Every action tool takes that `sessionId` (and the `grpcUrl` when your
   client is not pointed at a default one).
3. `stop` releases the browser. It needs only the `sessionId`.

Sessions also expire on their own after `rentSeconds`, so a forgotten one
does not bill forever — but calling `stop` when the flow is done frees the
browser and its proxy immediately.

Nothing says the session has to have been rented over MCP. Any session on
your account can be driven this way, which is what makes **live debugging**
work: log `sessionId` and `grpcUrl` from your code, and the agent can attach
to a session your program is running right now — or to the half-finished
state a failed run left behind — and inspect or drive the real page instead
of reconstructing what happened from a stack trace.

### What it is good at

Use MCP for the parts where looking beats guessing:

- Walking an unfamiliar flow once, by hand, before writing any code.
- Proving a selector resolves to exactly one element — `evaluate` the
  expression you are about to bake into `click`, and check the count.
- Attaching to a live run to see what the code is actually looking at, and
  taking a `screenshot` when the logs do not explain it.
- Reading the page with `observe`, which is the same view
  [`GetObservation`](/docs/guides/reading) returns in the SDK.

Every tool maps one-to-one onto an SDK method, so a call that worked over
MCP is already a line of the script you are about to write. What MCP is
*not* good for is running the work: that is what the CLI part is for.

## The CLI

`browserscale` is a scaffolder and a dev loop. It writes the boilerplate
every browser bot needs before the interesting part starts, so your agent
begins from a project that runs rather than an empty folder.

### Install

```bash
npm i -g browserscale
```

Or with a Go toolchain (1.25 or newer), which puts the binary in your
`GOPATH/bin`:

```bash
go install github.com/browserscale/browserscale@latest
```

### Scaffold a module with `init`

```bash
browserscale init
```

Run without flags, it asks a few questions and every answer has a default,
so you can hold enter:

| Question | Default | What it decides |
| --- | --- | --- |
| Module name | `my-module` | The name, and the directory it is written to. |
| Go module path | the name | What goes into `go.mod`. |
| Lifecycle | `one-shot` | The shape of the run loop (see below). |
| Parallelism | `configurable` | Whether worker count is fixed, single, or a runtime flag. |
| Workers | `4` | Only asked when parallelism is `fixed`. |

The lifecycle is the one answer worth thinking about, because it decides
what `run.go` looks like:

| Lifecycle | The generated loop |
| --- | --- |
| `one-shot` | Runs once, then exits. |
| `queue` | Drains an input list across workers — the usual choice for a list of accounts or URLs. |
| `repeat` | Runs until N successes, retrying failures. |
| `continuous` | Loops forever until you stop it. |
| `scheduled` | Runs on a fixed interval. |
| `custom` | A minimal `Run()` you write yourself. |

Parallelism is `single` (one worker), `fixed` (a number baked in now) or
`configurable` (a `-threads` flag at runtime, which is the default).

To skip the wizard entirely — handy when an agent runs the command — pass
the answers as flags and add `-yes`:

```bash
browserscale init -name orders-bot -kind queue -threading fixed -fixed-threads 8 -yes
```

The target directory has to be empty or not exist yet, so `init` can never
overwrite work you already have.

### What you get

```text
orders-bot/
  main.go             hands off to the kit's harness
  module.go           config schema + worker count
  flow.go             thin dispatcher: picks which flow to run
  register.go         a complete, working browser flow
  run.go              the lifecycle loop you chose
  rent.go             proxy pool + rent-with-backoff
  browserscale.yaml   your init answers, in writing
  AGENTS.md           the agent's guide to this project
  README.md
  docs/               the whole SDK reference, offline
  data/proxies.txt    one proxy per line
```

A few of these matter more than the others:

- **`register.go`** (or `task.go`, depending on lifecycle) holds a complete
  worked flow against the browserscale playground. It runs before you change
  anything, which means your agent has a working example to pattern-match
  against rather than a blank function.
- **`AGENTS.md`** is a written playbook: the file layout, the SDK and kit
  APIs with copy-paste snippets, proven patterns, and an explicit list of
  things not to do (no `el.click()` through `evaluate`, no inventing
  methods that do not exist).
- **`docs/`** is this documentation, bundled as Markdown. The agent greps
  it instead of guessing an API — which is the single biggest difference
  between a flow that works and one that hallucinates half a method name.
- **`rent.go`** already retries the errors worth retrying, so a busy pool
  or a brief credit dip does not kill the run.

The generated module pulls in
[`browserscale-go`](https://github.com/browserscale/browserscale-go) for the
browser and `browserscale-kit` for everything around it: the worker
harness, a config form, work queues, proxy pools, a SQLite store for
de-duplication and resume, an IMAP fetcher for one-time codes, and a
logger. Neither version is pinned — `go mod tidy` resolves both to their
latest release.

### Config

You don't have to get the project running before handing it over. The
config schema is declared in `module.go`, and the module ships its own
configurator that saves the values to `data/config.json`:

```bash
cd orders-bot
go mod tidy
go run .
```

Your agent can drive that, and extend the schema when the flow needs
another field. Two values can only come from you, because nobody else has
them: your **API key**, and at least one entry in `data/proxies.txt` —
every session is proxied, and the exit IP, the geolocation and the timezone
all follow from it.

### Open it in your agent

```bash
cursor orders-bot
codex orders-bot
claude
```

Describe the job in plain language — "log in with every account in
`data/accounts.csv` and export each order to a file" — and let it work. The
agent has a running example, the offline reference and `AGENTS.md`, which
covers the file layout, the SDK and kit APIs, and how to iterate. With an
MCP client connected it can also look at the real page while it writes, and
attach to its own runs when something doesn't hold.

### Iterate with `dev`

```bash
browserscale dev
```

One command builds the module, kills the previous run, starts the new
binary and streams stdout and stderr until you press Ctrl+C. Run it again
and it rebuilds and restarts. This is what you want an agent using: it can
start its own run and read what actually happened, instead of hand-rolling
a build-and-kill loop mid-session.

| Flag | What it does |
| --- | --- |
| `-dir <path>` | The module directory. Defaults to the current one. |
| `-run <name>` | Write to a fixed `runs/<name>/` folder instead of a timestamped one. |

`dev` starts the module non-interactively and headless, so it runs against
whatever is saved in `data/config.json` rather than opening the
configurator.

## Gotchas

- **`https://mcp.browserscale.cloud/` is not the endpoint.** The endpoint is
  `https://mcp.browserscale.cloud/mcp`. The bare host is a human-readable
  tool overview.
- **`rent` needs a proxy.** The exit IP comes from the upstream proxy you
  pass, and country and timezone are derived from it server-side — so leave
  `countryCode` and `timezone` empty unless you deliberately want to
  override what the proxy implies.
- **`stop` what you `rent`.** Sessions expire after `rentSeconds`, but an
  abandoned one holds a browser and a proxy until then.
- **`init` refuses a non-empty directory.** That is deliberate; point `-dir`
  somewhere new.
- **`dev` runs against the saved config.** It starts the module
  non-interactively, so `data/config.json` has to exist — the configurator
  (`go run .`) is what writes it.
- **The CLI does not configure MCP for you.** The MCP server is set up in
  your coding agent's own config; the API key your *module* uses lives in
  `data/config.json`. They are two separate places, and CLI-side auth and
  MCP wiring are still on the roadmap.
- **Attaching needs the `grpcUrl`, not just the id.** Log both from your code
  if you want the agent to reach a running session; the id alone is only
  enough for `stop`.

## See also

- [Reading the page](/docs/guides/reading) — `GetObservation`, the SDK method behind the `observe` tool.
- [Targeting elements](/docs/guides/locators) — how to pick a target that still works tomorrow, which is what you want the agent baking into code.
- [Cookies, storage & sessions](/docs/guides/cookies) — carrying a logged-in identity between runs.
- [mcp.browserscale.cloud](https://mcp.browserscale.cloud/) — the full tool list with descriptions.
- [browserscale on GitHub](https://github.com/browserscale/browserscale) — source and issues.

→ Continue: [API Reference](/docs/api-reference)
