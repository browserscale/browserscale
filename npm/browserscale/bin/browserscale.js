#!/usr/bin/env node
// Launcher for the npm distribution. This directory is the source template,
// not a publishable package — scripts/build-npm.mjs stages the real one into
// dist/npm/ with the release version filled in.
//
// The real CLI is a Go binary; this package
// only carries the per-platform packages as optionalDependencies, so npm picks
// the matching one via their "os"/"cpu" fields and skips the rest. That keeps
// the install free of postinstall scripts and network fetches, which means it
// also works under `npm ci --ignore-scripts` and in offline caches.
"use strict";

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const pkg = `@browserscale/cli-${process.platform}-${process.arch}`;
const exe = process.platform === "win32" ? "browserscale.exe" : "browserscale";

function resolveBinary() {
  try {
    // Resolve the manifest rather than the binary: it is the one path every
    // package manager layout (npm, pnpm, yarn) agrees on.
    const manifest = require.resolve(`${pkg}/package.json`);
    const binary = path.join(path.dirname(manifest), "bin", exe);
    return fs.existsSync(binary) ? binary : null;
  } catch {
    return null;
  }
}

const binary = resolveBinary();
if (!binary) {
  process.stderr.write(
    `browserscale: no binary for ${process.platform}-${process.arch} (looked for ${pkg}).\n\n` +
      `If this platform should be supported, the install probably skipped optional\n` +
      `dependencies. Reinstall without --no-optional / --omit=optional.\n\n` +
      `Otherwise install from source with a Go toolchain:\n` +
      `  go install github.com/browserscale/browserscale@latest\n`
  );
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  process.stderr.write(`browserscale: could not run ${binary}: ${result.error.message}\n`);
  process.exit(1);
}
// A signalled child reports a null status; Ctrl+C during `browserscale dev`
// lands here because the terminal signals the whole process group.
process.exit(result.status === null ? 1 : result.status);
