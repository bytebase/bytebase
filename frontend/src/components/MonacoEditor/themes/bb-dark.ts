import type { editor as Editor } from "monaco-editor";

export const bbDarkTheme: Editor.IStandaloneThemeData = {
  base: "vs-dark",
  inherit: true,
  rules: [
    {
      foreground: "88ff88",
      token: "keyword",
    },
  ],
  colors: {
    "editor.foreground": "#00cc00",
    "editorCursor.foreground": "#88ff88",
    "editorLineNumber.foreground": "#00cc00",
    "editorLineNumber.activeForeground": "#88ff88",
  },
};
