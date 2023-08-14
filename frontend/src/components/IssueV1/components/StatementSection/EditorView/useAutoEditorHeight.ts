import { ref } from "vue";
import MonacoEditor from "@/components/MonacoEditor";

const EDITOR_MIN_HEIGHT = 120; // ~= 6 lines, a reasonable size to start writing SQL

export const useAutoEditorHeight = () => {
  const editorRef = ref<InstanceType<typeof MonacoEditor>>();
  const updateEditorHeight = () => {
    requestAnimationFrame(() => {
      const contentHeight =
        editorRef.value?.editorInstance?.getContentHeight() as number;
      let actualHeight = contentHeight;
      if (actualHeight < EDITOR_MIN_HEIGHT) {
        actualHeight = EDITOR_MIN_HEIGHT;
      }
      editorRef.value?.setEditorContentHeight(actualHeight);
    });
  };

  return {
    editorRef,
    updateEditorHeight,
  };
};
