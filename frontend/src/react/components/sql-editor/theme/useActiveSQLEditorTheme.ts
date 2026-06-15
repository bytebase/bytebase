import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { useSQLEditorTabState } from "@/react/stores/sqlEditor/tab";
import { resolveAdminTheme } from "./derive";
import { resolveThemeId } from "./presets";
import type { SQLEditorTheme } from "./types";

/**
 * The SQL Editor theme that should currently apply to the foreground code
 * surface: the selected theme in worksheet mode, or its admin resolution
 * (selected-if-dark-else-dark) in admin mode. Used both for the chrome's
 * Monaco theme name and to drive the global Monaco theme controller.
 */
export function useActiveSQLEditorTheme(): SQLEditorTheme {
  const themeId = useSQLEditorEditorState((s) => s.themeId);
  const mode = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.mode
  );
  const selected = resolveThemeId(themeId);
  return mode === "ADMIN" ? resolveAdminTheme(selected) : selected;
}
