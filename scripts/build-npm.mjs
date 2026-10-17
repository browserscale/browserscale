#!/usr/bin/env node
// Turns a release's archives into publishable npm packages: one package per
// platform with the command and Studio, plus the wrapper that depends on all
// of them.
//
//   node scripts/build-npm.mjs --version 1.2.3 --assets assets [--out dist/npm]
//
// --assets holds the release's files as `gh release download` leaves them:
// for macOS and Linux the browserscale_<version>_<goos>_<goarch>.tar.gz
// archives, each with the command and Studio side by side (browserscale and
// browserscale-studio, or on macOS the "browserscale studio.app" bundle), and
// for Windows the two programs themselves, browserscale_<version>_windows_<goarch>.exe
// and browserscale-studio_<version>_windows_<goarch>.exe. The command finds
// Studio next to itself.
//
// Writes nothing outside --out and publishes nothing. Run it on Linux or
// macOS: it sets the executable bits, and chmod is a no-op on Windows, so a
// set packed there installs programs nobody can run.
import { spawnSync } from "node:child_process";
import { chmodSync, copyFileSync, cpSync, existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import os from "node:os";
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
const BUNDLE = "browserscale studio.app";

function parseArgs(argv) {
  const out = { assets: "assets", out: path.join("dist", "npm"), version: "" };
  for (let i = 0; i < argv.length; i++) {
    const [flag, inline] = argv[i].split("=", 2);
    const value = inline ?? argv[++i];
    switch (flag) {
      case "--version":
        out.version = value;
        break;
      case "--assets":
        out.assets = value;
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

function run(cmd, args) {
  const r = spawnSync(cmd, args, { stdio: ["ignore", "ignore", "pipe"] });
  if (r.error) return r.error;
  if (r.status !== 0) return new Error(`${cmd} exited ${r.status}: ${r.stderr.toString().trim()}`);
  return null;
}

function extract(archive, dir) {
  mkdirSync(dir, { recursive: true });
  const err = run("tar", ["-xzf", archive, "-C", dir]);
  if (err) fail(`could not extract ${archive}: ${err.message}`);
}

/** The release files for each target, by their names; every target must have them. */
function readAssets(dir, version) {
  if (!existsSync(dir)) fail(`${dir} not found: download the release's assets into it first`);
  const names = new Set(readdirSync(dir));
  const found = new Map();
  const missing = [];
  for (const t of TARGETS) {
    const files = t.os === "win32"
      ? [`browserscale_${version}_windows_${t.goarch}.exe`, `browserscale-studio_${version}_windows_${t.goarch}.exe`]
      : [`browserscale_${version}_${t.goos}_${t.goarch}.tar.gz`];
    const absent = files.filter((n) => !names.has(n));
    if (absent.length === 0) found.set(t, files.map((n) => path.join(dir, n)));
    else missing.push(...absent);
  }
  if (missing.length > 0) {
    fail(
      `missing in ${dir}: ${missing.join(", ")}\n` +
        `  (publishing a partial set would leave those platforms with an unusable wrapper)`
    );
  }
  return found;
}

/** What each platform package carries in bin/, as the release names it inside an archive or by itself. */
function contents(t) {
  if (t.os === "win32") return ["browserscale.exe", "browserscale-studio.exe"];
  if (t.os === "darwin") return ["browserscale", BUNDLE];
  return ["browserscale", "browserscale-studio"];
}

function packageJSON(t, version) {
  return {
    name: `${SCOPE}/cli-${t.os}-${t.cpu}`,
    version,
    description: `browserscale studio and its command line for ${t.os} ${t.cpu}`,
    homepage: "https://browserscale.cloud/studio",
    repository: { type: "git", url: "git+https://github.com/browserscale/browserscale.git" },
    license: "SEE LICENSE IN LICENSE",
    os: [t.os],
    cpu: [t.cpu],
    files: ["bin"],
    // Yarn PnP would otherwise zip the package, which breaks exec.
    preferUnplugged: true,
  };
}

function writeJSON(file, value) {
  writeFileSync(file, JSON.stringify(value, null, 2) + "\n");
}

function buildPlatformPackage(t, files, version, outDir, scratch) {
  const name = `cli-${t.os}-${t.cpu}`;
  const dir = path.join(outDir, SCOPE, name);
  const bin = path.join(dir, "bin");
  mkdirSync(bin, { recursive: true });

  if (t.os === "win32") {
    // The programs themselves, in contents' order, under their own names.
    contents(t).forEach((entry, i) => copyFileSync(files[i], path.join(bin, entry)));
  } else {
    const [archive] = files;
    const unpacked = path.join(scratch, name);
    extract(archive, unpacked);
    for (const entry of contents(t)) {
      const from = path.join(unpacked, entry);
      if (!existsSync(from)) fail(`${path.basename(archive)} has no ${entry}`);
      cpSync(from, path.join(bin, entry), { recursive: true });
    }
  }
  // npm keeps the mode from the tarball, so the bits have to be set here.
  const programs = contents(t).flatMap((e) =>
    e === BUNDLE ? readdirSync(path.join(bin, BUNDLE, "Contents", "MacOS")).map((f) => path.join(BUNDLE, "Contents", "MacOS", f)) : [e]
  );
  for (const p of programs) chmodSync(path.join(bin, p), 0o755);

  writeJSON(path.join(dir, "package.json"), packageJSON(t, version));
  copyFileSync("LICENSE", path.join(dir, "LICENSE"));
  writeFileSync(
    path.join(dir, "README.md"),
    `# ${SCOPE}/${name}\n\nbrowserscale studio and its command line for ${t.os} ${t.cpu}.\n\n` +
      `Do not install this directly. Install [\`browserscale\`](https://www.npmjs.com/package/browserscale), ` +
      `which pulls in the package matching your platform.\n`
  );
  return `${SCOPE}/${name}`;
}

function buildWrapperPackage(version, outDir) {
  const dir = path.join(outDir, "browserscale");
  mkdirSync(path.join(dir, "bin"), { recursive: true });

  const manifest = JSON.parse(readFileSync(path.join(WRAPPER_SRC, "package.json"), "utf8"));
  manifest.version = version;
  // The template guards itself against being published directly: its
  // placeholder version would ship a wrapper whose optionalDependencies
  // resolve to nothing. Both guards come off the staged copy, which is the
  // one with real versions. (`private` alone is not enough: npm only enforces
  // it for workspace publishes, so the prepublishOnly script is what actually
  // stops it.)
  delete manifest.private;
  delete manifest.scripts;
  // Exact pins: a wrapper must never resolve a binary from a different release.
  manifest.optionalDependencies = Object.fromEntries(TARGETS.map((t) => [`${SCOPE}/cli-${t.os}-${t.cpu}`, version]));
  writeJSON(path.join(dir, "package.json"), manifest);
  copyFileSync(path.join(WRAPPER_SRC, "bin", "browserscale.js"), path.join(dir, "bin", "browserscale.js"));
  // The repository README, so npm and GitHub never tell two different stories.
  // It must keep using absolute links: npm does not resolve relative ones.
  copyFileSync("README.md", path.join(dir, "README.md"));
  copyFileSync("LICENSE", path.join(dir, "LICENSE"));
  return dir;
}

const args = parseArgs(process.argv.slice(2));
const assets = readAssets(args.assets, args.version);

rmSync(args.out, { recursive: true, force: true });
mkdirSync(args.out, { recursive: true });
const scratch = mkdtempSync(path.join(os.tmpdir(), "build-npm-"));

try {
  const platformPackages = TARGETS.map((t) => buildPlatformPackage(t, assets.get(t), args.version, args.out, scratch));
  buildWrapperPackage(args.version, args.out);
  process.stdout.write(
    `build-npm: staged ${platformPackages.length + 1} packages at version ${args.version} in ${args.out}\n` +
      `  publish the platform packages first, then the wrapper:\n` +
      platformPackages.map((p) => `    ${p}\n`).join("") +
      `    browserscale\n`
  );
} finally {
  rmSync(scratch, { recursive: true, force: true });
}
