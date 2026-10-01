// Refreshes the viewer's vendored front-end assets from sibling checkouts of
// browserscale-widget and browserscale-devtools, after `npm run build` in each.
//
//   node scripts/sync-assets.mjs           copy them in
//   node scripts/sync-assets.mjs --check   exit 1 if any copy has drifted
//
// Point at other checkouts with BROWSERSCALE_WIDGET_DIR and
// BROWSERSCALE_DEVTOOLS_DIR.
import { readFile, writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const assets = join(root, "internal", "viewer", "assets");
const widget = process.env.BROWSERSCALE_WIDGET_DIR || join(root, "..", "browserscale-widget");
const devtools = process.env.BROWSERSCALE_DEVTOOLS_DIR || join(root, "..", "browserscale-devtools");

const files = [
  [join(widget, "dist", "index.js"), "widget.js"],
  [join(devtools, "dist", "standalone.js"), "devtools.js"],
  [join(devtools, "dist", "style.css"), "devtools.css"],
];

const check = process.argv.includes("--check");
let drift = 0;

for (const [from, name] of files) {
  let src;
  try {
    src = await readFile(from);
  } catch {
    console.error(`missing ${from}: run npm run build in that package first`);
    process.exit(2);
  }
  const to = join(assets, name);
  const cur = await readFile(to).catch(() => null);
  if (cur && Buffer.compare(cur, src) === 0) continue;
  if (check) {
    console.error(`${name} differs from ${from}`);
    drift++;
    continue;
  }
  await writeFile(to, src);
  console.log(`updated ${name}`);
}

if (drift) process.exit(1);
