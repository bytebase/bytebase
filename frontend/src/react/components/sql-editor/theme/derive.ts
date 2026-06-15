import type { editor as Editor } from "monaco-editor";
import type { CSSProperties } from "react";
import { PRESET_BY_ID } from "./presets";
import type { SQLEditorTheme } from "./types";
import { SQL_EDITOR_THEME_TOKENS } from "./types";

/** Chrome: tokens → inline CSS custom properties for a container's `style`. */
export function themeToCssVars(
  tokens: SQLEditorTheme["tokens"]
): CSSProperties & Record<string, string> {
  const vars: Record<string, string> = {};
  for (const token of SQL_EDITOR_THEME_TOKENS) {
    vars[token] = tokens[token];
  }
  return vars as CSSProperties & Record<string, string>;
}

/** Monaco theme name registered via defineTheme + used by setTheme. */
export function monacoThemeName(theme: SQLEditorTheme): string {
  return `bb-${theme.id}`;
}

/**
 * Monaco: a theme → registered standalone theme data. The editor canvas is
 * owned by the codingame VSCode theme service (which ignores per-theme colors
 * passed to `defineTheme`), so this only carries the base (`vs`/`vs-dark`) plus
 * the long-standing "no word-highlight box" reset. The editor background comes
 * from the chrome `--color-background` via the transparent-canvas CSS; syntax
 * token colors come from the base.
 */
export function buildMonacoTheme(
  theme: SQLEditorTheme
): Editor.IStandaloneThemeData {
  return {
    base: theme.monacoBase,
    inherit: true,
    rules: [],
    colors: {
      "editor.wordHighlightBackground": "#00000000",
      "editor.wordHighlightStrongBackground": "#00000000",
    },
  };
}

/**
 * Admin foreground resolution: keep dark themes (those keyed off the
 * `vs-dark` Monaco base), else fall back to the dark preset.
 */
export function resolveAdminTheme(selected: SQLEditorTheme): SQLEditorTheme {
  return selected.monacoBase === "vs-dark" ? selected : PRESET_BY_ID.dark;
}

/** Throws if any chrome token is missing. */
export function validateTheme(theme: SQLEditorTheme): void {
  for (const token of SQL_EDITOR_THEME_TOKENS) {
    if (typeof theme.tokens[token] !== "string") {
      throw new Error(`theme "${theme.id}" missing chrome token ${token}`);
    }
  }
}
