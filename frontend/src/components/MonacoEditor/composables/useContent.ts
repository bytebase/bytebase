import { debounce } from "lodash-es";
import * as monaco from "monaco-editor";
import { ref } from "vue";
import type { MonacoModule } from "../types";

export const useContent = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor
) => {
  const content = ref(getContent(editor));

  // Debounce content updates to reduce excessive reactive updates
  const debouncedUpdate = debounce(() => {
    content.value = getContent(editor);
  }, 50); // Short debounce to balance responsiveness and performance

  const update = () => {
    // For model changes, update immediately
    content.value = getContent(editor);
  };

  editor.onDidChangeModel(update);
  editor.onDidChangeModelContent(debouncedUpdate);

  return content;
};

const getContent = (editor: monaco.editor.IStandaloneCodeEditor) => {
  const model = editor.getModel();
  if (!model) return "";

  return model.getValue();
};
