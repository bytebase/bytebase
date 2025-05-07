import { useLocalStorage } from "@vueuse/core";
import { orderBy } from "lodash-es";
import * as monaco from "monaco-editor";
import { computed, ref } from "vue";

interface WebSocketMessage {
  method: string;
  params: any;
}

interface StatementRangeMessage {
  uri: string;
  ranges: {
    end: {
      line: number;
      character: number;
    };
    start: {
      line: number;
      character: number;
    };
  }[];
}

export const useActiveRangeByCursor = (
  editor: monaco.editor.IStandaloneCodeEditor
) => {
  const statementRangeByUri = useLocalStorage<Map<string, monaco.IRange[]>>(
    "bb.sql-editor.statement-range",
    new Map()
  );
  const activeCursorPosition = ref<
    { line: number; column: number } | undefined
  >();

  editor.onDidChangeCursorPosition(
    (e: monaco.editor.ICursorPositionChangedEvent) => {
      activeCursorPosition.value = {
        line: e.position.lineNumber,
        column: e.position.column,
      };
    }
  );

  const activeRange = computed((): monaco.IRange | undefined => {
    if (!activeCursorPosition.value) {
      return;
    }
    let activeRange: monaco.IRange | undefined = undefined;
    const model = editor.getModel();
    if (!model) {
      return;
    }
    const ranges = statementRangeByUri.value.get(model.uri.toString()) ?? [];
    for (let i = 0; i < ranges.length; i++) {
      const range = ranges[i];
      if (
        range.startLineNumber <= activeCursorPosition.value.line &&
        range.endLineNumber >= activeCursorPosition.value.line
      ) {
        if (range.endColumn >= activeCursorPosition.value.column) {
          activeRange = range;
          break;
        }

        if (
          i === ranges.length - 1 ||
          ranges[i + 1].startLineNumber > activeCursorPosition.value.line
        ) {
          activeRange = range;
          break;
        }
      }
    }
    if (!activeRange) {
      return;
    }

    // Check if the last line is empty.
    const lastLineStatement = model.getValueInRange({
      startLineNumber: activeRange.endLineNumber,
      startColumn: 1,
      endLineNumber: activeRange.endLineNumber,
      endColumn: activeRange.endColumn,
    });
    if (
      !lastLineStatement &&
      activeRange.endLineNumber > activeRange.startLineNumber
    ) {
      const range = {
        startLineNumber: activeRange.startLineNumber,
        startColumn: activeRange.startColumn,
        endLineNumber: activeRange.endLineNumber - 1,
        endColumn: Infinity,
      };
      if (activeCursorPosition.value.line > range.endLineNumber) {
        return;
      }
      return range;
    }
    return activeRange;
  });

  import("../lsp-client").then(async ({ connectionWebSocket }) => {
    connectionWebSocket.value?.then((ws) => {
      ws.addEventListener("message", (msg) => {
        try {
          if (!msg || !msg.data) {
            return;
          }
          const data = JSON.parse(msg.data) as WebSocketMessage;
          if (data.method !== "$/textDocument/statementRanges") {
            return;
          }
          const rangeMessage = data.params as StatementRangeMessage;
          if (!rangeMessage.uri || !Array.isArray(rangeMessage.ranges)) {
            return;
          }
          statementRangeByUri.value.set(
            rangeMessage.uri,
            orderBy(rangeMessage.ranges, (range) => range.start, "asc").map(
              (range) => {
                // The position starts from 1 in the editor.
                return {
                  startLineNumber: range.start.line + 1,
                  endLineNumber: range.end.line + 1,
                  startColumn: range.start.character + 1,
                  endColumn: range.end.character + 1,
                };
              }
            )
          );
        } catch {
          // nothing
        }
      });
    });
  });

  return activeRange;
};
