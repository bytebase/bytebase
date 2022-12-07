import type { editor as Editor } from "monaco-editor";
import { callVar } from "./utils";

export const getBBTheme = (): Editor.IStandaloneThemeData => ({
  base: "vs",
  inherit: true,
  rules: [],
  colors: {
    "editorCursor.foreground": callVar("--color-accent"),
    "editorLineNumber.foreground": callVar("--color-control-placeholder"),
    "editorLineNumber.activeForeground": callVar("--color-main"),
  },
});
