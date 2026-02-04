// Centralized localStorage key registry.
// Static keys are plain constants; dynamic (user-scoped) keys are builder functions.

// --- Global (no user scope) ---
export const STORAGE_KEY_BACK_PATH = "bb.back-path";
export const STORAGE_KEY_LANGUAGE = "bb.language";
export const STORAGE_KEY_RELEASE = "bb.release";
export const STORAGE_KEY_ONBOARDING = "bb.onboarding";
export const STORAGE_KEY_SCHEMA_EDITOR_PREVIEW =
  "bb.schema-editor.preview-expanded";
export const STORAGE_KEY_ROLES_EXPIRATION =
  "bb.roles.last-expiration-selection";
export const STORAGE_KEY_AI_DISMISS = "bb.ai.dismiss-placeholder";
export const STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT = "bb.sql-editor.result-limit";
export const STORAGE_KEY_SQL_EDITOR_REDIS_NODE = "bb.sql-editor.redis-node";
export const STORAGE_KEY_SQL_EDITOR_LAST_PROJECT = "bb.sql-editor.last-project";
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
export const STORAGE_KEY_MY_ISSUES_TAB = "bb.components.MY_ISSUES.id";

// --- User-scoped (cleaned on logout) ---
export const storageKeyRecentVisit = (email: string) =>
  `bb.recent-visit.${email}`;
export const storageKeyRecentProjects = (email: string) =>
  `bb.recent-projects.${email}`;
export const storageKeyQuickAccess = (email: string) =>
  `bb.quick-access.${email}`;
export const storageKeyLastActivity = (email: string) =>
  `bb.last-activity.${email}`;
export const storageKeyCollapseState = (email: string) =>
  `bb.collapse-state.${email}`;
export const storageKeyIntroState = (email: string) =>
  `bb.intro-state.${email}`;
export const storageKeyIamRemind = (email: string) => `bb.iam-remind.${email}`;
export const storageKeyResetPassword = (email: string) =>
  `bb.reset-password.${email}`;
export const storageKeySearch = (routePath: string, email: string) =>
  `bb.search.${routePath}.${email}`;
export const storageKeyContextMenu = (key: string) => `bb.context-menu.${key}`;

// --- SQL Editor (user-scoped) ---
export const storageKeySqlEditorTabs = (project: string, email: string) =>
  `bb.sql-editor.tabs.${project}.${email}`;
export const storageKeySqlEditorCurrentTab = (project: string, email: string) =>
  `bb.sql-editor.current-tab.${project}.${email}`;
export const storageKeySqlEditorConnExpanded = (env: string, email: string) =>
  `bb.sql-editor.conn-expanded.${env}.${email}`;
export const storageKeySqlEditorConnExpandedKeys = (email: string) =>
  `bb.sql-editor.conn-expanded-keys.${email}`;
export const storageKeySqlEditorWorksheetFilter = (
  project: string,
  email: string
) => `bb.sql-editor.worksheet-filter.${project}.${email}`;
export const storageKeySqlEditorWorksheetTree = (
  project: string,
  email: string
) => `bb.sql-editor.worksheet-tree.${project}.${email}`;
export const storageKeySqlEditorWorksheetFolder = (
  project: string,
  viewMode: string,
  email: string
) => `bb.sql-editor.worksheet-folder.${project}.${viewMode}.${email}`;
export const storageKeySqlEditorAiSuggestion = (email: string) =>
  `bb.sql-editor.ai-suggestion.${email}`;

// --- AI ---
export const storageKeyAiSuggestions = (hash: string) =>
  `bb.ai.suggestions.${hash}`;

// --- PagedTable session keys ---
export const storageKeyPagedTable = (sessionKey: string, email: string) =>
  `${sessionKey}.${email}`;
