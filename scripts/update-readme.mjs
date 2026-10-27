#!/usr/bin/env node
// Writes the current versions into README.md, between its versions markers:
//
//   gh release list --exclude-drafts --limit 200 --json tagName,publishedAt > releases.json
//   node scripts/update-readme.mjs --releases releases.json --readme README.md
//
// A version with a pre-release part (0.5.0-rc.1) is a prerelease, as on npm,
// whatever the release's flag on GitHub says. The newest stable release always
// shows; the newest prerelease only while it is ahead of it.
import { readFileSync, writeFileSync } from "node:fs";
import process from "node:process";

const START = "<!-- versions:start -->";
const END = "<!-- versions:end -->";
const RELEASES = "https://github.com/browserscale/browserscale/releases/tag";

function parseArgs(argv) {
  const out = { releases: "", readme: "README.md" };
  for (let i = 0; i < argv.length; i++) {
    const [flag, inline] = argv[i].split("=", 2);
    const value = inline ?? argv[++i];
    switch (flag) {
      case "--releases":
        out.releases = value;
        break;
      case "--readme":
        out.readme = value;
        break;
      default:
        fail(`unknown flag ${flag}`);
    }
  }
  if (!out.releases) fail("--releases is required (gh release list --json tagName,publishedAt)");
  return out;
}

function fail(message) {
  process.stderr.write(`update-readme: ${message}\n`);
  process.exit(1);
}

/** major.minor.patch and the pre-release identifiers, or null for other tags. */
function parse(tag) {
  const m = /^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/.exec(tag);
  if (!m) return null;
  return { core: [+m[1], +m[2], +m[3]], pre: m[4] ? m[4].split(".") : [] };
}

/** Semver precedence: a version without a pre-release part ranks above one with. */
function compare(a, b) {
  for (let i = 0; i < 3; i++) {
    if (a.core[i] !== b.core[i]) return a.core[i] - b.core[i];
  }
  if (!a.pre.length || !b.pre.length) return b.pre.length - a.pre.length;
  for (let i = 0; i < Math.max(a.pre.length, b.pre.length); i++) {
    const x = a.pre[i];
    const y = b.pre[i];
    if (x === undefined) return -1;
    if (y === undefined) return 1;
    const nx = /^\d+$/.test(x);
    const ny = /^\d+$/.test(y);
    if (nx && ny && +x !== +y) return +x - +y;
    if (nx !== ny) return nx ? -1 : 1;
    if (x !== y) return x < y ? -1 : 1;
  }
  return 0;
}

function newest(releases) {
  return releases.reduce((best, r) => (!best || compare(r.v, best.v) > 0 ? r : best), null);
}

function row(label, r, install) {
  const version = r.tagName.replace(/^v/, "");
  const date = (r.publishedAt || "").slice(0, 10);
  return `| **${label}** | [${version}](${RELEASES}/${r.tagName}) | ${date} | \`${install}\` |`;
}

const args = parseArgs(process.argv.slice(2));
let releases;
try {
  releases = JSON.parse(readFileSync(args.releases, "utf8"));
} catch (err) {
  fail(`could not read ${args.releases}: ${err.message}`);
}
releases = releases
  .map((r) => ({ ...r, v: parse(r.tagName) }))
  .filter((r) => r.v && r.publishedAt);

const stable = newest(releases.filter((r) => !r.v.pre.length));
const pre = newest(releases.filter((r) => r.v.pre.length));
const rows = [];
const shown = [];
if (stable) {
  rows.push(row("Stable", stable, "npm i -g browserscale"));
  shown.push(`stable ${stable.tagName}`);
}
if (pre && (!stable || compare(pre.v, stable.v) > 0)) {
  rows.push(row("Prerelease", pre, "npm i -g browserscale@next"));
  shown.push(`prerelease ${pre.tagName}`);
}
if (!rows.length) fail(`${args.releases} has no published release with a version tag`);

const block = [START, "| | Version | Released | npm |", "| --- | --- | --- | --- |", ...rows, END].join("\n");

const readme = readFileSync(args.readme, "utf8");
const start = readme.indexOf(START);
const end = readme.indexOf(END);
if (start < 0 || end < start) fail(`${args.readme} has no ${START} … ${END} block`);
const updated = readme.slice(0, start) + block + readme.slice(end + END.length);
writeFileSync(args.readme, updated);
process.stdout.write(`update-readme: ${shown.join(", ")}\n`);
