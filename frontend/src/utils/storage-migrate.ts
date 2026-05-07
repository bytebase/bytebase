// One-time migration for renamed localStorage keys.
// Runs before stores/components initialize so they read from the new keys.

import {
  STORAGE_KEY_AI_DISMISS,
  STORAGE_KEY_BACK_PATH,
  STORAGE_KEY_LANGUAGE,
  STORAGE_KEY_ONBOARDING,
  STORAGE_KEY_SCHEMA_EDITOR_PREVIEW,
  STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
  STORAGE_KEY_SQL_EDITOR_LAST_PROJECT,
  STORAGE_KEY_SQL_EDITOR_REDIS_NODE,
  STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT,
} from "./storage-keys";

const MIGRATION_MARKER = "bb.storage-migration-v1";

// Static key renames: [oldKey, newKey]
const STATIC_KEY_RENAMES: [string, string][] = [
  ["ui.backPath", STORAGE_KEY_BACK_PATH],
  ["bb.onboarding-state", STORAGE_KEY_ONBOARDING],
  ["bb.sql-editor.result-rows-limit", STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT],
  ["bb.sql-editor.redis-command-node", STORAGE_KEY_SQL_EDITOR_REDIS_NODE],
  ["bb.schema-editor.preview.expanded", STORAGE_KEY_SCHEMA_EDITOR_PREVIEW],
  ["bb.plugin.open-ai.dismiss-placeholder", STORAGE_KEY_AI_DISMISS],
  ["bb.plugin.editor.ai-panel-size", STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE],
  ["bb.sql-editor.last-viewed-project", STORAGE_KEY_SQL_EDITOR_LAST_PROJECT],
];

// Prefix renames: [oldPrefix, newPrefix]
// Moves all keys matching `oldPrefix.*` to `newPrefix.*`
const PREFIX_RENAMES: [string, string][] = [
  ["bb.plugin.open-ai.suggestions.", "bb.ai.suggestions."],
  ["bb.context-menu-button.", "bb.context-menu."],
];

function moveKey(oldKey: string, newKey: string) {
  const value = localStorage.getItem(oldKey);
  if (value !== null && localStorage.getItem(newKey) === null) {
    localStorage.setItem(newKey, value);
  }
  localStorage.removeItem(oldKey);
}

function migrateLanguage() {
  // Old format: key "bytebase_options", value {"appearance":{"language":"zh-CN"}}
  // New format: key "bb.language", value zh-CN (plain string)
  const old = localStorage.getItem("bytebase_options");
  if (!old) return;

  if (localStorage.getItem(STORAGE_KEY_LANGUAGE) === null) {
    try {
      const parsed = JSON.parse(old) as {
        appearance?: { language?: string };
      };
      const lang = parsed?.appearance?.language;
      if (lang) {
        localStorage.setItem(STORAGE_KEY_LANGUAGE, lang);
      }
    } catch {
      // ignore malformed
    }
  }
  localStorage.removeItem("bytebase_options");
}

function migrateStaticKeys() {
  for (const [oldKey, newKey] of STATIC_KEY_RENAMES) {
    moveKey(oldKey, newKey);
  }
}

function migratePrefixKeys() {
  for (const [oldPrefix, newPrefix] of PREFIX_RENAMES) {
    const keysToMigrate: [string, string][] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith(oldPrefix)) {
        const suffix = key.slice(oldPrefix.length);
        keysToMigrate.push([key, `${newPrefix}${suffix}`]);
      }
    }
    for (const [oldKey, newKey] of keysToMigrate) {
      moveKey(oldKey, newKey);
    }
  }
}

function migrateUIScopeKeys() {
  // Old UI state keys used different prefixes scoped by email:
  //   "ui.list.collapse.{email}" → "bb.collapse-state.{email}"
  //   "ui.intro.{email}"        → "bb.intro-state.{email}"
  //   "{email}.require_reset_password" → "bb.reset-password.{email}"
  const keysToMigrate: [string, string][] = [];
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (!key) continue;

    if (key.startsWith("ui.list.collapse.")) {
      const email = key.slice("ui.list.collapse.".length);
      keysToMigrate.push([key, `bb.collapse-state.${email}`]);
    } else if (key.startsWith("ui.intro.")) {
      const email = key.slice("ui.intro.".length);
      keysToMigrate.push([key, `bb.intro-state.${email}`]);
    } else if (key.endsWith(".require_reset_password")) {
      const email = key.slice(0, -".require_reset_password".length);
      keysToMigrate.push([key, `bb.reset-password.${email}`]);
    }
  }
  for (const [oldKey, newKey] of keysToMigrate) {
    moveKey(oldKey, newKey);
  }
}

function migrateSqlEditorTabKeys() {
  // Old: "bb.sql-editor-tab.{project}.{email}.opening-tab-list"
  //    → "bb.sql-editor.tabs.{project}.{email}"
  // Old: "bb.sql-editor-tab.{project}.{email}.current-tab-id"
  //    → "bb.sql-editor.current-tab.{project}.{email}"
  //
  // Must run here (before stores init) because useDynamicLocalStorage
  // writes default [] to the new key when it doesn't exist yet,
  // which would race with the in-store migrateTabKeys.
  const OLD_PREFIX = "bb.sql-editor-tab.";
  const TAB_LIST_SUFFIX = ".opening-tab-list";
  const CURRENT_TAB_SUFFIX = ".current-tab-id";

  const keysToMigrate: [string, string][] = [];
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (!key?.startsWith(OLD_PREFIX)) continue;

    if (key.endsWith(TAB_LIST_SUFFIX)) {
      // "bb.sql-editor-tab.{project}.{email}.opening-tab-list"
      //   → middle = "{project}.{email}"
      const middle = key.slice(OLD_PREFIX.length, -TAB_LIST_SUFFIX.length);
      keysToMigrate.push([key, `bb.sql-editor.tabs.${middle}`]);
    } else if (key.endsWith(CURRENT_TAB_SUFFIX)) {
      const middle = key.slice(OLD_PREFIX.length, -CURRENT_TAB_SUFFIX.length);
      keysToMigrate.push([key, `bb.sql-editor.current-tab.${middle}`]);
    }
  }
  for (const [oldKey, newKey] of keysToMigrate) {
    moveKey(oldKey, newKey);
  }
}

function migrateSqlEditorConnKeys() {
  // "bb.sql-editor.connection-pane.expanded_{env}.{email}"
  //   → "bb.sql-editor.conn-expanded.{env}.{email}"
  // "bb.sql-editor.connection-pane.expanded-keys.{name}"
  //   → "bb.sql-editor.conn-expanded-keys.{email}"  (name→email can't auto-migrate)
  const keysToMigrate: [string, string][] = [];
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (!key) continue;

    if (key.startsWith("bb.sql-editor.connection-pane.expanded_")) {
      // Format: bb.sql-editor.connection-pane.expanded_{env}.{email}
      const suffix = key.slice(
        "bb.sql-editor.connection-pane.expanded_".length
      );
      keysToMigrate.push([key, `bb.sql-editor.conn-expanded.${suffix}`]);
    }
  }
  for (const [oldKey, newKey] of keysToMigrate) {
    moveKey(oldKey, newKey);
  }
}

/**
 * Run all localStorage key migrations. Idempotent — skips if already done.
 * Must be called before any store or composable reads from localStorage.
 */
export function migrateStorageKeys() {
  if (localStorage.getItem(MIGRATION_MARKER)) return;

  migrateLanguage();
  migrateStaticKeys();
  migratePrefixKeys();
  migrateUIScopeKeys();
  migrateSqlEditorTabKeys();
  migrateSqlEditorConnKeys();

  localStorage.setItem(MIGRATION_MARKER, "1");
}

/**
 * Migrate user-scoped localStorage keys when email changes.
 * Renames all keys ending with `.{oldEmail}` to end with `.{newEmail}`.
 */
export function migrateUserStorage(oldEmail: string, newEmail: string) {
  if (!oldEmail || !newEmail || oldEmail === newEmail) return;

  const suffix = `.${oldEmail}`;
  const keysToMigrate: string[] = [];

  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (key?.endsWith(suffix)) {
      keysToMigrate.push(key);
    }
  }

  for (const key of keysToMigrate) {
    const newKey = key.slice(0, -suffix.length) + `.${newEmail}`;
    const value = localStorage.getItem(key);
    if (value !== null) {
      localStorage.setItem(newKey, value);
      localStorage.removeItem(key);
    }
  }
}

const UID_MIGRATION_MARKER = "bb.storage-migration-uid-to-email";

// Known bb.* prefixes that historically used a UID as the last dot-segment.
const UID_SCOPED_PREFIXES = [
  "bb.recent-visit.",
  "bb.recent-projects.",
  "bb.quick-access.",
  "bb.last-activity.",
  "bb.collapse-state.",
  "bb.intro-state.",
  "bb.iam-remind.",
  "bb.reset-password.",
  "bb.sql-editor.tabs.",
  "bb.sql-editor.current-tab.",
  "bb.sql-editor.conn-expanded.",
  "bb.sql-editor.show-missing-query-db.",
  "bb.sql-editor.worksheet-filter.",
  "bb.sql-editor.worksheet-tree.",
  "bb.sql-editor.worksheet-folder.",
  "bb.sql-editor.ai-suggestion.",
  "bb.sql-editor-tab.", // old format before centralization
  "bb.search.",
];

/**
 * Detect whether a string looks like a numeric UID (not an email).
 * UIDs are plain integers; emails always contain "@".
 */
function isNumericUID(segment: string): boolean {
  return /^\d+$/.test(segment);
}

/**
 * If both values parse as JSON arrays, merge old entries into the existing
 * array (append items from old that aren't already present by JSON identity).
 * For non-array values, the existing (email-keyed, newer) entry wins.
 */
function mergeJsonArrays(
  key: string,
  existingRaw: string,
  oldRaw: string
): void {
  try {
    const existing = JSON.parse(existingRaw);
    const old = JSON.parse(oldRaw);
    if (!Array.isArray(existing) || !Array.isArray(old)) return;

    // Build a set of serialized existing items for deduplication.
    const seen = new Set(existing.map((item) => JSON.stringify(item)));
    let merged = false;
    for (const item of old) {
      const serialized = JSON.stringify(item);
      if (!seen.has(serialized)) {
        existing.push(item);
        seen.add(serialized);
        merged = true;
      }
    }
    if (merged) {
      localStorage.setItem(key, JSON.stringify(existing));
    }
  } catch {
    // Not valid JSON or not arrays — keep the existing value.
  }
}

/**
 * Post-login migration: rename localStorage keys that used the old
 * numeric UID as user identifier to use the current user's email.
 *
 * This fixes orphaned keys created between v3.13.0 (user.name changed
 * from users/{uid} to users/{email}) and v3.15.0 (storage keys centralized).
 *
 * Must be called after `fetchCurrentUser` so the email is known.
 * Idempotent — skips if already done for this email.
 */
export function migrateUIDStorageKeys(email: string) {
  if (!email) return;
  const marker = `${UID_MIGRATION_MARKER}.${email}`;
  if (localStorage.getItem(marker)) return;

  const keysToMigrate: [string, string][] = [];

  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (!key) continue;

    for (const prefix of UID_SCOPED_PREFIXES) {
      if (!key.startsWith(prefix)) continue;

      // The key is `prefix + rest`. The UID may be the last segment or
      // appear before additional segments (e.g., `bb.sql-editor.tabs.{project}.{uid}`).
      // We scan all dot-segments in `rest` for a numeric UID and replace it with email.
      const rest = key.slice(prefix.length);
      const parts = rest.split(".");
      let replaced = false;
      for (let j = parts.length - 1; j >= 0; j--) {
        if (isNumericUID(parts[j])) {
          parts[j] = email;
          replaced = true;
          break; // only replace the last numeric segment (the UID)
        }
      }
      if (replaced) {
        const newKey = prefix + parts.join(".");
        if (newKey !== key) {
          keysToMigrate.push([key, newKey]);
        }
      }
      break; // matched a prefix, no need to check others
    }
  }

  for (const [oldKey, newKey] of keysToMigrate) {
    const oldValue = localStorage.getItem(oldKey);
    if (oldValue === null) {
      localStorage.removeItem(oldKey);
      continue;
    }
    const existingValue = localStorage.getItem(newKey);
    if (existingValue === null) {
      // No email-keyed entry yet — just move the value.
      localStorage.setItem(newKey, oldValue);
    } else {
      // Both exist — try to merge if both are JSON arrays (e.g., tabs,
      // recent projects). For non-arrays, the email-keyed entry is newer
      // and wins.
      mergeJsonArrays(newKey, existingValue, oldValue);
    }
    localStorage.removeItem(oldKey);
  }

  localStorage.setItem(marker, "1");
}
