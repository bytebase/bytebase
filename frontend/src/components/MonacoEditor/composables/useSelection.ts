import * as monaco from "monaco-editor";
import { shallowRef } from "vue";

export const useSelection = (editor: monaco.editor.IStandaloneCodeEditor) => {
  const selection = shallowRef<monaco.Selection | null>(getSelection(editor));
  const update = () => {
    selection.value = getSelection(editor);
  };

  editor.onDidChangeCursorSelection(update);
  editor.onDidChangeModel(update);
  editor.onDidChangeModelContent(update);

  return selection;
};

const getSelection = (editor: monaco.editor.IStandaloneCodeEditor) => {
  const model = editor.getModel();
  if (!model) return null;
  const selection = editor.getSelection();
  return selection;
};
