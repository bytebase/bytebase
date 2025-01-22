import type monaco from "monaco-editor";
import { ref } from "vue";

export const useSelectedContent = (
  editor: monaco.editor.IStandaloneCodeEditor
) => {
  const selectedContent = ref(getSelectedContent(editor));
  const update = () => {
    selectedContent.value = getSelectedContent(editor);
  };

  editor.onDidChangeCursorSelection(update);
  editor.onDidChangeModel(update);
  editor.onDidChangeModelContent(update);

  return selectedContent;
};

const getSelectedContent = (editor: monaco.editor.IStandaloneCodeEditor) => {
  const model = editor.getModel();
  if (!model) return "";
  const selection = editor.getSelection();
  if (!selection) return "";

  return model.getValueInRange(selection);
};

export const getActiveContentByCursor = (
  editor: monaco.editor.IStandaloneCodeEditor
) => {
  const model = editor.getModel();
  if (!model) {
    return "";
  }
  const position = editor.getPosition();
  if (!position) {
    return "";
  }
  return model.getValueInRange({
    startLineNumber: position.lineNumber,
    startColumn: 0,
    endLineNumber: position.lineNumber,
    endColumn: editor.getBottomForLineNumber(position.lineNumber),
  });
};
