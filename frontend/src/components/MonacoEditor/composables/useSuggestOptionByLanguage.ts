import type monaco from "monaco-editor";
import { ISuggestOptions } from "vscode/vscode/vs/editor/common/config/editorOptions";
import { watchEffect } from "vue";
import type { MonacoModule } from "../types";
import { useTextModelLanguage } from "./common";

export const useSuggestOptionByLanguage = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor
) => {
  const language = useTextModelLanguage(editor);

  const defaultSuggestOption: ISuggestOptions = {
    ...editor.getOption(monaco.editor.EditorOption.suggest),
    showStatusBar: true,
    preview: true,
  };

  watchEffect(() => {
    if (language.value === "javascript") {
      // Disable default auto-complete suggestions for javascript (MongoDB)
      editor.updateOptions({
        suggest: overrideAllFields(defaultSuggestOption, false),
      });
    } else {
      // Enable built-in auto-complete suggestions otherwise
      editor.updateOptions({
        suggest: defaultSuggestOption,
      });
    }
  });
};

const overrideAllFields = (obj: any, value: any) => {
  const updated: any = { ...obj };
  for (const key in updated) {
    updated[key] = value;
  }
  return updated;
};
