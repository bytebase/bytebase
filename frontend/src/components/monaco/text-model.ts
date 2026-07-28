import type * as MonacoType from "monaco-editor";
import type { Language } from "@/types";
import { loadMonacoEditor } from "./core";

const textModelMapByFilename = new Map<string, MonacoType.editor.ITextModel>();
const viewStateMapByUri = new Map<
  string,
  MonacoType.editor.ICodeEditorViewState | null
>();

export const getUriByFilename = async (filename: string) => {
  const monaco = await loadMonacoEditor();
  return monaco.Uri.parse(`file:///workspace/${filename}`);
};

export const getOrCreateTextModel = async (
  filename: string,
  content: string,
  language: Language
) => {
  const existing = textModelMapByFilename.get(filename);
  if (existing) {
    return existing;
  }

  const monaco = await loadMonacoEditor();
  const uri = await getUriByFilename(filename);
  const model = monaco.editor.createModel(content, language, uri);
  textModelMapByFilename.set(filename, model);
  return model;
};

export const storeViewState = (
  editor: MonacoType.editor.IStandaloneCodeEditor,
  model: MonacoType.editor.ITextModel | null | undefined
) => {
  if (!model) return;
  viewStateMapByUri.set(model.uri.toString(), editor.saveViewState());
};

export const restoreViewState = (
  editor: MonacoType.editor.IStandaloneCodeEditor,
  model: MonacoType.editor.ITextModel | null | undefined
) => {
  if (!model) return;
  editor.restoreViewState(viewStateMapByUri.get(model.uri.toString()) ?? null);
};
