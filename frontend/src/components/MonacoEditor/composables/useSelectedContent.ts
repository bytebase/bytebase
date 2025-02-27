import type monaco from "monaco-editor";
import { ref, type ShallowRef, watchEffect } from "vue";

export const useSelectedContent = (
  editor: monaco.editor.IStandaloneCodeEditor,
  selection: ShallowRef<monaco.Selection | null>
) => {
  const selectedContent = ref<string>("");

  watchEffect(() => {
    const model = editor.getModel();
    if (selection.value && model) {
      selectedContent.value = model.getValueInRange(selection.value);
    }
  });

  return selectedContent;
};
