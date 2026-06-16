import { useEffect, useRef } from "react";
import {
  getResolvedTheme,
  loadMonacoEditor,
} from "@/react/components/monaco/core";
import { monacoThemeName } from "./derive";
import type { SQLEditorTheme } from "./types";
import { useActiveSQLEditorTheme } from "./useActiveSQLEditorTheme";

/**
 * Live-applies the active SQL Editor theme to Monaco's (global) theme when the
 * user switches themes while editors are already mounted.
 *
 * IMPORTANT: it must NOT call `monaco.editor.setTheme` on the initial mount.
 * Each SQL Editor Monaco instance already sets its own theme at construction via
 * its `options.theme`; calling the global `setTheme` *while an editor is still
 * constructing* races the codingame VSCode theme service and throws, which makes
 * the editor fall back to its read-only `<pre>`. So we only re-apply on a GENUINE
 * theme change (prev exists and differs). This also makes the effect safe under
 * React StrictMode's double-invoke — the replay runs with `prev === theme` and is
 * skipped.
 */
export function useMonacoThemeController() {
  const theme = useActiveSQLEditorTheme();
  const prevThemeRef = useRef<SQLEditorTheme | null>(null);

  useEffect(() => {
    const prev = prevThemeRef.current;
    prevThemeRef.current = theme;
    // Initial mount (prev === null) or StrictMode replay / no real change
    // (prev === theme): editors apply the theme themselves at construction.
    if (prev === null || prev === theme) return;

    let cancelled = false;
    void loadMonacoEditor().then((monaco) => {
      if (cancelled) return;
      monaco.editor.setTheme(getResolvedTheme(monacoThemeName(theme)));
    });
    return () => {
      cancelled = true;
    };
  }, [theme]);
}
