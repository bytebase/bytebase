// frontend/scripts/check-react-i18n-unused-keys.mjs
//
// Checks React i18n locale files for:
//   1. Unused keys — keys in locale files not referenced in React .tsx code
//   2. Cross-locale consistency — all locale files must have the same set of keys
//
// Missing keys (code → locale) are handled by the eslint plugin
// eslint-plugin-i18next-no-undefined-translation-keys.
//
// Usage: node frontend/scripts/check-react-i18n-unused-keys.mjs

import { readFileSync, readdirSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");
const REACT_DIR = resolve(ROOT, "src/react");
const LOCALES_DIR = resolve(REACT_DIR, "locales");
const LOCALES = ["en-US", "zh-CN", "es-ES", "ja-JP", "vi-VN"];

// Dynamic key prefixes — constructed at runtime via template literals,
// so they won't appear as literal strings in source code.
const DYNAMIC_PREFIXES = [
  "dynamic.subscription.features.",
  "dynamic.subscription.purchase.features.",
  "dynamic.settings.sensitive-data.semantic-types.template.",
  "subscription.plan.",
  "settings.sensitive-data.algorithms.",
];

let hasErrors = false;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function findFiles(dir, ext) {
  const results = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = resolve(dir, entry.name);
    if (entry.isDirectory() && entry.name !== "locales") {
      results.push(...findFiles(full, ext));
    } else if (entry.isFile() && entry.name.endsWith(ext)) {
      results.push(full);
    }
  }
  return results;
}

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

function collectKeysFromSource() {
  const files = findFiles(REACT_DIR, ".tsx");
  const keys = new Set();
  const patterns = [/\bt\(\s*["'`]([^"'`]+)["'`]/g];
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

function loadLocaleKeys(locale) {
  const main = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `${locale}.json`), "utf-8")
  );
  const dynamic = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `dynamic/${locale}.json`), "utf-8")
  );
  return new Set(Object.keys(flatten({ ...main, dynamic })));
}

// i18next pluralization: _one/_other suffixes map to the base key in code
function baseKey(key) {
  return key.replace(/_(one|other|zero|few|many)$/, "");
}

// ---------------------------------------------------------------------------
// Check 1: Unused keys (en-US locale vs source code)
// ---------------------------------------------------------------------------
const sourceKeys = collectKeysFromSource();
const enUSKeys = loadLocaleKeys("en-US");

const unused = [];
for (const key of enUSKeys) {
  const base = baseKey(key);
  if (sourceKeys.has(key) || sourceKeys.has(base)) continue;
  if (DYNAMIC_PREFIXES.some((p) => key.startsWith(p))) continue;
  unused.push(key);
}

if (unused.length > 0) {
  hasErrors = true;
  console.error(`\nUnused keys (${unused.length}):\n`);
  for (const key of unused.sort()) {
    console.error(`  - ${key}`);
  }
  console.error(
    "\nRemove these from frontend/src/react/locales/ or add to DYNAMIC_PREFIXES if constructed at runtime.\n"
  );
}

// ---------------------------------------------------------------------------
// Check 2: Cross-locale consistency (all locales must have same keys as en-US)
// ---------------------------------------------------------------------------
const referenceKeys = enUSKeys;

for (const locale of LOCALES) {
  if (locale === "en-US") continue;
  const localeKeys = loadLocaleKeys(locale);

  const missing = [...referenceKeys].filter((k) => !localeKeys.has(k));
  const extra = [...localeKeys].filter((k) => !referenceKeys.has(k));

  if (missing.length > 0) {
    hasErrors = true;
    console.error(`\n${locale}: missing ${missing.length} key(s) (present in en-US):\n`);
    for (const key of missing.sort()) {
      console.error(`  - ${key}`);
    }
  }

  if (extra.length > 0) {
    hasErrors = true;
    console.error(`\n${locale}: extra ${extra.length} key(s) (not in en-US):\n`);
    for (const key of extra.sort()) {
      console.error(`  + ${key}`);
    }
  }
}

// ---------------------------------------------------------------------------
// Result
// ---------------------------------------------------------------------------
if (hasErrors) {
  process.exit(1);
} else {
  console.log("React i18n checks passed: no unused keys, all locales consistent.");
}
