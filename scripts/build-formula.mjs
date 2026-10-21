#!/usr/bin/env node
// Writes the Homebrew formula for a release, for the tap browserscale/homebrew-tap:
//
//   node scripts/build-formula.mjs --version 1.2.3 --checksums assets/checksums.txt --out Formula/browserscale.rb
//
// The formula installs the macOS archives browserscale-studio_<version>_darwin_<goarch>.tar.gz,
// whose checksums it takes from the release's checksums.txt (sha256sum's
// format). It puts the command and "browserscale studio.app" side by side in
// libexec, where the command finds Studio, and links only the command into bin.
import { readFileSync, writeFileSync } from "node:fs";
import process from "node:process";

const RELEASES = "https://github.com/browserscale/browserscale/releases/download";

function parseArgs(argv) {
  const out = { checksums: "checksums.txt", out: "", version: "" };
  for (let i = 0; i < argv.length; i++) {
    const [flag, inline] = argv[i].split("=", 2);
    const value = inline ?? argv[++i];
    switch (flag) {
      case "--version":
        out.version = value;
        break;
      case "--checksums":
        out.checksums = value;
        break;
      case "--out":
        out.out = value;
        break;
      default:
        fail(`unknown flag ${flag}`);
    }
  }
  if (!out.version) fail("--version is required (e.g. --version 1.2.3)");
  if (!out.out) fail("--out is required (e.g. --out Formula/browserscale.rb)");
  out.version = out.version.replace(/^v/, "");
  if (!/^\d+\.\d+\.\d+(-[\w.]+)?$/.test(out.version)) {
    fail(`--version ${out.version} does not look like a semver version`);
  }
  return out;
}

function fail(message) {
  process.stderr.write(`build-formula: ${message}\n`);
  process.exit(1);
}

/** The sha256 of each file in checksums.txt, by file name. */
function readChecksums(file) {
  let text;
  try {
    text = readFileSync(file, "utf8");
  } catch (err) {
    fail(`could not read ${file}: ${err.message}`);
  }
  const sums = new Map();
  for (const line of text.split("\n")) {
    const m = /^([0-9a-f]{64}) [ *](.+)$/.exec(line.trim());
    if (m) sums.set(m[2], m[1]);
  }
  return sums;
}

function archive(version, goarch) {
  return `browserscale-studio_${version}_darwin_${goarch}.tar.gz`;
}

const args = parseArgs(process.argv.slice(2));
const sums = readChecksums(args.checksums);
const sha = {};
for (const goarch of ["arm64", "amd64"]) {
  sha[goarch] = sums.get(archive(args.version, goarch));
  if (!sha[goarch]) fail(`${args.checksums} has no ${archive(args.version, goarch)}`);
}

const formula = `class Browserscale < Formula
  desc "Build browser automations by chatting with an agent"
  homepage "https://browserscale.cloud"
  version "${args.version}"

  on_arm do
    url "${RELEASES}/v#{version}/browserscale-studio_#{version}_darwin_arm64.tar.gz"
    sha256 "${sha.arm64}"
  end
  on_intel do
    url "${RELEASES}/v#{version}/browserscale-studio_#{version}_darwin_amd64.tar.gz"
    sha256 "${sha.amd64}"
  end

  depends_on :macos

  def install
    # The command finds Studio's bundle next to itself, through the symlink.
    libexec.install "browserscale", "browserscale studio.app"
    bin.install_symlink libexec/"browserscale"
  end

  test do
    system bin/"browserscale", "version"
  end
end
`;

writeFileSync(args.out, formula);
process.stdout.write(`build-formula: wrote ${args.out} for ${args.version}\n`);
