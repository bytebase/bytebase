// frontend/scripts/extract-react-locales.mjs
//
// Usage: node frontend/scripts/extract-react-locales.mjs
//
// Scans all .tsx files under frontend/src/react/ for translation keys,
// extracts matching entries from Vue locale JSONs, resolves vue-i18n
// syntax (@: linked messages, | pluralization), and writes
// frontend/src/react/locales/<locale>.json.

import { readFileSync, writeFileSync, mkdirSync, readdirSync } from "fs";
import { resolve, dirname, join } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");
const LOCALES_DIR = resolve(ROOT, "src/locales");
const REACT_DIR = resolve(ROOT, "src/react");
const OUT_DIR = resolve(ROOT, "src/react/locales");
const LOCALES = ["en-US", "zh-CN", "es-ES", "ja-JP", "vi-VN"];

// Dynamic key prefixes — these subtrees are used via template literals
// in React code, so we copy the entire prefix tree.
const DYNAMIC_PREFIXES = [
  "dynamic.subscription.features.",
  "dynamic.subscription.purchase.features.",
  "dynamic.settings.sensitive-data.semantic-types.template.",
  "subscription.plan.",
  "settings.sensitive-data.algorithms.",
];

// ---------------------------------------------------------------------------
// 1. Collect translation keys from React source files
// ---------------------------------------------------------------------------
function findTsxFiles(dir) {
  const results = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...findTsxFiles(fullPath));
    } else if (entry.name.endsWith(".tsx")) {
      results.push(fullPath);
    }
  }
  return results;
}

function collectKeys() {
  const files = findTsxFiles(REACT_DIR);
  const keys = new Set();

  // Matches: t("key"), t('key'), tVue(t, "key"), vueT("key")
  const patterns = [
    /\bt\(\s*["'`]([^"'`]+)["'`]/g,
    /\btVue\(\s*\w+,\s*["'`]([^"'`]+)["'`]/g,
    /\bvueT\(\s*["'`]([^"'`]+)["'`]/g,
  ];

  for (const file of files) {
    const src = readFileSync(file, "utf-8");
    for (const re of patterns) {
      re.lastIndex = 0;
      let m;
      while ((m = re.exec(src))) {
        keys.add(m[1]);
      }
    }
  }
  return keys;
}

// ---------------------------------------------------------------------------
// 2. Load a flat key→value map from nested JSON
// ---------------------------------------------------------------------------
function flatten(obj, prefix = "") {
  const result = {};
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (typeof v === "object" && v !== null && !Array.isArray(v)) {
      Object.assign(result, flatten(v, key));
    } else {
      result[key] = v;
    }
  }
  return result;
}

// ---------------------------------------------------------------------------
// 3. Deep merge (like lodash merge) for combining locale sources
// ---------------------------------------------------------------------------
function deepMerge(target, ...sources) {
  for (const source of sources) {
    for (const [key, val] of Object.entries(source)) {
      if (
        typeof val === "object" &&
        val !== null &&
        !Array.isArray(val) &&
        typeof target[key] === "object" &&
        target[key] !== null
      ) {
        target[key] = deepMerge({}, target[key], val);
      } else {
        target[key] = val;
      }
    }
  }
  return target;
}

// ---------------------------------------------------------------------------
// 4. Resolve vue-i18n @:linked messages
// ---------------------------------------------------------------------------
function resolveLinked(value, flat) {
  return value.replace(/@:(?:\{'([^']+)'\}|(\S+))/g, (_m, quoted, plain) => {
    const ref = quoted ?? plain;
    const resolved = flat[ref];
    if (resolved === undefined) return ref;
    return resolveLinked(String(resolved), flat);
  });
}

// ---------------------------------------------------------------------------
// 5. Unflatten dot-separated keys back into nested JSON
// ---------------------------------------------------------------------------
function unflatten(flat) {
  const result = {};
  for (const [key, value] of Object.entries(flat)) {
    const parts = key.split(".");
    let node = result;
    for (let i = 0; i < parts.length - 1; i++) {
      if (!(parts[i] in node)) node[parts[i]] = {};
      node = node[parts[i]];
    }
    node[parts[parts.length - 1]] = value;
  }
  return result;
}

// ---------------------------------------------------------------------------
// 6. Process a single value: resolve links, handle pluralization
// ---------------------------------------------------------------------------
function processValue(key, value, flat, output) {
  if (value.includes(" | ")) {
    const parts = value.split(" | ");
    const singular = resolveLinked(parts[0].trim(), flat);
    const plural = resolveLinked(parts[parts.length - 1].trim(), flat);
    // Normalize {n} to {count} for i18next pluralization
    output[`${key}_one`] = singular.replace(/\{n\}/g, "{count}");
    output[`${key}_other`] = plural.replace(/\{n\}/g, "{count}");
  } else {
    output[key] = resolveLinked(value, flat);
  }
}

// ---------------------------------------------------------------------------
// 7. Main
// ---------------------------------------------------------------------------
const reactKeys = collectKeys();
console.log(`Found ${reactKeys.size} unique translation keys in React code.`);

mkdirSync(OUT_DIR, { recursive: true });

for (const locale of LOCALES) {
  // Deep-merge all source files (main has a subscription key too)
  const main = JSON.parse(
    readFileSync(`${LOCALES_DIR}/${locale}.json`, "utf-8")
  );
  const sub = JSON.parse(
    readFileSync(`${LOCALES_DIR}/subscription/${locale}.json`, "utf-8")
  );
  const dynamic = JSON.parse(
    readFileSync(`${LOCALES_DIR}/dynamic/${locale}.json`, "utf-8")
  );

  const merged = deepMerge({}, main, { subscription: sub }, { dynamic });
  const flat = flatten(merged);

  const output = {};

  // Static keys extracted from source code
  for (const key of reactKeys) {
    const raw = flat[key];
    if (raw === undefined) continue;
    processValue(key, String(raw), flat, output);
  }

  // Dynamic prefix keys — copy entire subtrees
  for (const [fkey, fval] of Object.entries(flat)) {
    if (DYNAMIC_PREFIXES.some((p) => fkey.startsWith(p))) {
      processValue(fkey, String(fval), flat, output);
    }
  }

  const nested = unflatten(output);
  const outPath = `${OUT_DIR}/${locale}.json`;
  writeFileSync(outPath, JSON.stringify(nested, null, 2) + "\n");
  console.log(`Wrote ${outPath} (${Object.keys(output).length} keys)`);
}

console.log("Done.");
