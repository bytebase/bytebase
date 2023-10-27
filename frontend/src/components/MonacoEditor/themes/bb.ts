import type { editor as Editor } from "monaco-editor";
import { callCssVariable } from "@/utils";

export const getBBTheme = (): Editor.IStandaloneThemeData => ({
  base: "vs",
  inherit: true,
  rules: [],
  colors: {
    "editorCursor.foreground": callCssVariable("--color-accent"),
    "editorLineNumber.foreground": callCssVariable(
      "--color-control-placeholder"
    ),
    "editorLineNumber.activeForeground": callCssVariable("--color-main"),
  },
});
