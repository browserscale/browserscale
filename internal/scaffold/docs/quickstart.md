<!--
  url: https://browserscale.cloud/docs/quickstart
  title: Quickstart
  description: Install the browserscale Go or TypeScript SDK, rent a cloud Chromium session, and run your first script in under five minutes.
-->

# Quickstart

In the next four steps you will install the SDK, grab an API key, rent
your first browser, drive it through one navigation and one JavaScript
evaluation, then release the session. Start to finish in under a minute.

> **TL;DR**
>
> - Install the SDK in your language of choice.
> - Grab an API key from your dashboard.
> - Copy the example below and replace `YOUR_API_KEY`.
> - Run it; you should see `Example Domain` printed.

## 1. Install

**Go:**

```go
go get github.com/browserscale/browserscale-go
```

**TypeScript:**

```ts
npm install browserscale-ts
```

Go users need Go **1.21+**; Node users need Node **18+**. The Go module
ships the gRPC stubs precompiled, the npm package ships an ES-module
build with bundled types.

## 2. Get an API key

browserscale charges credits **per minute, rounded up** — and the full reserved
window is deducted upfront when you call `RentBrowser`, refunded
automatically if the rent itself fails. Top up a few credits before you
start.

1. Sign up at [browserscale.cloud/register](/register).
2. Open [the API key page](/dashboard/api-keys).
3. Copy the `sk_…` value into the snippet below.

For development a plain `.env` file is fine; the SDK never reads env vars
on its own, you always pass the key explicitly.

## 3. Your first script

The script below is the smallest end-to-end browserscale program. It rents a
session, navigates to `example.com`, waits for the page's `h1` to
appear, evaluates one line of JavaScript on the page, and stops
cleanly. Every later example in these docs is just a variation on this
skeleton — so it is worth reading the comments rather than just
copy-pasting.

**Go:**

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/browserscale/browserscale-go"
)

func main() {
    ctx := context.Background()

    // 1. Configure the rental. Empty proxy fields tell browserscale to allocate
    //    a managed proxy server-side; pass your own host/port/creds to
    //    bring your own.
    cfg := browserscale.NewBrowserConfig(
        "YOUR_API_KEY", // sk_…
        300,            // rentDuration in seconds (5 minutes)
        "", 0, "", "",  // proxy host / port / user / pass
    )

    // 2. Rent a fresh browser session.
    browser, err := browserscale.RentBrowser(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer browser.Close() // always release the session

    // 3. Drive it. Navigate returns when the page *commits*, so wait
    //    for an element you actually need before reading the DOM.
    if _, err := browser.Navigate(ctx, "https://example.com", 0); err != nil {
        log.Fatal(err)
    }
    if _, err := browser.Wait(ctx, browserscale.CSS("h1")); err != nil {
        log.Fatal(err)
    }

    res, err := browser.Evaluate(ctx, "document.title")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("title:", res.Value)
}
```

**TypeScript:**

```ts
import { rentBrowser, BrowserConfig, css } from "browserscale-ts";

async function main() {
    // 1. Configure the rental. Empty proxy fields tell browserscale to allocate
    //    a managed proxy server-side; pass your own host/port/creds to
    //    bring your own.
    const cfg = new BrowserConfig(
        "YOUR_API_KEY", // sk_…
        300,            // rentDuration in seconds (5 minutes)
        "", 0, "", "",  // proxy host / port / user / pass
    );

    // 2. Rent a fresh browser session.
    const browser = await rentBrowser(cfg);
    try {
        // 3. Drive it. navigate returns when the page *commits*, so
        //    wait for an element you actually need before reading the DOM.
        await browser.navigate("https://example.com");
        await browser.wait(css("h1"));

        const res = await browser.evaluate("document.title");
        console.log("title:", res.value);
    } finally {
        await browser.stopBrowser(); // always release the session
    }
}

main();
```

## 4. Run it

**Go:**

```go
go run main.go
# title: Example Domain
```

**TypeScript:**

```ts
# save as quickstart.ts
npx tsx quickstart.ts
# title: Example Domain
```

If you see `title: Example Domain`, congratulations — the script just
rented a real Chromium session somewhere on our infrastructure, drove it
through a navigation and a JS call, and released it. From here, every
other method in the SDK is incremental.

## See also

- The [Go SDK reference](/docs/api-reference/go) lists every method, signature, and option.
- [`navigate`](/docs/api-reference/go#Navigate) returns when the page *commits*, not when it finishes loading — that distinction is covered in Core concepts.
- [`evaluate`](/docs/api-reference/go#Evaluate) is the escape hatch when no built-in method does what you need.

→ Continue: [Core concepts](/docs/concepts)
