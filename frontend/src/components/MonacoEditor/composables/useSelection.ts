import * as monaco from "monaco-editor";
import { shallowRef } from "vue";

export const useSelection = (editor: monaco.editor.IStandaloneCodeEditor) => {
  const selection = shallowRef<monaco.Selection | null>(getSelection(editor));
  const update = () => {
    selection.value = getSelection(editor);
  };

  // Only update selection when cursor selection actually changes
  editor.onDidChangeCursorSelection(update);
  editor.onDidChangeModel(update);
  // REMOVED: onDidChangeModelContent - selection doesn't change when typing
  // This was causing unnecessary updates on every keystroke

  return selection;
};

const getSelection = (editor: monaco.editor.IStandaloneCodeEditor) => {
  const model = editor.getModel();
  if (!model) return null;
  const selection = editor.getSelection();
  return selection;
};
