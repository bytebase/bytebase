import type { CSSProperties } from "react";
// Import the `dark` preset DIRECTLY (a plain token object), NOT the `./presets`
// barrel — the barrel may pull in presets *derived* from this file (via
// `deriveThemeFromAnchors`), which would create an init-order cycle (TDZ on the
// helper consts below).
import { dark } from "./presets/dark";
import {
  DEFAULT_DARK_EDITOR_THEME,
  DEFAULT_LIGHT_EDITOR_THEME,
  SQL_EDITOR_THEME_TOKENS,
  type SQLEditorTheme,
} from "./types";

/** Chrome: tokens → inline CSS custom properties for a container's `style`. */
export function themeToCssVars(
  tokens: SQLEditorTheme["tokens"]
): CSSProperties & Record<string, string> {
  const vars: Record<string, string> = {};
  for (const token of SQL_EDITOR_THEME_TOKENS) {
    vars[token] = hexToRgbString(tokens[token]);
  }
  return vars as CSSProperties & Record<string, string>;
}

/**
 * The registered Monaco/VSCode theme id to apply via `monaco.editor.setTheme`
 * for this theme — i.e. its `monacoBase` (e.g. "vs-dark", "Dark Modern"). Kept
 * as a function so call sites read intent ("the editor theme to apply") rather
 * than poking the field.
 */
export function monacoThemeName(theme: SQLEditorTheme): string {
  return theme.monacoBase;
}

/** Whether the theme reads as dark (by its chrome background luminance). */
export function isDarkTheme(theme: SQLEditorTheme): boolean {
  return luminance(hexToRgb(theme.tokens["--color-background"])) < 0.5;
}

/**
 * Admin foreground resolution: keep a dark theme, else fall back to the dark
 * preset (admin mode is always dark).
 */
export function resolveAdminTheme(selected: SQLEditorTheme): SQLEditorTheme {
  return isDarkTheme(selected) ? selected : dark;
}

/**
 * The CSS `color-scheme` for a theme. Apply it (`style.colorScheme`) alongside
 * the token vars so the platform renders NATIVE controls — date pickers,
 * scrollbars, `<select>` popups — to match. These are drawn by the browser/OS
 * and can't be styled with our `--color-*` tokens; `color-scheme` is the only
 * lever. Keyed off the Monaco base so a dark theme reports `dark`.
 */
export function themeColorScheme(theme: SQLEditorTheme): "dark" | "light" {
  return isDarkTheme(theme) ? "dark" : "light";
}

/** Throws if any chrome token is missing. */
export function validateTheme(theme: SQLEditorTheme): void {
  for (const token of SQL_EDITOR_THEME_TOKENS) {
    const value = theme.tokens[token];
    if (typeof value !== "string" || !isHexColor(value)) {
      throw new TypeError(`theme "${theme.id}" missing chrome token ${token}`);
    }
  }
}

/**
 * The 4 colors an admin picks (each a `#rrggbb` hex). We derive all 29 chrome
 * tokens from these — the elevated "surface" is the background nudged toward the
 * text, status + matrix-green are held fixed.
 */
export interface ThemeAnchors {
  background: string;
  text: string;
  accent: string;
  border: string;
}

// Semantic status + matrix-green tokens — constant across brands so they stay
// recognizable (warning=amber, error=red, success=green). Inlined (the light
// preset's values) so `deriveThemeFromAnchors` has no dependency on the preset
// catalog. NOTE: `--color-info` is intentionally NOT here — it's derived from
// the accent (the least-semantic "brand" status) so info surfaces match the
// theme instead of clashing with a fixed blue.
const FIXED_TOKEN_VALUES: Record<string, string> = {
  "--color-warning": "#f59e0b",
  "--color-warning-hover": "#b45309",
  "--color-error": "#dc2626",
  "--color-error-hover": "#b91c1c",
  "--color-success": "#16a34a",
  "--color-success-hover": "#15803d",
  "--color-matrix-green": "#00cc00",
  "--color-matrix-green-hover": "#88ff88",
};

type RGB3 = [number, number, number];
const isHexColor = (value: string): boolean => /^#[\da-fA-F]{6}$/.test(value);
const hexToRgb = (h: string): RGB3 => {
  const x = h.replace("#", "");
  return [0, 2, 4].map((i) => Number.parseInt(x.slice(i, i + 2), 16)) as RGB3;
};
const normalizeRgb = (c: RGB3): RGB3 =>
  c.map((n) => Math.max(0, Math.min(255, Math.round(n)))) as RGB3;
const toHex = (c: RGB3): string =>
  "#" +
  normalizeRgb(c)
    .map((n) => n.toString(16).padStart(2, "0"))
    .join("");
const hexToRgbString = (h: string): string =>
  normalizeRgb(hexToRgb(h)).join(" ");
const mix = (a: RGB3, b: RGB3, t: number): RGB3 =>
  a.map((v, i) => v + (b[i] - v) * t) as RGB3;
const luminance = ([r, g, b]: RGB3): number =>
  (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;

/**
 * Derive a complete theme from 5 anchor colors. The 5 anchors map directly to
 * background/control-bg/main/accent/block-border; the rest are blended toward
 * the text or background to keep contrast. Status + matrix tokens are fixed.
 * `monacoBase` defaults to the background luminance (dark background →
 * `vs-dark`), but callers may pass an explicit base to override that — the
 * editor's syntax palette (light/dark) is then admin-chosen, not inferred.
 */
export function deriveThemeFromAnchors(
  anchors: ThemeAnchors,
  name: string,
  monacoBase?: SQLEditorTheme["monacoBase"]
): SQLEditorTheme {
  const bg = hexToRgb(anchors.background);
  const text = hexToRgb(anchors.text);
  const accent = hexToRgb(anchors.accent);
  const border = hexToRgb(anchors.border);
  const dark = luminance(bg) < 0.5;
  const elevate = (c: RGB3, t: number) => mix(c, text, t); // toward text (more contrast)
  const recede = (c: RGB3, t: number) => mix(c, bg, t); // toward background

  // Elevated surface (panels/headers/hover) — the background nudged toward the
  // text. Derived (not an anchor) so the admin only picks 4 colors.
  const surface = elevate(bg, 0.06);
  const accentHex = toHex(accent);
  const accentHover = toHex(dark ? elevate(accent, 0.15) : recede(accent, 0.2));

  const tokens: Record<string, string> = {
    "--color-background": toHex(bg),
    "--color-dark-bg": toHex(bg),
    "--color-control-bg": toHex(surface),
    "--color-control-bg-hover": toHex(elevate(surface, 0.06)),
    "--color-control": toHex(text),
    "--color-control-hover": toHex(recede(text, 0.15)),
    "--color-control-light": toHex(recede(text, 0.3)),
    "--color-control-light-hover": toHex(recede(text, 0.15)),
    "--color-control-placeholder": toHex(recede(text, 0.5)),
    "--color-control-border": toHex(border),
    "--color-block-border": toHex(border),
    "--color-link-hover": toHex(border),
    "--color-accent": accentHex,
    "--color-accent-hover": accentHover,
    "--color-accent-disabled": toHex(recede(accent, 0.5)),
    // On-accent text (e.g. the Run button label): use whichever of the theme's
    // Text / Background anchors contrasts better with the accent — so it follows
    // the theme's own colors while staying legible on the accent fill.
    "--color-accent-text":
      Math.abs(luminance(bg) - luminance(accent)) >=
      Math.abs(luminance(text) - luminance(accent))
        ? toHex(bg)
        : toHex(text),
    // Info mirrors the brand accent (warning/error/success stay semantic).
    "--color-info": accentHex,
    "--color-info-hover": accentHover,
    "--color-main": toHex(text),
    "--color-main-hover": toHex(recede(text, 0.2)),
    "--color-main-text": luminance(text) < 0.5 ? "#ffffff" : "#18181b",
  };
  Object.assign(tokens, FIXED_TOKEN_VALUES);

  const theme: SQLEditorTheme = {
    id: "",
    name,
    monacoBase:
      monacoBase ??
      (dark ? DEFAULT_DARK_EDITOR_THEME : DEFAULT_LIGHT_EDITOR_THEME),
    tokens,
  };
  validateTheme(theme); // throws if a key is missing
  return theme;
}

/** Inverse of {@link deriveThemeFromAnchors}: recover the 4 picked anchors. */
export function themeToAnchors(theme: SQLEditorTheme): ThemeAnchors {
  return {
    background: theme.tokens["--color-background"],
    text: theme.tokens["--color-main"],
    accent: theme.tokens["--color-accent"],
    border: theme.tokens["--color-block-border"],
  };
}
