import * as monaco from "monaco-editor";
import { computed, ref, watch, type Ref, type ShallowRef } from "vue";
import type { MonacoModule } from "../types";

export const useDecoration = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  selection: ShallowRef<monaco.Selection | null>,
  activeRange: Ref<monaco.IRange | undefined | null>
) => {
  const decorationsCollection =
    ref<monaco.editor.IEditorDecorationsCollection>();

  const hasSelection = computed(() => {
    return (
      selection.value &&
      (selection.value.startLineNumber !== selection.value.endLineNumber ||
        selection.value.startColumn !== selection.value.endColumn)
    );
  });

  watch([() => selection.value, () => activeRange.value], () => {
    decorationsCollection.value?.clear();
    // Has manual selection or no active range, do not highlight.
    if (hasSelection.value || !activeRange.value) {
      return;
    }
    decorationsCollection.value = editor.createDecorationsCollection([
      {
        range: activeRange.value,
        options: {
          isWholeLine: false,
          shouldFillLineOnLineBreak: true,
          className: "bg-gray-200",
        },
      },
    ]);
  });

  return { activeRange };
};
