import type monaco from "monaco-editor";
import { watchEffect } from "vue";
import type { MonacoModule } from "../types";
import { useTextModelLanguage } from "./common";

export const useSuggestOptionByLanguage = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor
) => {
  const language = useTextModelLanguage(editor);

  const defaultSuggestOption = {
    ...editor.getOption(monaco.editor.EditorOption.suggest),
  };

  watchEffect(() => {
    if (language.value === "sql") {
      editor.updateOptions({
        suggest: defaultSuggestOption,
      });
    } else {
      // Disable default auto-complete suggestions for javascript (MongoDB)
      editor.updateOptions({
        suggest: overrideAllFields(defaultSuggestOption, false),
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
