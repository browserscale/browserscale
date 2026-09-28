# browserscale

The command line for [browserscale.cloud](https://browserscale.cloud) — scaffold a runnable automation module, then build, restart and stream it.

```bash
npm i -g browserscale
browserscale init
```

This package is a thin launcher. The CLI itself is a single static Go binary, shipped as one small package per platform (`@browserscale/cli-linux-x64` and friends) and selected by npm through the `os`/`cpu` fields. There is no postinstall script and nothing is downloaded at install time.

Or build it from source with a Go toolchain:

```bash
go install github.com/browserscale/browserscale@latest
```

Full documentation: [browserscale.cloud/docs](https://browserscale.cloud/docs) · Source and issues: [github.com/browserscale/browserscale](https://github.com/browserscale/browserscale)

MIT © browserscale
