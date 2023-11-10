import type monaco from "monaco-editor";
import { ref } from "vue";
import { MonacoModule } from "../editor";

export const useSelectedContent = (
  monaco: MonacoModule,
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
