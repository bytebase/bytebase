import type monaco from "monaco-editor";
import { Ref, unref, watch } from "vue";
import type { MaybeRef } from "@/types";
import { minmax } from "@/utils";
import type { MonacoModule } from "../types";
import { useContent } from "./useContent";

export type AutoHeightOptions = {
  min?: number;
  max?: number;
  padding?: number;
};

export const useAutoHeight = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  containerRef: Ref<HTMLElement | undefined>,
  opts: MaybeRef<AutoHeightOptions | undefined>
) => {
  const updateHeight = (height: number | undefined = undefined) => {
    const _opts = unref(opts);
    if (!_opts) return;

    const container = containerRef.value;
    if (!container) return;
    const { min, max, padding } = _opts;

    container.style.height = `${
      height ??
      minmax(
        editor.getContentHeight() + (padding ?? 0),
        min ?? 0,
        max ?? Number.MAX_SAFE_INTEGER
      )
    }px`;
  };

  const content = useContent(monaco, editor);

  watch(
    [content, () => unref(opts)],
    () => {
      if (unref(opts)) {
        updateHeight();
      }
    },
    {
      immediate: true,
    }
  );

  return updateHeight;
};
