import { readFile, readdir } from "node:fs/promises";
import { gzipSync } from "node:zlib";

const manifest = JSON.parse(await readFile(new URL("../dist/.vite/manifest.json", import.meta.url), "utf8"));
const entryKey = manifest["index.html"] ? "index.html" : "src/main.ts";
const entry = manifest[entryKey];
if (!entry?.file) throw new Error("Unable to locate the frontend entry in the Vite manifest.");

const assetsDirectory = new URL("../dist/assets/", import.meta.url);
const assetNames = await readdir(assetsDirectory);
const gzipKb = async (url) => gzipSync(await readFile(url)).byteLength / 1024;
const eagerKeys = new Set();
function collectEagerImports(key) {
  if (eagerKeys.has(key) || !manifest[key]) return;
  eagerKeys.add(key);
  for (const importedKey of manifest[key].imports ?? []) collectEagerImports(importedKey);
}
collectEagerImports(entryKey);
const eagerJsFiles = Array.from(eagerKeys, (key) => manifest[key].file);
const eagerCssFiles = Array.from(new Set(Array.from(eagerKeys).flatMap((key) => manifest[key].css ?? [])));
const initialJsGzipKb = (await Promise.all(eagerJsFiles.map((file) => gzipKb(new URL(`../dist/${file}`, import.meta.url))))).reduce((sum, size) => sum + size, 0);
const initialCssGzipKb = (await Promise.all(eagerCssFiles.map((file) => gzipKb(new URL(`../dist/${file}`, import.meta.url))))).reduce((sum, size) => sum + size, 0);
const largestLazyGzipKb = Math.max(...await Promise.all(
  assetNames.filter((name) => name.endsWith(".js") && !eagerJsFiles.some((file) => file.endsWith(name))).map((name) => gzipKb(new URL(name, assetsDirectory))),
));

const budgets = {
  initialJsGzipKb: 115,
  initialCssGzipKb: 60,
  largestLazyGzipKb: 110,
};
const measurements = { initialJsGzipKb, initialCssGzipKb, largestLazyGzipKb };
const failures = Object.entries(budgets).filter(([name, limit]) => measurements[name] > limit);

console.log("Frontend bundle budgets (gzip kB)");
for (const [name, value] of Object.entries(measurements)) {
  console.log(`- ${name}: ${value.toFixed(1)} / ${budgets[name].toFixed(1)}`);
}
if (failures.length) {
  throw new Error(`Bundle budget exceeded: ${failures.map(([name]) => name).join(", ")}`);
}
