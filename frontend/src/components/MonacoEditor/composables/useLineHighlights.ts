import * as monaco from "monaco-editor";
import { unref, watchEffect } from "vue";
import type { MaybeRef } from "@/types";
import type { LineHighlightOption, MonacoModule } from "../types";

export const useLineHighlights = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  options: MaybeRef<LineHighlightOption[]>
) => {
  watchEffect((onCleanup) => {
    const opts = unref(options);
    requestAnimationFrame(() => {
      const decorators = editor.createDecorationsCollection(
        opts.map((opt) => {
          return {
            range: new monaco.Range(
              opt.startLineNumber,
              1,
              opt.endLineNumber,
              Infinity
            ),
            options: {
              ...opt.options,
              blockPadding: [3, 3, 3, 3],
              stickiness:
                monaco.editor.TrackedRangeStickiness
                  .AlwaysGrowsWhenTypingAtEdges,
            },
          };
        })
      );
      onCleanup(() => {
        decorators.clear();
      });
    });
  });
};
