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
            // minimap: {
            //   color: { id: "minimap.errorHighlight" },
            //   position: monaco.editor.MinimapPosition.Inline,
            // },
            overviewRuler: opt.overviewRuler
              ? {
                  color: opt.overviewRuler.color,
                  position:
                    OverviewRulerPositionMap[opt.overviewRuler.position],
                }
              : undefined,
            // className:
            //   maxSeverity === "ERROR" ? "squiggly-error" : "squiggly-warning",
            // minimap: {
            //   color: { id: "minimap.errorHighlight" },
            //   position: monaco.editor.MinimapPosition.Inline,
            // },
            // overviewRuler: {
            //   color: { id: "editorOverviewRuler.errorForeground" },
            //   position: monaco.editor.OverviewRulerLane.Right,
            // },
            // showIfCollapsed: true,
            // stickiness:
            //   monaco.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges,
            // zIndex: 30,
            // hoverMessage: {
            //   value: buildHoverMessage(opt),
            //   isTrusted: true,
            // },
          },
        };
      })
    );
    onCleanup(() => {
      decorators.clear();
    });
  });
};
