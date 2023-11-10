import type monaco from "monaco-editor";
import { Ref, watch } from "vue";
import { MonacoModule } from "../editor";

// Store ViewState (e.g., selection and scroll position) for each TextModel
export const ViewStateMapByUri = new Map<
  string,
  monaco.editor.ICodeEditorViewState | null
>();

export const useModel = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  model: Ref<monaco.editor.ITextModel | undefined>
) => {
  watch(
    model,
    (newModel, oldModel) => {
      if (oldModel) {
        // Save ViewState for oldModel
        const uri = oldModel.uri.toString();
        const vs = editor.saveViewState();
        ViewStateMapByUri.set(uri, vs);
      }

      editor.setModel(newModel ?? null);

      if (newModel) {
        // Restore ViewState for newModel
        const uri = newModel.uri.toString();
        const vs = ViewStateMapByUri.get(uri);
        editor.restoreViewState(vs ?? null);
      }
    },
    { immediate: true }
  );
};
