import type { editor as Editor } from "monaco-editor";
import { callVar } from "./utils";

export const bbDarkTheme: Editor.IStandaloneThemeData = {
  base: "vs-dark",
  inherit: true,
  rules: [
    {
      foreground: callVar("--color-dark-green-hover"),
      token: "keyword",
    },
  ],
  colors: {
    "editor.foreground": callVar("--color-dark-green"),
    "editorCursor.foreground": callVar("--color-dark-green-hover"),
    "editorLineNumber.foreground": callVar("--color-dark-green"),
    "editorLineNumber.activeForeground": callVar("--color-dark-green-hover"),
  },
};
