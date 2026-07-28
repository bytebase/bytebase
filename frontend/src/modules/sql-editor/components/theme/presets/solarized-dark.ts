import { DEFAULT_DARK_EDITOR_THEME, type SQLEditorTheme } from "../types";

// Ethan Schoonover's Solarized Dark palette.
//   base03 #002b36  base02 #073642  base01 #586e75  base00 #657b83
//   base0  #839496  base1  #93a1a1  base2  #eee8d5  base3  #fdf6e3
//   yellow #b58900  orange #cb4b16  red #dc322f  magenta #d33682
//   violet #6c71c4  blue #268bd2  cyan #2aa198  green #859900
// Starting values — tunable visually via the dev theme switcher.
export const solarizedDark: SQLEditorTheme = {
  id: "solarized-dark",
  name: "Solarized Dark",
  monacoBase: DEFAULT_DARK_EDITOR_THEME,
  tokens: {
    // Text bumped up the palette for readability — base01 (#586e75) is too dim
    // for UI secondary text on the dark base03 bg.
    "--color-control": "#93a1a1", // base1
    "--color-control-hover": "#cbd5d5", // lightened base1
    "--color-control-light": "#839496", // base0
    "--color-control-light-hover": "#93a1a1", // base1
    "--color-control-bg": "#073642", // base02
    "--color-control-bg-hover": "#124754",
    "--color-control-placeholder": "#657b83", // base00
    "--color-control-border": "#1e505c",
    "--color-accent": "#268bd2", // blue
    "--color-accent-hover": "#4da3de",
    "--color-accent-disabled": "#2c6482",
    "--color-accent-text": "#fdf6e3", // base3
    "--color-main": "#93a1a1", // base1
    "--color-main-hover": "#cbd5d5", // lightened base1 (brighter, not dimmer)
    "--color-main-text": "#002b36", // base03 (text on a base1 surface)
    "--color-background": "#002b36", // base03
    "--color-block-border": "#124754",
    "--color-link-hover": "#124754",
    "--color-info": "#268bd2", // blue
    "--color-info-hover": "#4da3de",
    "--color-warning": "#b58900", // yellow
    "--color-warning-hover": "#cc9900",
    "--color-error": "#dc322f", // red
    "--color-error-hover": "#e95a58",
    "--color-success": "#859900", // green
    "--color-success-hover": "#a3b81e",
    "--color-matrix-green": "#00cc00",
    "--color-matrix-green-hover": "#88ff88",
    "--color-dark-bg": "#002b36", // base03
  },
};
