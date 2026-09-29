#!/usr/bin/env node
// Turns the binaries GoReleaser produced into publishable npm packages: one
// tiny package per platform plus the wrapper that depends on all of them.
//
// GoReleaser has no npm publisher, so this reads dist/artifacts.json (its own
// manifest) instead of guessing directory names — those carry suffixes like
// _amd64_v1 that change with GOAMD64 settings.
//
//   node scripts/build-npm.mjs --version 1.2.3 [--dist dist] [--out dist/npm]
//
// Writes nothing outside --out and publishes nothing. Run it on Linux or
// macOS: it sets the executable bit on the binaries, and chmod is a no-op on
// Windows, so a set packed there installs a binary nobody can run.
import { existsSync, mkdirSync, readFileSync, writeFileSync, copyFileSync, chmodSync, rmSync } from "node:fs";
import path from "node:path";
import process from "node:process";

// npm matches these against process.platform / process.arch at install time,
// which is why the package names use Node's spelling and not Go's.
const TARGETS = [
  { goos: "darwin", goarch: "amd64", os: "darwin", cpu: "x64" },
  { goos: "darwin", goarch: "arm64", os: "darwin", cpu: "arm64" },
  { goos: "linux", goarch: "amd64", os: "linux", cpu: "x64" },
  { goos: "linux", goarch: "arm64", os: "linux", cpu: "arm64" },
  { goos: "windows", goarch: "amd64", os: "win32", cpu: "x64" },
  { goos: "windows", goarch: "arm64", os: "win32", cpu: "arm64" },
];

const SCOPE = "@browserscale";
const WRAPPER_SRC = "npm/browserscale";

function parseArgs(argv) {
  const out = { dist: "dist", out: path.join("dist", "npm"), version: "" };
  for (let i = 0; i < argv.length; i++) {
    const [flag, inline] = argv[i].split("=", 2);
    const value = inline ?? argv[++i];
    switch (flag) {
      case "--version":
        out.version = value;
        break;
      case "--dist":
        out.dist = value;
        break;
      case "--out":
        out.out = value;
        break;
      default:
        fail(`unknown flag ${flag}`);
    }
  }
  if (!out.version) fail("--version is required (e.g. --version 1.2.3)");
  out.version = out.version.replace(/^v/, "");
  if (!/^\d+\.\d+\.\d+(-[\w.]+)?$/.test(out.version)) {
    fail(`--version ${out.version} does not look like a semver version`);
  }
  return out;
}

function fail(message) {
  process.stderr.write(`build-npm: ${message}\n`);
  process.exit(1);
}

// GoReleaser records paths relative to the directory it ran in, which is
// normally the repo root — but allow being called from elsewhere.
function resolveArtifact(distDir, p) {
  for (const candidate of [p, path.resolve(process.cwd(), p), path.resolve(distDir, "..", p)]) {
    if (existsSync(candidate)) return candidate;
  }
  return null;
}

function readBinaries(distDir) {
  const manifest = path.join(distDir, "artifacts.json");
  if (!existsSync(manifest)) {
    fail(`${manifest} not found — run goreleaser first (\`goreleaser release --snapshot --clean\` for a dry run)`);
  }
  const artifacts = JSON.parse(readFileSync(manifest, "utf8"));
  const found = new Map();
  for (const a of artifacts) {
    if (a.type !== "Binary") continue;
    const resolved = resolveArtifact(distDir, a.path);
    if (resolved) found.set(`${a.goos}/${a.goarch}`, resolved);
  }
  return found;
}

function packageJSON(target, version) {
  return {
    name: `${SCOPE}/cli-${target.os}-${target.cpu}`,
    version,
    description: `browserscale CLI binary for ${target.os} ${target.cpu}`,
    homepage: "https://browserscale.cloud",
    repository: { type: "git", url: "git+https://github.com/browserscale/browserscale.git" },
    license: "MIT",
    os: [target.os],
    cpu: [target.cpu],
    files: ["bin"],
    // Yarn PnP would otherwise zip the package, which breaks exec.
    preferUnplugged: true,
  };
}

function writeJSON(file, value) {
  writeFileSync(file, JSON.stringify(value, null, 2) + "\n");
}

function buildPlatformPackage(target, binary, version, outDir) {
  const name = `cli-${target.os}-${target.cpu}`;
  const dir = path.join(outDir, SCOPE, name);
  mkdirSync(path.join(dir, "bin"), { recursive: true });

  const exe = target.os === "win32" ? "browserscale.exe" : "browserscale";
  const dest = path.join(dir, "bin", exe);
  copyFileSync(binary, dest);
  // npm keeps the mode from the tarball, so the bit has to be set here. On
  // Windows this is a no-op — pack the release on Linux or macOS.
  chmodSync(dest, 0o755);

  writeJSON(path.join(dir, "package.json"), packageJSON(target, version));
  writeFileSync(
    path.join(dir, "README.md"),
    `# ${SCOPE}/${name}\n\nThe browserscale CLI binary for ${target.os} ${target.cpu}.\n\n` +
      `Do not install this directly — install [\`browserscale\`](https://www.npmjs.com/package/browserscale), ` +
      `which pulls in the package matching your platform.\n`
  );
  return `${SCOPE}/${name}`;
}

function buildWrapperPackage(version, outDir) {
  const dir = path.join(outDir, "browserscale");
  mkdirSync(path.join(dir, "bin"), { recursive: true });

  const manifest = JSON.parse(readFileSync(path.join(WRAPPER_SRC, "package.json"), "utf8"));
  manifest.version = version;
  // The template guards itself against being published directly — its
  // placeholder version would ship a wrapper whose optionalDependencies
  // resolve to nothing. Both guards have to come off the staged copy, which
  // is the one with real versions. (`private` alone is not enough: npm only
  // enforces it for workspace publishes, so the prepublishOnly script is what
  // actually stops it.)
  delete manifest.private;
  delete manifest.scripts;
  // Exact pins: a wrapper must never resolve a binary from a different release.
  manifest.optionalDependencies = Object.fromEntries(
    TARGETS.map((t) => [`${SCOPE}/cli-${t.os}-${t.cpu}`, version])
  );
  writeJSON(path.join(dir, "package.json"), manifest);
  copyFileSync(path.join(WRAPPER_SRC, "bin", "browserscale.js"), path.join(dir, "bin", "browserscale.js"));
  // The repository README, so npm and GitHub never tell two different stories.
  // It must keep using absolute links: npm does not resolve relative ones.
  copyFileSync("README.md", path.join(dir, "README.md"));
  return dir;
}

const args = parseArgs(process.argv.slice(2));
const binaries = readBinaries(args.dist);

const missing = TARGETS.filter((t) => !binaries.has(`${t.goos}/${t.goarch}`));
if (missing.length > 0) {
  fail(
    `no binary in ${args.dist} for: ${missing.map((t) => `${t.goos}/${t.goarch}`).join(", ")}\n` +
      `  (publishing a partial set would leave those platforms with an unusable wrapper)`
  );
}

rmSync(args.out, { recursive: true, force: true });
mkdirSync(args.out, { recursive: true });

const platformPackages = TARGETS.map((t) =>
  buildPlatformPackage(t, binaries.get(`${t.goos}/${t.goarch}`), args.version, args.out)
);
buildWrapperPackage(args.version, args.out);

process.stdout.write(
  `build-npm: staged ${platformPackages.length + 1} packages at version ${args.version} in ${args.out}\n` +
    `  publish the platform packages first, then the wrapper:\n` +
    platformPackages.map((p) => `    ${p}\n`).join("") +
    `    browserscale\n`
);
