// Centralized localStorage key registry.
// Static keys are plain constants; dynamic (user-scoped) keys are builder functions.

// Workspace cache scope. Self-host installs have a single workspace, so their
// keys stay workspace-agnostic — existing cached data is preserved with no
// migration. SaaS installs scope by workspace id so switching workspace (same
// email) never reads another workspace's cached resource references (project /
// database / environment names and route paths are unique only within a
// workspace). Returns "" for self-host, the workspace name for SaaS.
export const workspaceCacheScope = (
  isSaaS: boolean,
  workspace: string
): string => (isSaaS && workspace ? workspace : "");

// Joins a base key with an optional workspace scope segment and the rest.
// When scope is "" the key is byte-identical to the pre-scoping form (so
// self-host caches are untouched). `email` stays the trailing segment so the
// email-change migration (migrateUserStorage, renames by `.{email}` suffix)
// keeps working.
const withScope = (base: string, scope: string, ...rest: string[]): string =>
  [base, ...(scope ? [scope] : []), ...rest].join(".");

// --- Global (no user scope) ---
export const STORAGE_KEY_BACK_PATH = "bb.back-path";
export const STORAGE_KEY_LANGUAGE = "bb.language";
export const STORAGE_KEY_ONBOARDING = "bb.onboarding";
export const STORAGE_KEY_SCHEMA_EDITOR_PREVIEW =
  "bb.schema-editor.preview-expanded";
export const STORAGE_KEY_AI_DISMISS = "bb.ai.dismiss-placeholder";
export const STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT = "bb.sql-editor.result-limit";
export const STORAGE_KEY_SQL_EDITOR_REDIS_NODE = "bb.sql-editor.redis-node";
export const STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE =
  "bb.sql-editor.ai-panel-size";
export const STORAGE_KEY_SQL_EDITOR_SIDEBAR_TAB =
  "bb.sql-editor.sidebar.last-visited-tab";
export const STORAGE_KEY_SQL_EDITOR_CODE_VIEWER_FORMAT =
  "bb.sql-editor.editor-panel.code-viewer.format";
export const STORAGE_KEY_SQL_EDITOR_DETAIL_FORMAT =
  "bb.sql-editor.detail-panel.format";
export const STORAGE_KEY_SQL_EDITOR_DETAIL_LINE_WRAP =
  "bb.sql-editor.detail-panel.line-wrap";
export const STORAGE_KEY_SQL_EDITOR_NOSQL_TABLE_VIEW =
  "bb.sql-editor.es-table-view";
export const STORAGE_KEY_MY_ISSUES_TAB = "bb.components.MY_ISSUES.id";

// --- User-scoped (cleaned on logout) ---
// Workspace-scoped: holds route paths / project names that are workspace-local.
export const storageKeyRecentVisit = (scope: string, email: string) =>
  withScope("bb.recent-visit", scope, email);
export const storageKeyRecentProjects = (scope: string, email: string) =>
  withScope("bb.recent-projects", scope, email);
export const storageKeyQuickAccess = (email: string) =>
  `bb.quick-access.${email}`;
export const storageKeyLastActivity = (email: string) =>
  `bb.last-activity.${email}`;
export const storageKeyCollapseState = (email: string) =>
  `bb.collapse-state.${email}`;
export const storageKeyIntroState = (email: string) =>
  `bb.intro-state.${email}`;
// Workspace-scoped: value keys embed project resource names.
export const storageKeyIamRemind = (scope: string, email: string) =>
  withScope("bb.iam-remind", scope, email);
export const storageKeyResetPassword = (email: string) =>
  `bb.reset-password.${email}`;
export const storageKeyContextMenu = (key: string) => `bb.context-menu.${key}`;

// --- SQL Editor (user-scoped) ---
// Workspace-scoped: keys carry project / environment resource ids, which are
// unique only within a workspace (so they collide across workspaces in SaaS).
export const storageKeySqlEditorLastProject = (scope: string) =>
  withScope("bb.sql-editor.last-project", scope);
export const storageKeySqlEditorTabs = (
  scope: string,
  project: string,
  email: string
) => withScope("bb.sql-editor.tabs", scope, project, email);
export const storageKeySqlEditorCurrentTab = (
  scope: string,
  project: string,
  email: string
) => withScope("bb.sql-editor.current-tab", scope, project, email);
export const storageKeySqlEditorConnExpanded = (
  scope: string,
  env: string,
  email: string
) => withScope("bb.sql-editor.conn-expanded", scope, env, email);
export const storageKeySqlEditorShowMissingQueryDb = (email: string) =>
  `bb.sql-editor.show-missing-query-db.${email}`;
export const storageKeySqlEditorWorksheetFilter = (
  scope: string,
  project: string,
  email: string
) => withScope("bb.sql-editor.worksheet-filter", scope, project, email);
export const storageKeySqlEditorWorksheetTree = (
  scope: string,
  project: string,
  email: string
) => withScope("bb.sql-editor.worksheet-tree", scope, project, email);
export const storageKeySqlEditorWorksheetFolder = (
  scope: string,
  project: string,
  viewMode: string,
  email: string
) =>
  withScope("bb.sql-editor.worksheet-folder", scope, project, viewMode, email);
export const storageKeySqlEditorAiSuggestion = (email: string) =>
  `bb.sql-editor.ai-suggestion.${email}`;

// --- AI ---
export const storageKeyAiSuggestions = (hash: string) =>
  `bb.ai.suggestions.${hash}`;

// --- PagedTable session keys ---
export const storageKeyPagedTable = (sessionKey: string, email: string) =>
  `${sessionKey}.${email}`;
