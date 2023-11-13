import type monaco from "monaco-editor";
import { Ref, unref, watch, watchEffect } from "vue";
import type { MaybeRef } from "@/types";
import type { MonacoModule } from "../types";
import { EditorType } from "./common";

type OptionsType<E> = E extends monaco.editor.IStandaloneCodeEditor
  ? monaco.editor.IEditorOptions
  : E extends monaco.editor.IStandaloneDiffEditor
  ? monaco.editor.IDiffEditorOptions
  : never;

export const useOptions = <E extends EditorType>(
  monaco: MonacoModule,
  editor: E,
  options: Ref<OptionsType<E> | undefined>
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

export const useOptionByKey = <
  E extends EditorType,
  K extends keyof OptionsType<E>
>(
  monaco: MonacoModule,
  editor: E,
  key: K,
  value: MaybeRef<OptionsType<E>[K] | undefined>
) => {
  watchEffect(() => {
    editor.updateOptions({
      [key]: unref(value),
    });
  });
};
