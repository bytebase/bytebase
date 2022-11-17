import type { editor as Editor } from "monaco-editor";

export const bbTheme: Editor.IStandaloneThemeData = {
  base: "vs",
  inherit: true,
  rules: [],
  colors: {
    "editorCursor.foreground": "#504de2",
    "editorLineNumber.foreground": "#aaaaaa",
    "editorLineNumber.activeForeground": "#111111",
  },
};
