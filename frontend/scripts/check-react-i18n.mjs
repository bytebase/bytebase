// frontend/scripts/check-react-i18n.mjs
//
// Enforces strict 1:1 mapping between source code and canonical locale files:
//   1. Missing keys      — t("key") in code but key not in locale files
//   2. Unused keys       — key in locale files but not referenced in code
//   3. Consistency       — all locale files must have the exact same key set
//   4. Placeholder syntax — react-i18next uses {{name}}; flag stray Vue-style
//                           {name} placeholders left over from migration
//
// Usage: node frontend/scripts/check-react-i18n.mjs

import { readFileSync, readdirSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

// Keys whose usage the checker cannot trace statically — exempt from the
// unused-key check. Matched with String.startsWith, so an entry ending in
// "." is a prefix family (matches any key under it) and an entry without a
// trailing "." matches itself (and, technically, anything that starts with
// it — see `instance.selected-n-instances` for that convention).
const DYNAMIC_PREFIXES = [
  "dynamic.subscription.features.",
  "dynamic.subscription.purchase.features.",
  "dynamic.settings.sensitive-data.semantic-types.template.",
  "sql-review.category.",
  "sql-review.engine.",
  "sql-review.level.",
  "sql-review.rule.",
  "sql-review.template.",
  "subscription.plan.",
  "subscription.purchase.cancel-dialog.reason.",
  "settings.sensitive-data.algorithms.",
  // SQL Editor theme anchor labels, rendered via t(`…anchor.${key}`) over a
  // key list in ThemeAnchorEditor.tsx, not as literal t("…") calls.
  "settings.general.workspace.sql-editor-theme.anchor.",
  "instance.selected-n-instances",
  "settings.sidebar.",
  // Stored as messageKey string literals and translated via t(messageKey) at
  // render, not as literal t("…") calls. See
  // frontend/src/react/pages/auth/OAuthCallbackPage.tsx.
  "auth.oauth-callback.",
  // Returned from getReviewBadge as labelKey string literals, not invoked
  // via t("…") in source. See frontend/src/react/pages/project/utils/reviewBadge.ts.
  "common.bypassed",
  "common.closed",
  "common.rejected",
  "common.skipped",
  "common.under-review",
  "issue.table.approved",
  // Role-grant expiration presets, rendered via t(preset.labelKey) in
  // MembersPage.tsx (EXPIRATION_PRESETS), not as literal t("…") calls.
  "project.members.expiration-presets.",
  // Keys consumed by SHARED non-React `.ts` modules that translate via
  // `@/react/i18n` (the same react-i18next instance — vue-i18n is gone), but
  // live OUTSIDE this checker's React-only scan, so they read as unused. They
  // must stay in the canonical locale files. Listed with the shared caller
  // that uses them:
  //   - role.*.self / .description — displayRoleTitle / displayRoleTitleFromList
  //     (src/utils/role.ts, src/react/lib/role.ts)
  "role.",
  //   - project.webhook.activity-item.* — projectWebhookV1ActivityItemList
  //     (src/types/v1/projectWebhook.ts)
  "project.webhook.activity-item.",
  //   - webhook / IM type names — projectWebhookV1TypeItemList (same file)
  "common.dingtalk",
  "common.discord",
  "common.feishu",
  "common.google-chat",
  "common.lark",
  "common.slack",
  "common.teams",
  "common.wecom",
  //   - common.canceled — src/utils/accessGrant.ts
  "common.canceled",
  //   - common.no-license — src/utils/v1/instance.ts (instance display name)
  "common.no-license",
  //   - data-source.{admin,read-only} — dataSourceType (src/utils/v1/instance.ts)
  "data-source.admin",
  "data-source.read-only",
  //   - sheet.{mine,shared} — src/views/sql-editor/Sheet/context.ts
  "sheet.mine",
  "sheet.shared",
  //   - task.status.available — src/utils/v1/issue/rollout.ts
  "task.status.available",
  //   - auth.token-expired-description — src/connect/middlewares/authInterceptorMiddleware.ts
  "auth.token-expired-description",
  //   - issue.title.export-data — src/utils/v1/issue/issue.ts
  "issue.title.export-data",
];

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");
const SOURCE_DIR = resolve(ROOT, "src");
const LOCALES_DIR = resolve(ROOT, "src/locales");
const SOURCE_SCAN_DIRS = [SOURCE_DIR];
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
    if (entry.isDirectory() && !["locales", "proto-es"].includes(entry.name)) {
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

function mergeMessages(base, override) {
  const result = { ...base };
  for (const [key, value] of Object.entries(override)) {
    const existing = result[key];
    if (
      existing &&
      typeof existing === "object" &&
      !Array.isArray(existing) &&
      value &&
      typeof value === "object" &&
      !Array.isArray(value)
    ) {
      result[key] = mergeMessages(existing, value);
    } else {
      result[key] = value;
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
// Collect translation keys from source files
// ---------------------------------------------------------------------------
function collectSourceKeys() {
  const files = SOURCE_SCAN_DIRS.flatMap((dir) => [
    ...findFiles(dir, ".tsx"),
    ...findFiles(dir, ".ts"),
    ...findFiles(dir, ".vue"),
  ]);
  const keys = new Set();
  // Only match single/double quoted strings — template literals with ${} are dynamic keys.
  // This covers React `t("key")`, shared-module `t("key")`, and Vue `$t("key")`.
  const re = /\bt\(\s*["']([^"']+)["']/g;
  // <Trans i18nKey="..."> — react-i18next component-interpolation pattern
  const transRe = /\bi18nKey\s*=\s*["']([^"']+)["']/g;
  for (const file of files) {
    const src = readFileSync(file, "utf-8");
    re.lastIndex = 0;
    let m;
    while ((m = re.exec(src))) {
      keys.add(m[1]);
    }
    transRe.lastIndex = 0;
    while ((m = transRe.exec(src))) {
      keys.add(m[1]);
    }
  }
  return keys;
}

// ---------------------------------------------------------------------------
// Collect keys from a locale (main + generated/dynamic sections)
// ---------------------------------------------------------------------------
function loadLocaleKeys(locale) {
  const main = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `${locale}.json`), "utf-8")
  );
  const dynamic = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `dynamic/${locale}.json`), "utf-8")
  );
  const sqlReview = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `sql-review/${locale}.json`), "utf-8")
  );
  const subscription = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `subscription/${locale}.json`), "utf-8")
  );
  const sections = mergeMessages(
    {
      "sql-review": sqlReview,
      subscription,
    },
    main
  );
  return new Set(
    Object.keys(
      flatten({
        ...sections,
        dynamic,
      })
    )
  );
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
    "\nRemove from frontend/src/locales/ or add to DYNAMIC_PREFIXES if referenced indirectly (helper return, template literal).\n"
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
// Check 4: Vue-style {name} placeholders (must be {{name}} for react-i18next)
// ---------------------------------------------------------------------------
// react-i18next interpolates {{name}}. A bare {name} (not part of {{name}})
// is a left-over from Vue's vue-i18n syntax and renders literally — see
// the bug fixed alongside this check.
const SINGLE_BRACE_RE = /(?<!\{)\{([a-zA-Z_][a-zA-Z0-9_]*)\}(?!\})/g;

function findSingleBracePlaceholders(obj, path = "") {
  const issues = [];
  if (obj && typeof obj === "object" && !Array.isArray(obj)) {
    for (const [k, v] of Object.entries(obj)) {
      issues.push(...findSingleBracePlaceholders(v, path ? `${path}.${k}` : k));
    }
  } else if (typeof obj === "string") {
    SINGLE_BRACE_RE.lastIndex = 0;
    const names = [];
    let m;
    while ((m = SINGLE_BRACE_RE.exec(obj))) names.push(m[1]);
    if (names.length > 0) issues.push({ key: path, value: obj, names });
  }
  return issues;
}

for (const locale of LOCALES) {
  const main = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `${locale}.json`), "utf-8")
  );
  const dynamic = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `dynamic/${locale}.json`), "utf-8")
  );
  const sqlReview = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `sql-review/${locale}.json`), "utf-8")
  );
  const subscription = JSON.parse(
    readFileSync(resolve(LOCALES_DIR, `subscription/${locale}.json`), "utf-8")
  );
  const issues = [
    ...findSingleBracePlaceholders(main),
    ...findSingleBracePlaceholders(dynamic, "dynamic"),
    ...findSingleBracePlaceholders(sqlReview, "sql-review"),
    ...findSingleBracePlaceholders(subscription, "subscription"),
  ];
  if (issues.length > 0) {
    console.error(
      `${locale}: ${issues.length} string(s) with Vue-style {name} placeholders — react-i18next needs {{name}}:\n`
    );
    for (const { key, value, names } of issues) {
      error(`  - ${key} → ${JSON.stringify(value)} (placeholders: ${names.join(", ")})`);
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
  console.log("React i18n: all checks passed (missing keys, unused keys, cross-locale consistency, placeholder syntax).");
}
