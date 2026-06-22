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
    "--color-control": "147 161 161", // base1
    "--color-control-hover": "203 213 213", // lightened base1
    "--color-control-light": "131 148 150", // base0
    "--color-control-light-hover": "147 161 161", // base1
    "--color-control-bg": "7 54 66", // base02
    "--color-control-bg-hover": "18 71 84",
    "--color-control-placeholder": "101 123 131", // base00
    "--color-control-border": "30 80 92",
    "--color-accent": "38 139 210", // blue
    "--color-accent-hover": "77 163 222",
    "--color-accent-disabled": "44 100 130",
    "--color-accent-text": "253 246 227", // base3
    "--color-main": "147 161 161", // base1
    "--color-main-hover": "203 213 213", // lightened base1 (brighter, not dimmer)
    "--color-main-text": "0 43 54", // base03 (text on a base1 surface)
    "--color-background": "0 43 54", // base03
    "--color-block-border": "18 71 84",
    "--color-link-hover": "18 71 84",
    "--color-info": "38 139 210", // blue
    "--color-info-hover": "77 163 222",
    "--color-warning": "181 137 0", // yellow
    "--color-warning-hover": "204 153 0",
    "--color-error": "220 50 47", // red
    "--color-error-hover": "233 90 88",
    "--color-success": "133 153 0", // green
    "--color-success-hover": "163 184 30",
    "--color-matrix-green": "0 204 0",
    "--color-matrix-green-hover": "136 255 136",
    "--color-dark-bg": "0 43 54", // base03
  },
};
