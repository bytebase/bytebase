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
  // New format: key "bb.language", value "zh-CN" (plain string, JSON-serialized)
  const old = localStorage.getItem("bytebase_options");
  if (!old) return;

  if (localStorage.getItem(STORAGE_KEY_LANGUAGE) === null) {
    try {
      const parsed = JSON.parse(old) as {
        appearance?: { language?: string };
      };
      const lang = parsed?.appearance?.language;
      if (lang) {
        // useLocalStorage with string type stores as JSON-serialized string
        localStorage.setItem(STORAGE_KEY_LANGUAGE, JSON.stringify(lang));
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
