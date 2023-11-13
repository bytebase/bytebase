import { isRef, unref, watch } from "vue";
import { Language, MaybeRef } from "@/types";
import { IStandaloneCodeEditor } from "./types";

export const extensionNameOfLanguage = (lang: Language) => {
  switch (lang) {
    case "sql":
      return "sql";
    case "javascript":
      return "js";
    case "redis":
      return "redis";
  }
  // A simple fallback
  return "txt";
};

export const useEditorContextKey = <
  T extends string | number | boolean | null | undefined
>(
  editor: IStandaloneCodeEditor,
  key: string,
  valueOrRef: MaybeRef<T>
) => {
  const contextKey = editor.createContextKey<T>(key, unref(valueOrRef));
  if (isRef(valueOrRef)) {
    watch(valueOrRef, (value) => contextKey?.set(value));
  }
  return contextKey;
};
