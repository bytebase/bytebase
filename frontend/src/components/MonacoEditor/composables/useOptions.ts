import type monaco from "monaco-editor";
import { Ref, unref, watch, watchEffect } from "vue";
import { MaybeRef } from "@/types";
import { MonacoModule } from "../editor";

export const useOptions = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  options: Ref<monaco.editor.IEditorOptions | undefined>
) => {
  watch(
    options,
    (opts) => {
      if (!opts) return;
      editor.updateOptions(opts);
    },
    { deep: true }
  );
};

export const useOptionByKey = <K extends keyof monaco.editor.IEditorOptions>(
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  key: K,
  value: MaybeRef<monaco.editor.IEditorOptions[K] | undefined>
) => {
  watchEffect(() => {
    editor.updateOptions({
      [key]: unref(value),
    });
  });
};
