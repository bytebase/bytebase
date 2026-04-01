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

  // Only match single/double quoted strings — template literals with ${} are dynamic keys
  const patterns = [/\bt\(\s*["']([^"']+)["']/g];

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
// 7. Main
// ---------------------------------------------------------------------------
const reactKeys = collectKeys();
console.log(`Found ${reactKeys.size} unique translation keys in React code.`);

mkdirSync(OUT_DIR, { recursive: true });

function loadMergedFlat(locale) {
  const main = JSON.parse(
    readFileSync(`${LOCALES_DIR}/${locale}.json`, "utf-8")
  );
  const sub = JSON.parse(
    readFileSync(`${LOCALES_DIR}/subscription/${locale}.json`, "utf-8")
  );
  const dynamic = JSON.parse(
    readFileSync(`${LOCALES_DIR}/dynamic/${locale}.json`, "utf-8")
  );
  return flatten(deepMerge({}, main, { subscription: sub }, { dynamic }));
}

// First pass: determine which keys are pluralized in en-US (the reference locale).
// Other locales that lack pluralization (e.g. zh-CN) must still emit _one/_other
// to keep the key set consistent across all locales.
const enUSFlat = loadMergedFlat("en-US");
const pluralizedKeys = new Set();
for (const key of reactKeys) {
  const raw = enUSFlat[key];
  if (raw !== undefined && String(raw).includes(" | ")) {
    pluralizedKeys.add(key);
  }
}
for (const [fkey, fval] of Object.entries(enUSFlat)) {
  if (
    DYNAMIC_PREFIXES.some((p) => fkey.startsWith(p)) &&
    String(fval).includes(" | ")
  ) {
    pluralizedKeys.add(fkey);
  }
}

// Process a value, forcing pluralization split when en-US has it
function processValueConsistent(key, value, flat, output) {
  if (value.includes(" | ")) {
    // This locale has pluralization — split normally
    const parts = value.split(" | ");
    const singular = resolveLinked(parts[0].trim(), flat);
    const plural = resolveLinked(parts[parts.length - 1].trim(), flat);
    output[`${key}_one`] = singular.replace(/\{n\}/g, "{count}");
    output[`${key}_other`] = plural.replace(/\{n\}/g, "{count}");
  } else if (pluralizedKeys.has(key)) {
    // en-US pluralizes this key but this locale doesn't — emit both _one/_other
    // with the same resolved value to keep key sets consistent
    const resolved = resolveLinked(value, flat).replace(/\{n\}/g, "{count}");
    output[`${key}_one`] = resolved;
    output[`${key}_other`] = resolved;
  } else {
    output[key] = resolveLinked(value, flat);
  }
}

for (const locale of LOCALES) {
  const flat = loadMergedFlat(locale);
  const output = {};

  // Static keys extracted from source code
  for (const key of reactKeys) {
    const raw = flat[key];
    if (raw === undefined) continue;
    processValueConsistent(key, String(raw), flat, output);
  }

  // Dynamic prefix keys — copy entire subtrees
  for (const [fkey, fval] of Object.entries(flat)) {
    if (DYNAMIC_PREFIXES.some((p) => fkey.startsWith(p))) {
      processValueConsistent(fkey, String(fval), flat, output);
    }
  }

  const nested = unflatten(output);

  // Split dynamic keys into a separate file (mirrors Vue's locales/dynamic/ structure)
  const dynamicData = nested.dynamic || {};
  delete nested.dynamic;

  const mainOutPath = `${OUT_DIR}/${locale}.json`;
  writeFileSync(mainOutPath, JSON.stringify(nested, null, 2) + "\n");

  mkdirSync(`${OUT_DIR}/dynamic`, { recursive: true });
  const dynamicOutPath = `${OUT_DIR}/dynamic/${locale}.json`;
  writeFileSync(dynamicOutPath, JSON.stringify(dynamicData, null, 2) + "\n");

  console.log(`Wrote ${mainOutPath} + ${dynamicOutPath} (${Object.keys(output).length} keys)`);
}

console.log("Done.");
