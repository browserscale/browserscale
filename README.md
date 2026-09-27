<div align="center">

# browserscale

**The command line for [browserscale](https://browserscale.cloud) — one binary for working with the platform from your terminal.**

Two halves. `init` writes a complete Go project wired to the [SDK](https://github.com/browserscale/browserscale-go) and the [kit](https://github.com/browserscale/browserscale-kit), and `dev` builds, restarts and streams it — both work on a directory. `list`, `rent`, `run`, `view` and `stop` work on the cloud browsers your API key is paying for. `browserscale help` is always the truth about what your binary can do.

[![Go Reference](https://pkg.go.dev/badge/github.com/browserscale/browserscale.svg)](https://pkg.go.dev/github.com/browserscale/browserscale)
![Go](https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue)

[Install](#install) · [Commands](#commands) · [`init`](#init) · [`dev`](#dev) · [Your browsers](#your-browsers) · [Agentic coding](#agentic-coding)

</div>

---

## Install

Every route installs the same single static binary — pick whichever fits your setup. Linux, macOS and Windows are all supported.

### npm

```bash
npm i -g browserscale
```

No Go toolchain needed. The package is a small launcher around the binary, which ships as one prebuilt package per platform, so there is no postinstall script and nothing is downloaded at install time. Works with `npm ci --ignore-scripts` and offline caches. Node 18+.

### Go

```bash
go install github.com/browserscale/browserscale@latest
```

Builds from source, needs **Go 1.25+**. Pin a version with `@v0.3.0` instead of `@latest`. The binary lands in your `GOPATH/bin`.

### Homebrew (macOS)

```bash
brew tap browserscale/tap
brew install browserscale
```

The tap is a one-time step; after that `brew install browserscale` and `brew upgrade browserscale` are the whole story.

This route is macOS-only, which is about Homebrew and not about the CLI: the tap publishes a *cask*, and Homebrew refuses casks on Linux. On Linux, install with npm or Go — same binary, same `browserscale` command, nothing missing.

## Commands

The commands come in two groups, and the split is about what a mistake costs. The first two act on the directory you are standing in: no key, no network, and a wrong answer costs a rebuild. The rest act on rented browsers — they need an API key, reach a machine elsewhere, and a wrong session id can throw away work you are paying for.

**Your project**

| Command | What it does |
| --- | --- |
| [`init`](#init) | Scaffold a new automation module. |
| [`dev`](#dev) | Build, restart, and stream a module's logs. |

**Your browsers**

| Command | What it does |
| --- | --- |
| [`login`](#login) | Save an API key so the commands below need no flag. |
| [`list`](#list) | Show the sessions this key is paying for. |
| [`rent`](#rent) | Rent a session and leave it running. |
| [`run`](#run) | Run a script inside a session. |
| [`view`](#view) | Watch a session's screen, and drive it by hand. |
| [`stop`](#stop) | End a session. |
| [`runs`](#runs) | Work with the scripts inside a session. |

**Other**

| Command | What it does |
| --- | --- |
| `version` | Print the CLI version. |
| `help` | Show the command list. |

Every command takes `-h` for its own flags, and every browser command takes `-json`. One section per command follows, each self-contained.

### `init`

Every browserscale automation starts with the same setup: a `go.mod` pinned to the SDK and kit, a `main()` that hands off to the harness, a config schema, and a rent-with-backoff loop, all *before* you get to the actual browser flow. `init` writes that scaffolding for you, so the generated folder **compiles and runs as-is** against the playground and is ready to drop straight into an agentic coding tool.

`flow.go` stays a thin dispatcher; the worked example lives in a named flow file (`register.go` or `task.go`). Add further flows as sibling files (`enter.go`, `login.go`, …) — one flow per file. The generated `AGENTS.md` plus an offline `docs/` tree mean an agent reads real API signatures instead of guessing — less vibe-coding, fewer wrong-API round-trips.

> **The workflow:** run `init`, open the generated folder in your coding agent, and describe your task. It has the docs, the structure, and a working example to pattern-match against.

Run it with no flags for an interactive wizard:

```bash
browserscale init
```

```text
browserscale init — answer a few questions (enter = default)

Module name .............. example_module
Go module path ........... github.com/me/example_module
Lifecycle ................ one-shot | queue | repeat | continuous | scheduled | custom
Parallelism .............. single | configurable | fixed
```

…or drive it fully non-interactively (ideal for CI and agents) with `-yes`:

```bash
browserscale init -name example_module -kind queue -threading configurable -yes
```

**Flags**

| Flag | Default | Description |
| --- | --- | --- |
| `-name` | — | Module name (and target directory). Required. |
| `-module` | `<name>` | Go module path written into `go.mod`. |
| `-kind` | `one-shot` | Lifecycle: `one-shot` \| `queue` \| `repeat` \| `continuous` \| `scheduled` \| `custom`. |
| `-threading` | `configurable` | Parallelism: `single` \| `fixed` \| `configurable`. |
| `-fixed-threads` | `4` | Worker count when `-threading=fixed`. |
| `-dir` | `./<name>` | Target directory. |
| `-yes`, `-y` | `false` | Accept defaults, skip the wizard. |

The generated `go.mod` pins no versions; the first `go mod tidy` (step one below) resolves **browserscale-go** and **browserscale-kit** to their latest published releases.

#### What it generates

A single-module Go project, formatted and ready to run:

```text
example_module/
├── go.mod                 # resolves latest browserscale-go + browserscale-kit on `go mod tidy`
├── main.go                # hands off to harness.Run — you rarely touch this
├── module.go              # module identity + config Schema (form DSL)
├── flow.go                # thin doTask — dispatches to a named flow
├── register.go            # (queue / one-shot) YOUR browser flow — worked playground signup
│   or task.go             # (other kinds) YOUR browser flow — worked playground example
├── run.go                 # the lifecycle loop (see kinds below) — yours to edit
├── rent.go                # rent-with-backoff loop (INVALID_API_KEY / NO_CAPACITY / …)
├── AGENTS.md              # SDK + kit reference and best-practices for a code agent
├── README.md              # per-module readme
├── browserscale.yaml      # the init answers, verbatim (the module manifest)
├── .gitignore
├── data/
│   ├── proxies.txt        # sample proxy list (every run is proxied)
│   └── accounts.csv       # sample input (queue kind only)
├── runs/                  # created at runtime — per-run outputs
└── docs/                  # the FULL browserscale docs, offline
    ├── introduction.md  ·  quickstart.md  ·  concepts.md
    ├── guides/           # locators, waiting, loading, interaction, reading, evaluation,
    │                     # frames, network, cookies, captchas, shadow-canvas, agentic-coding
    └── api-reference/go.md
```

Config is a single `data/config.json` (written by the configurator). Input files
sit flat under `data/`; outputs go to `runs/<name>/`; the KV store is
`data/store.db`.

The named flow file ships as a **complete, working example** against the playground — navigate → wait → fill → click → branch on the outcome — so you can run the module immediately and then replace that body. `flow.go` stays the switchboard:

```go
// flow.go
func (m *Module) doTask(ctx context.Context, browser *browserscale.CloudBrowser) error {
	return m.register(ctx, browser) // or m.enter / m.login as the module grows
}
```

Add further flows as sibling files (`enter.go`, `login.go`, …) — one named flow per file.

Run it:

```bash
cd example_module
go mod tidy
go run .              # once: configurator → data/config.json
browserscale dev      # iterate: rebuild → restart → stream logs
browserscale dev -run dig
```

#### What the agent reads

Two of the generated files exist purely to make an agent productive from the first prompt:

- **`AGENTS.md`** — a focused SDK + kit reference: the thin-`doTask` / one-file-per-flow layout, the canonical *navigate → wait → act → branch* rhythm, how to use `JS()` locators instead of hand-rolling `clickByText`, multi-condition `Wait`, `Fill` vs `InsertText`, the `browserscale-kit` helpers (store, proxy, input, mail with the `mailTime` idiom), and — when connected — the optional **browserscale MCP server** for probing a live session.
- **`docs/`** — the entire browserscale documentation, **bundled offline**, so the agent opens the exact page (e.g. `docs/guides/interaction.md`) or greps for a method (`grep -rin waitforany docs/`) instead of guessing an API or making a network call. The bundle is mirrored from the SDK docs pipeline and kept in sync by CI.

The result: less "vibe coding", fewer wrong-API round-trips, and a correct project structure from the start.

#### Lifecycle kinds

`-kind` shapes the generated `run.go` — the loop that calls your `doTask`. Pick the rhythm your task needs; you own the file afterwards.

| Kind | Behavior |
| --- | --- |
| `one-shot` | Run `doTask` once, then exit. |
| `queue` | Drain an input list (CSV) across workers — one record each. Ships `register.go` + sample `accounts.csv`. |
| `repeat` | Run until *N* successes (a `nil` return counts), then stop. |
| `continuous` | Workers loop forever until interrupted; errors are logged and the worker loops again. |
| `scheduled` | Run on a fixed interval; errors are logged and the next tick still fires. |
| `custom` | A minimal `Run()` — you write the whole flow yourself. |

#### Threading modes

`-threading` controls how many workers run concurrently:

| Mode | Behavior |
| --- | --- |
| `single` | Exactly one worker. |
| `configurable` | Worker count exposed as a runtime option (the default). |
| `fixed` | A set worker count baked in via `-fixed-threads` (default 4). |

### `dev`

From inside a generated module (or with `-dir`). Always headless — needs `data/config.json`, which `go run .` creates once through the configurator:

```bash
browserscale dev
```

This checks that `go` is on `PATH`, stops any previous `dev` child for that directory, runs `go build` into `.browserscale/`, starts the binary with `-yes` (no TUI), and streams stdout/stderr. Re-run after edits to rebuild and restart; Ctrl+C stops the child.

**Flags**

| Flag | Default | Description |
| --- | --- | --- |
| `-dir` | `.` | Module directory. |
| `-run` | timestamp | Write to a fixed `runs/<name>/` instead of `runs/<timestamp>/`. |

## Your browsers

The commands below act on the sessions your API key holds. They share two things.

**Where the key comes from**, most explicit first:

1. `-key` on the command
2. `BROWSERSCALE_API_KEY` in the environment
3. the key [`login`](#login) saved
4. `cloudKey` in `./data/config.json`, when you are inside a module

The last one is why these commands work with no setup at all in a project directory. It is also why anything stated explicitly has to beat it — a module's key is the right one for that project and the wrong one as soon as you have two. Reading a module's config never writes to it.

**`-json`**, on every one of them, for scripting. `list -json` gives you the session objects, `run -json` gives you one document with the return value and the whole log.

Point the CLI at a private deployment with `BROWSERSCALE_API_URL`, or `login -endpoint` to make it stick.

### `login`

```bash
browserscale login                    # paste the key when prompted
browserscale login -key sk_...
browserscale login < key.txt
browserscale login -show              # which key is in use, and from where
```

Writes the key to this machine's config (`%AppData%\browserscale\config.json` on Windows, `$XDG_CONFIG_HOME/browserscale/config.json` elsewhere) with owner-only permissions. The command sends nothing anywhere — it does not verify the key, so the first real check is your next `list`.

If `BROWSERSCALE_API_KEY` is set it will keep winning, and `login` says so rather than leaving you to wonder why the wrong account answers.

### `list`

```bash
browserscale list
```

```text
SESSION                               COUNTRY  AGE     REMAINING  EGRESS          PROXY
5f3c8a12-...                          us       12m3s   47m57s     203.0.113.7     -
9b71e0d4-...                          de       2m11s   unlimited  198.51.100.22   proxy.example.com
```

Every other command here takes a session id, and this is where one comes from. That matters more than it sounds: an id is otherwise known only to the process that rented the session, so a script that crashed between renting and stopping leaves a browser running that nothing can name — and that you keep paying for.

### `rent`

```bash
browserscale rent -duration 10m -country us
```

Prints the session id on stdout and leaves the browser running. For trying something by hand: rent one, drive it with [`run`](#run), stop it when done. Production code rents through the SDK instead, so a session's lifetime is tied to the program that needs it.

With no `-duration` the session has no expiry, which means nothing will clean it up for you.

**Flags**

| Flag | Default | Description |
| --- | --- | --- |
| `-duration` | unlimited | Rental length, as seconds (`600`) or with a unit (`10m`). |
| `-country` | server's choice | Geo-IP country code. |
| `-timezone` | matches the country | IANA timezone. |
| `-proxy` | none | `host:port` or `host:port:user:pass`. |

### `run`

```bash
browserscale run <session-id> scrape.js
browserscale run <session-id> -e "return (await browser.getPages())[0].url"
browserscale run <session-id> -            # read the script from stdin
```

The script does not run in the page. It runs beside the browser in an isolate of its own and reaches the document through the engine: a cross-origin `<iframe>` is read as plain `contentDocument` with no frame ids anywhere, values come back as live objects you can assign to rather than snapshots, an element can be handed straight to `browser.click`, and the page sees nothing injected. Steps cost microseconds instead of network round trips, so loops are affordable.

A guide for it is still to come. Until then, the examples above and `browserscale run -h`.

**The log streams.** Lines appear on stdout as the script prints them, not in one block when it finishes — so a script that runs for a minute is something you can watch, and a script that hangs shows you where. The return value follows the log on the same stream once the script ends.

Only the CLI's own remarks go to stderr. When something has to *parse* the result, use `-json`: it gives one document with the value and the log as separate fields, rather than a stream meant for a human to read.

The command exits non-zero if the script throws, and Ctrl-C cancels the run in the browser rather than leaving it going.

**Flags**

| Flag | Default | Description |
| --- | --- | --- |
| `-e` | — | Run this source instead of a file. |
| `-detach` | off | Start the script, print its run id, and exit. The script keeps running. |

### `view`

```bash
browserscale view <session-id>              # watch it, and drive it
browserscale view <session-id> -read-only   # watch without touching it
browserscale view <session-id> -tab         # an ordinary browser tab, not a window
browserscale view <session-id> -no-open     # print the URL, open it yourself
```

Shows a session's screen, with your mouse and keyboard wired through to it. Useful for the parts of a flow that are easier done than automated — signing in once by hand so a script can inherit the cookies, or watching what a selector actually hits when a run keeps failing on it.

**It opens a window of its own, not a picture in the terminal.** The stream is H.264 over WebRTC, and the thing on your machine that already decodes that well is a browser. So the CLI starts a small server on loopback, serves the same stream widget the web panel uses, and opens it in a window with no address bar or tabs, sized to the session's own screen. That window needs a Chromium-based browser — Chrome, Edge or Brave; without one, and with `-tab`, the page opens as an ordinary tab instead.

What that buys over just opening the panel is authentication. This command holds your API key and signs the signalling calls itself, so watching a session needs nothing but a key — no web login — and it works against a local or private deployment exactly as it works against the public one. The key never reaches the page, and the page is reachable only with a one-time token the CLI mints, so another site you have open cannot drive your session through it.

Video travels from the browser engine to your browser directly over the relay. It does not pass through the CLI, which is why watching costs nothing in latency and works the same on a slow machine.

Ctrl-C stops watching and frees the engine's encoder. The session keeps running.

**Flags**

| Flag | Default | Description |
| --- | --- | --- |
| `-read-only` | off | Watch without sending mouse or keyboard. |
| `-tab` | off | Open an ordinary browser tab instead of a window of its own. |
| `-no-open` | off | Print the URL instead of launching a browser. |
| `-port` | free port | Port for the local viewer. |

### `stop`

```bash
browserscale stop <session-id>
browserscale stop -all
```

Ends a session and stops billing for it; unused credits are refunded. This discards the session's state — cookies, logins, whatever a script was part way through — and there is no reconnecting afterwards.

`-all` stops every session the key holds, including ones another machine is using. It is a cleanup tool for a run that leaked sessions, not a way to tidy up "your" sessions in a shared account.

To cancel a *script* while keeping the browser, use [`runs stop`](#runs).

### `runs`

```bash
browserscale runs list   <session-id>
browserscale runs follow <session-id> [run-id]
browserscale runs stop   <session-id> [run-id]
```

A run is a script inside a session. These are grouped under a noun rather than being flags on the commands above because of one collision: stopping a run and stopping a session are both destructive and they destroy different things. Had `runs stop` been a flag on `stop`, a typo would decide between cancelling a script and discarding a browser.

`follow` streams the console output of scripts already running — useful after `run -detach`, or to look in on a script a previous process started. Only output produced from then on arrives; a line printed before you attached is not kept anywhere. Detaching with Ctrl-C leaves the script running.

`list` shows only scripts still executing. A finished run is reported once to whoever was watching and then forgotten, so an empty list means nothing is running, not that nothing ever ran.

## Agentic coding

The CLI is one half of working with an AI agent. The other is the **hosted MCP server**, which hands the agent a live cloud browser it can look at and act on, one call at a time. Nothing to install and nothing to keep running:

| | |
| --- | --- |
| Endpoint | `https://mcp.browserscale.cloud/mcp` |
| Transport | Streamable HTTP (a remote server, not stdio) |
| Auth | `Authorization: Bearer <your-api-key>` |
| Tool overview | [mcp.browserscale.cloud](https://mcp.browserscale.cloud/) — a page for humans, not an endpoint |

Client configs differ in their field names, but they all want the same three things:

```json
{
  "mcpServers": {
    "browserscale": {
      "url": "https://mcp.browserscale.cloud/mcp",
      "headers": { "Authorization": "Bearer sk_your_api_key" }
    }
  }
}
```

The key travels as a header and never as a tool argument, so it stays out of the model's context window and out of the conversation log.

**Why both.** Every MCP tool maps one-to-one onto an SDK method, so a call that worked over MCP is already a line of the program. That makes MCP the right tool for looking — walking an unfamiliar flow once, proving a selector resolves to exactly one element, taking a screenshot when the logs don't explain it — and the wrong tool for running the work, where you would pay tokens for every step and keep nothing when the conversation ends. The agent explores over MCP and writes what it learned into a module that runs without a model in the loop.

The server is also stateless, and any session on your account can be driven through it — not just one the agent rented itself. Log the `sessionId` and `grpcUrl` from your code and the agent can attach to a run in progress, or to the half-finished state a failed run left behind, instead of reconstructing it from a stack trace.

Full walkthrough: [the agentic coding guide](https://browserscale.cloud/docs/guides/agentic-coding).

## Ecosystem

| Project | Role |
| --- | --- |
| **browserscale** (you are here) | The command line for the platform. |
| [**browserscale-go**](https://github.com/browserscale/browserscale-go) | The Go SDK the generated module drives. |
| [**browserscale-kit**](https://github.com/browserscale/browserscale-kit) | The toolkit the generated module builds on (config, store, queues, proxies, logging, mail). |
| [**browserscale-ts**](https://github.com/browserscale/browserscale-ts) | The TypeScript SDK. |

## License

MIT © browserscale
