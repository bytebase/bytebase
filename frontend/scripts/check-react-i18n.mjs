// frontend/scripts/check-react-i18n.mjs
//
// Enforces strict 1:1 mapping between React code and React locale files:
//   1. Missing keys  — t("key") in code but key not in locale files
//   2. Unused keys   — key in locale files but not referenced in code
//   3. Consistency   — all locale files must have the exact same key set
//
// Usage: node frontend/scripts/check-react-i18n.mjs

import { readFileSync, readdirSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

// Keys constructed at runtime via template literals — exempt from unused check.
const DYNAMIC_PREFIXES = [
  "dynamic.subscription.features.",
  "dynamic.subscription.purchase.features.",
  "dynamic.settings.sensitive-data.semantic-types.template.",
  "subscription.plan.",
  "settings.sensitive-data.algorithms.",
];

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");
const REACT_DIR = resolve(ROOT, "src/react");
const LOCALES_DIR = resolve(REACT_DIR, "locales");
const LOCALES = ["en-US", "zh-CN", "es-ES", "ja-JP", "vi-VN"];

let errors = 0;

function error(msg) {
  console.error(msg);
  errors++;
}

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

// i18next pluralization: _one/_other suffixes map to the base key in code
function baseKey(key) {
  return key.replace(/_(one|other|zero|few|many)$/, "");
}

function isDynamic(key) {
  return DYNAMIC_PREFIXES.some((p) => key.startsWith(p));
}

// ---------------------------------------------------------------------------
// Collect translation keys from React source (.tsx) files
// ---------------------------------------------------------------------------
function collectSourceKeys() {
  const files = findFiles(REACT_DIR, ".tsx");
  const keys = new Set();
  // Only match single/double quoted strings — template literals with ${} are dynamic keys
  const re = /\bt\(\s*["']([^"']+)["']/g;
  for (const file of files) {
    const src = readFileSync(file, "utf-8");
    re.lastIndex = 0;
    let m;
    while ((m = re.exec(src))) {
      keys.add(m[1]);
    }
  }
  return keys;
}

// ---------------------------------------------------------------------------
// Collect keys from a locale (main + dynamic)
// ---------------------------------------------------------------------------
function loadLocaleKeys(locale) {
  const main = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `${locale}.json`), "utf-8")
  );
  const dynamic = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `dynamic/${locale}.json`), "utf-8")
  );
  return new Set(Object.keys(flatten({ ...main, dynamic })));
}

// ---------------------------------------------------------------------------
// Check 1: Missing keys (in code but not in locale)
// ---------------------------------------------------------------------------
const sourceKeys = collectSourceKeys();
const enUSKeys = loadLocaleKeys("en-US");

// Build a set of "effective" locale keys: includes base keys for pluralized entries
const effectiveLocaleKeys = new Set(enUSKeys);
for (const key of enUSKeys) {
  effectiveLocaleKeys.add(baseKey(key));
}

const missing = [];
for (const key of sourceKeys) {
  if (effectiveLocaleKeys.has(key)) continue;
  if (isDynamic(key)) continue;
  missing.push(key);
}

if (missing.length > 0) {
  console.error(`Missing keys (${missing.length}) — used in code but not in locale files:\n`);
  for (const key of missing.sort()) {
    error(`  - ${key}`);
  }
  console.error();
}

// ---------------------------------------------------------------------------
// Check 2: Unused keys (in locale but not in code)
// ---------------------------------------------------------------------------
const unused = [];
for (const key of enUSKeys) {
  const base = baseKey(key);
  if (sourceKeys.has(key) || sourceKeys.has(base)) continue;
  if (isDynamic(key)) continue;
  unused.push(key);
}

if (unused.length > 0) {
  console.error(`Unused keys (${unused.length}) — in locale files but not in code:\n`);
  for (const key of unused.sort()) {
    error(`  - ${key}`);
  }
  console.error(
    "\nRemove from frontend/src/react/locales/ or add to DYNAMIC_PREFIXES if constructed at runtime.\n"
  );
}

// ---------------------------------------------------------------------------
// Check 3: Cross-locale consistency (all locales must match en-US key set)
// ---------------------------------------------------------------------------
for (const locale of LOCALES) {
  if (locale === "en-US") continue;
  const localeKeys = loadLocaleKeys(locale);

  const missingInLocale = [...enUSKeys].filter((k) => !localeKeys.has(k));
  const extraInLocale = [...localeKeys].filter((k) => !enUSKeys.has(k));

  if (missingInLocale.length > 0) {
    console.error(`${locale}: missing ${missingInLocale.length} key(s) (present in en-US):\n`);
    for (const key of missingInLocale.sort()) {
      error(`  - ${key}`);
    }
    console.error();
  }

  if (extraInLocale.length > 0) {
    console.error(`${locale}: extra ${extraInLocale.length} key(s) (not in en-US):\n`);
    for (const key of extraInLocale.sort()) {
      error(`  + ${key}`);
    }
    console.error();
  }
}

// ---------------------------------------------------------------------------
// Result
// ---------------------------------------------------------------------------
if (errors > 0) {
  process.exit(1);
} else {
  console.log("React i18n: all checks passed (missing keys, unused keys, cross-locale consistency).");
}
