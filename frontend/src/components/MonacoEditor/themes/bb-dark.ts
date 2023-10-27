import type { editor as Editor } from "monaco-editor";
import { callCssVariable } from "@/utils";

export const getBBDarkTheme = (): Editor.IStandaloneThemeData => ({
  base: "vs-dark",
  inherit: true,
  rules: [
    {
      foreground: callCssVariable("--color-matrix-green-hover"),
      token: "keyword",
    },
  ],
  colors: {
    "editor.foreground": callCssVariable("--color-matrix-green"),
    "editorCursor.foreground": callCssVariable("--color-matrix-green-hover"),
    "editorLineNumber.foreground": callCssVariable("--color-matrix-green"),
    "editorLineNumber.activeForeground": callCssVariable(
      "--color-matrix-green-hover"
    ),
  },
});
