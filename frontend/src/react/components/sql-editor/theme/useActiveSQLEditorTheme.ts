import { useSQLEditorTabState } from "@/react/stores/sqlEditor/tab";
import { resolveAdminTheme } from "./derive";
import type { SQLEditorTheme } from "./types";
import { useWorkspaceSQLEditorTheme } from "./useWorkspaceSQLEditorTheme";

/**
 * The SQL Editor theme that should currently apply to the foreground code
 * surface: the selected theme in worksheet mode, or its admin resolution
 * (selected-if-dark-else-dark) in admin mode. Used both for the chrome's
 * Monaco theme name and to drive the global Monaco theme controller.
 */
export function useActiveSQLEditorTheme(): SQLEditorTheme {
  const selected = useWorkspaceSQLEditorTheme();
  const mode = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.mode
  );
  return mode === "ADMIN" ? resolveAdminTheme(selected) : selected;
}
