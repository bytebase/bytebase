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

export interface SQLEditorTheme {
  id: string;
  // Literal display label, shown as-is (the theme switcher is dev-only, so
  // theme names aren't translated). Built-ins, proper-noun presets, and future
  // user-named custom themes all just set this.
  name: string;
  // The registered Monaco/VSCode color-theme id applied via
  // `monaco.editor.setTheme` (e.g. "vs", "vs-dark", "Dark Modern", "Light+"). The
  // editor canvas is themed by the codingame VSCode theme service, which owns the
  // theme registry — only ids it has registered (enumerated via
  // `getAvailableEditorThemes`) take effect; unknown ids fall back via
  // `getResolvedTheme`. The editor background still shows the chrome
  // `--color-background` through the transparent-canvas CSS; this id drives the
  // syntax token colors (and is light/dark per the chosen theme).
  monacoBase: string;
  // Chrome layer: the SQL-Editor `--color-*` token values for this theme,
  // each a "r g b" triple (Tailwind resolves to `rgb(var(--color-x) / <alpha>)`).
  tokens: Record<SQLEditorThemeToken, string>;
}

// The enumerated VSCode color-theme ids used as the light/dark defaults (the
// theme service registers these; `getAvailableEditorThemes` lists them). The
// standalone `vs`/`vs-dark` aliases are kept only as the catch-path fallback in
// `core.ts`/`editorThemes.ts` for when enumeration fails.
export const DEFAULT_LIGHT_EDITOR_THEME = "Visual Studio Light";
export const DEFAULT_DARK_EDITOR_THEME = "Visual Studio Dark";
