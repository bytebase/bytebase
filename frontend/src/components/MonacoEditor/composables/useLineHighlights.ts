import type monaco from "monaco-editor";
import { unref, watchEffect } from "vue";
import type { MaybeRef } from "@/types";
import type { LineHighlightOption, MonacoModule } from "../types";

export const useLineHighlights = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  options: MaybeRef<LineHighlightOption[]>
) => {
  const OverviewRulerPositionMap = {
    LEFT: monaco.editor.OverviewRulerLane.Left,
    RIGHT: monaco.editor.OverviewRulerLane.Right,
    CENTER: monaco.editor.OverviewRulerLane.Center,
    FULL: monaco.editor.OverviewRulerLane.Full,
  };

  watchEffect((onCleanup) => {
    const opts = unref(options);
    const decorators = editor.createDecorationsCollection(
      opts.map((opt) => {
        return {
          range: new monaco.Range(opt.lineNumber, 1, opt.lineNumber, Infinity),
          options: {
            isWholeLine: true,
            inlineClassName: opt.className,
            overviewRuler: opt.overviewRuler
              ? {
                  color: opt.overviewRuler.color,
                  position:
                    OverviewRulerPositionMap[opt.overviewRuler.position],
                }
              : undefined,
            stickiness:
              monaco.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges,
          },
        };
      })
    );
    onCleanup(() => {
      decorators.clear();
    });
  });
};
