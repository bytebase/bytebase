import type { editor as Editor } from "monaco-editor";
import type { CSSProperties } from "react";
import { PRESET_BY_ID } from "./presets";
import type {
  EditorChromeColors,
  SQLEditorTheme,
  SyntaxPalette,
} from "./types";
import { SQL_EDITOR_THEME_TOKENS } from "./types";

// Monaco wants hex without the leading "#" in rule foregrounds.
const bare = (hex: string): string => hex.replace(/^#/, "");

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

// Map the normalized syntax palette onto Monaco token scopes. SQL is the
// primary language; the generic scopes cover other languages Monaco loads.
const syntaxRules = (s: SyntaxPalette): Editor.ITokenThemeRule[] => {
  const rule = (token: string, hex: string): Editor.ITokenThemeRule => ({
    token,
    foreground: bare(hex),
  });
  return [
    rule("comment", s.comment),
    rule("keyword", s.keyword),
    rule("keyword.sql", s.keyword),
    rule("string", s.string),
    rule("string.sql", s.string),
    rule("number", s.number),
    rule("number.sql", s.number),
    rule("type", s.type),
    rule("type.sql", s.type),
    rule("function", s.function),
    rule("entity.name.function", s.function),
    rule("identifier", s.variable),
    rule("predefined", s.predefined),
    rule("predefined.sql", s.predefined),
    rule("operator", s.operator),
    rule("operator.sql", s.operator),
    rule("delimiter", s.delimiter),
  ];
};

const editorColors = (e: Partial<EditorChromeColors>): Editor.IColors => {
  const colors: Editor.IColors = {
    // Keep the existing "no word highlight box" behavior from bb/bb-dark.
    "editor.wordHighlightBackground": "#00000000",
    "editor.wordHighlightStrongBackground": "#00000000",
  };
  if (e.background !== undefined) colors["editor.background"] = e.background;
  if (e.selectionBackground !== undefined)
    colors["editor.selectionBackground"] = e.selectionBackground;
  if (e.cursor !== undefined) colors["editorCursor.foreground"] = e.cursor;
  if (e.lineHighlight !== undefined)
    colors["editor.lineHighlightBackground"] = e.lineHighlight;
  if (e.gutterBackground !== undefined)
    colors["editorGutter.background"] = e.gutterBackground;
  if (e.lineNumber !== undefined)
    colors["editorLineNumber.foreground"] = e.lineNumber;
  if (e.activeLineNumber !== undefined)
    colors["editorLineNumber.activeForeground"] = e.activeLineNumber;
  return colors;
};

/** Monaco: whole theme → registered standalone theme data. */
export function buildMonacoTheme(
  theme: SQLEditorTheme
): Editor.IStandaloneThemeData {
  return {
    base: theme.monacoBase,
    inherit: true,
    rules: theme.syntax ? syntaxRules(theme.syntax) : [],
    colors: editorColors(theme.editor),
  };
}

/**
 * Admin foreground resolution: keep dark themes (those keyed off the
 * `vs-dark` Monaco base), else fall back to the dark preset.
 */
export function resolveAdminTheme(selected: SQLEditorTheme): SQLEditorTheme {
  return selected.monacoBase === "vs-dark" ? selected : PRESET_BY_ID.dark;
}

/** Throws if any chrome token / editor key / syntax key is missing. */
export function validateTheme(theme: SQLEditorTheme): void {
  for (const token of SQL_EDITOR_THEME_TOKENS) {
    if (typeof theme.tokens[token] !== "string") {
      throw new Error(`theme "${theme.id}" missing chrome token ${token}`);
    }
  }
  for (const [key, value] of Object.entries(theme.editor)) {
    if (value !== undefined && typeof value !== "string") {
      throw new Error(`theme "${theme.id}" editor.${key} must be a hex string`);
    }
  }
  if (theme.syntax) {
    const syntaxKeys: (keyof SyntaxPalette)[] = [
      "comment",
      "keyword",
      "string",
      "number",
      "type",
      "function",
      "variable",
      "operator",
      "delimiter",
      "predefined",
    ];
    for (const key of syntaxKeys) {
      if (typeof theme.syntax[key] !== "string") {
        throw new Error(`theme "${theme.id}" missing syntax.${key}`);
      }
    }
  }
}
