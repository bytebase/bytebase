import { create } from "@bufbuild/protobuf";
import { maxBy } from "lodash-es";
import * as monaco from "monaco-editor";
import { unref, watchEffect } from "vue";
import type { MaybeRef } from "@/types";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import { callCssVariable, escapeMarkdown } from "@/utils";
import { batchConvertPositionToMonacoPosition } from "@/utils/v1/position";
import type { AdviceOption, MonacoModule } from "../types";

export const useAdvices = (
  monaco: MonacoModule,
  editor: monaco.editor.IStandaloneCodeEditor,
  advices: MaybeRef<AdviceOption[]>
) => {
  watchEffect((onCleanup) => {
    const _advices = unref(advices);
    const maxSeverity =
      maxBy(
        _advices.map((m) => m.severity),
        (s) => levelOfSeverity(s)
      ) ?? "WARNING";
    const content = editor.getModel()?.getValue() ?? "";
    const protoStartPositions = _advices.map((advice) => {
      return create(PositionSchema, {
        line: advice.startLineNumber,
        column: advice.startColumn,
      });
    });
    const protoEndPositions = _advices.map((advice) => {
      return create(PositionSchema, {
        line: advice.endLineNumber,
        column: advice.endColumn,
      });
    });
    const monacoStartPosition = batchConvertPositionToMonacoPosition(
      protoStartPositions,
      content
    );
    const monacoEndPosition = batchConvertPositionToMonacoPosition(
      protoEndPositions,
      content
    );

    const decorators = editor.createDecorationsCollection(
      _advices.map((advice, index) => {
        return {
          range: new monaco.Range(
            monacoStartPosition[index].lineNumber,
            monacoStartPosition[index].column,
            monacoEndPosition[index].lineNumber,
            monacoEndPosition[index].column
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
    WARNING: callCssVariable("--color-warning"),
    ERROR: callCssVariable("--color-error"),
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
