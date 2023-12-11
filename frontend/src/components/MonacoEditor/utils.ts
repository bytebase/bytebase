import { Range } from "monaco-editor";
import { isRef, unref, watch } from "vue";
import type { Language, MaybeRef, SQLDialect } from "@/types";
import sqlFormatter from "./sqlFormatter";
import type { IStandaloneCodeEditor } from "./types";

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
  console.warn("unexpected language", lang);
  return "sql";
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

export const trySetContentWithUndo = (
  editor: IStandaloneCodeEditor,
  content: string,
  source: string | undefined = undefined
) => {
  editor.executeEdits(source, [
    {
      range: new Range(1, 1, Number.MAX_SAFE_INTEGER, 1),
      text: "",
      forceMoveMarkers: true,
    },
    {
      range: new Range(1, 1, 1, 1),
      text: content,
      forceMoveMarkers: true,
    },
  ]);
};

export const formatEditorContent = async (
  editor: IStandaloneCodeEditor,
  dialect: SQLDialect | undefined
) => {
  const model = editor.getModel();
  if (!model) return;
  const sql = model.getValue();
  const { data, error } = await sqlFormatter(sql, dialect);
  if (error) {
    return;
  }
  const pos = editor.getPosition();

  trySetContentWithUndo(editor, data, "Format content");

  if (pos) {
    // Not that smart but best efforts to keep the cursor position
    editor.setPosition(pos);
  }
};
