import { maxBy } from "lodash-es";
import type { editor as Editor } from "monaco-editor";
import { unref, watchEffect } from "vue";
import type { MaybeRef } from "@/types";
import { escapeMarkdown } from "@/utils";
import { callVar } from "../themes/utils";
import type { AdviceOption } from "../types";

export const useAdvices = async (
  editor: Editor.IStandaloneCodeEditor,
  advices: MaybeRef<AdviceOption[]>
) => {
  const monaco = await import("monaco-editor");

  watchEffect((onCleanup) => {
    const _advices = unref(advices);
    const maxSeverity =
      maxBy(
        _advices.map((m) => m.severity),
        (s) => levelOfSeverity(s)
      ) ?? "WARNING";
    const decorators = editor.createDecorationsCollection(
      _advices.map((advice) => {
        return {
          range: new monaco.Range(
            advice.startLineNumber,
            advice.startColumn,
            advice.endLineNumber,
            advice.endColumn
          ),
          options: {
            className:
              maxSeverity === "ERROR" ? "squiggly-error" : "squiggly-warning",
            minimap: {
              color: { id: "minimap.errorHighlight" },
              position: monaco.editor.MinimapPosition.Inline,
            },
            overviewRuler: {
              color: { id: "editorOverviewRuler.errorForeground" },
              position: monaco.editor.OverviewRulerLane.Right,
            },
            showIfCollapsed: true,
            stickiness:
              monaco.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges,
            zIndex: 30,
            hoverMessage: {
              value: buildHoverMessage(advice),
              isTrusted: true,
            },
          },
        };
      })
    );
    onCleanup(() => {
      decorators.clear();
    });
  });
};

const buildHoverMessage = (advice: AdviceOption) => {
  const COLORS = {
    WARNING: callVar("--color-warning"),
    ERROR: callVar("--color-error"),
  };

  const { severity, message, source } = advice;
  const parts: string[] = [];
  parts.push(`<span style="color:${COLORS[severity]};">[${severity}]</span>`);
  if (source) {
    parts.push(` ${escapeMarkdown(source)}`);
  }
  parts.push(`\n${escapeMarkdown(message)}`);
  return parts.join("");
};

const levelOfSeverity = (severity: AdviceOption["severity"]) => {
  switch (severity) {
    case "WARNING":
      return 0;
    case "ERROR":
      return 1;
  }
  throw new Error(`unsupported value "${severity}"`);
};
