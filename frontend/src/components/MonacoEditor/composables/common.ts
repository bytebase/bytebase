import type monaco from "monaco-editor";
import { ref } from "vue";
import { Language } from "@/types";
import { SupportedLanguages } from "../services";

export const useTextModelLanguage = (
  editor: monaco.editor.IStandaloneCodeEditor
) => {
  const language = ref(normalizeLanguage(editor.getModel()?.getLanguageId()));

  editor.onDidChangeModel(() => {
    language.value = normalizeLanguage(editor.getModel()?.getLanguageId());
  });
  editor.onDidChangeModelLanguage((e) => {
    language.value = normalizeLanguage(e.newLanguage);
  });

  return language;
};

const normalizeLanguage = (lang: string | undefined) => {
  if (!lang) return undefined;
  if (
    SupportedLanguages.findIndex((definition) => definition.id === lang) >= 0
  ) {
    return lang as Language;
  }
  return undefined;
};
