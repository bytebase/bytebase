import { useEffect, useRef } from "react";
import {
  getResolvedTheme,
  loadMonacoEditor,
} from "@/react/components/monaco/core";
import { useSQLEditorTabState } from "@/react/stores/sqlEditor/tab";
import { monacoThemeName, resolveAdminTheme } from "./derive";
import type { SQLEditorTheme } from "./types";
import { useWorkspaceSQLEditorTheme } from "./useWorkspaceSQLEditorTheme";

/**
 * Live-applies the workspace SQL Editor theme to Monaco's (global) theme when
 * the user switches themes while editors are already mounted.
 *
 * IMPORTANT: it must NOT call `monaco.editor.setTheme` on the initial mount.
 * Each SQL Editor Monaco instance already sets its own theme at construction via
 * its `options.theme`; calling the global `setTheme` *while an editor is still
 * constructing* races the codingame VSCode theme service and throws, which makes
 * the editor fall back to its read-only `<pre>`. So we only re-apply on a GENUINE
 * workspace-theme change (prev exists and differs). Mode-only flips (worksheet
 * ⇄ admin) remount their editor surface with the right construction theme, and
 * calling global `setTheme` during that swap would race the new editor.
 * This also makes the effect safe under React StrictMode's double-invoke — the
 * replay runs with `prev === theme` and is skipped.
 */
export function useMonacoThemeController() {
  const workspaceTheme = useWorkspaceSQLEditorTheme();
  const mode = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.mode
  );
  const prevWorkspaceThemeRef = useRef<SQLEditorTheme | null>(null);

  useEffect(() => {
    const prev = prevWorkspaceThemeRef.current;
    prevWorkspaceThemeRef.current = workspaceTheme;
    // Initial mount (prev === null), StrictMode replay, or a mode-only change:
    // editors apply the theme themselves at construction.
    if (prev === null || prev === workspaceTheme) return;

    const theme =
      mode === "ADMIN" ? resolveAdminTheme(workspaceTheme) : workspaceTheme;

    let cancelled = false;
    void loadMonacoEditor().then((monaco) => {
      if (cancelled) return;
      monaco.editor.setTheme(getResolvedTheme(monacoThemeName(theme)));
    });
    return () => {
      cancelled = true;
    };
  }, [mode, workspaceTheme]);
}
