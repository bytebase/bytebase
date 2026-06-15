// Chrome tokens are "r g b" triples (Tailwind utilities resolve to
// `rgb(var(--color-x) / <alpha>)`). Monaco values are "#rrggbb" hex.
export type RGB = string;
export type Hex = string;

// All chrome token names, for iteration/validation. Single source of truth —
// the SQLEditorThemeToken union is derived from this array so they can never
// diverge.
export const SQL_EDITOR_THEME_TOKENS = [
  "--color-control",
  "--color-control-hover",
  "--color-control-light",
  "--color-control-light-hover",
  "--color-control-bg",
  "--color-control-bg-hover",
  "--color-control-placeholder",
  "--color-control-border",
  "--color-accent",
  "--color-accent-hover",
  "--color-accent-disabled",
  "--color-accent-text",
  "--color-main",
  "--color-main-hover",
  "--color-main-text",
  "--color-background",
  "--color-block-border",
  "--color-link-hover",
  "--color-info",
  "--color-info-hover",
  "--color-warning",
  "--color-warning-hover",
  "--color-error",
  "--color-error-hover",
  "--color-success",
  "--color-success-hover",
  "--color-matrix-green",
  "--color-matrix-green-hover",
  "--color-dark-bg",
] as const;

// The SQL-Editor subset of the existing --color-* names. Every preset MUST
// fill all of these (enforced by validateTheme).
export type SQLEditorThemeToken = (typeof SQL_EDITOR_THEME_TOKENS)[number];

// Monaco editor-surface colors (canvas — unreachable by CSS cascade).
export interface EditorChromeColors {
  background: Hex;
  selectionBackground: Hex;
  cursor: Hex;
  lineHighlight: Hex;
  gutterBackground: Hex;
  lineNumber: Hex;
  activeLineNumber: Hex;
}

// Normalized, language-agnostic syntax palette. buildMonacoTheme maps these
// onto Monaco token rules (incl. SQL scopes).
export interface SyntaxPalette {
  comment: Hex;
  keyword: Hex;
  string: Hex;
  number: Hex;
  type: Hex;
  function: Hex;
  variable: Hex;
  operator: Hex;
  delimiter: Hex;
  predefined: Hex;
}

export interface SQLEditorTheme {
  id: string;
  // Literal display label, shown as-is (the theme switcher is dev-only, so
  // theme names aren't translated). Built-ins, proper-noun presets, and future
  // user-named custom themes all just set this.
  name: string;
  monacoBase: "vs" | "vs-dark";
  tokens: Record<SQLEditorThemeToken, RGB>;
  editor: Partial<EditorChromeColors>;
  syntax?: SyntaxPalette;
}
