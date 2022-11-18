import type { editor as Editor } from "monaco-editor";
import { callVar } from "./utils";

export const getBBDarkTheme = (): Editor.IStandaloneThemeData => ({
  base: "vs-dark",
  inherit: true,
  rules: [
    {
      foreground: callVar("--color-matrix-green-hover"),
      token: "keyword",
    },
  ],
  colors: {
    "editor.foreground": callVar("--color-matrix-green"),
    "editorCursor.foreground": callVar("--color-matrix-green-hover"),
    "editorLineNumber.foreground": callVar("--color-matrix-green"),
    "editorLineNumber.activeForeground": callVar("--color-matrix-green-hover"),
  },
});
