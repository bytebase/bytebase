import { debounce, orderBy } from "lodash-es";
import * as monaco from "monaco-editor";
import { computed, ref, watchEffect } from "vue";

interface WebSocketMessage {
  method: string;
  params: unknown;
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

const rangeChangeEvent = "$/textDocument/statementRanges";

export const useActiveRangeByCursor = (
  editor: monaco.editor.IStandaloneCodeEditor
) => {
  // Use in-memory ref instead of localStorage to avoid I/O on every update
  const statementRangeByUri = ref<Map<string, monaco.IRange[]>>(new Map());
  const activeCursorPosition = ref<
    { line: number; column: number } | undefined
  >();

  // Debounce cursor position updates to reduce computation frequency
  const updateCursorPosition = debounce(
    (e: monaco.editor.ICursorPositionChangedEvent) => {
      activeCursorPosition.value = {
        line: e.position.lineNumber,
        column: e.position.column,
      };
    },
    50 // 50ms debounce for cursor movement
  );

  editor.onDidChangeCursorPosition(updateCursorPosition);

  const activeRange = computed((): monaco.IRange | undefined => {
    const cursorPos = activeCursorPosition.value;
    if (!cursorPos) {
      return;
    }

    const model = editor.getModel();
    if (!model) {
      return;
    }

    const ranges = statementRangeByUri.value.get(model.uri.toString());
    if (!ranges || ranges.length === 0) {
      return;
    }

    // Use binary search for better performance with many ranges
    let activeRange: monaco.IRange | undefined = undefined;
    const cursorLine = cursorPos.line;
    const cursorCol = cursorPos.column;

    // Quick search through ranges
    for (let i = 0; i < ranges.length; i++) {
      const range = ranges[i];

      // Skip ranges before cursor line
      if (range.endLineNumber < cursorLine) {
        continue;
      }

      // Found a range containing the cursor line
      if (
        range.startLineNumber <= cursorLine &&
        range.endLineNumber >= cursorLine
      ) {
        // Check column position
        if (range.endColumn >= cursorCol) {
          activeRange = range;
          break;
        }

        // Check if this is the last matching range
        if (
          i === ranges.length - 1 ||
          ranges[i + 1].startLineNumber > cursorLine
        ) {
          activeRange = range;
          break;
        }
      }

      // Passed cursor position, stop searching
      if (range.startLineNumber > cursorLine) {
        break;
      }
    }

    if (!activeRange) {
      return;
    }

    // Only check last line if multi-line range
    if (activeRange.endLineNumber > activeRange.startLineNumber) {
      const lastLineStatement = model.getValueInRange({
        startLineNumber: activeRange.endLineNumber,
        startColumn: 1,
        endLineNumber: activeRange.endLineNumber,
        endColumn: activeRange.endColumn,
      });

      if (!lastLineStatement) {
        const adjustedRange = {
          startLineNumber: activeRange.startLineNumber,
          startColumn: activeRange.startColumn,
          endLineNumber: activeRange.endLineNumber - 1,
          endColumn: Infinity,
        };

        // Return adjusted range only if cursor is within it
        if (cursorLine <= adjustedRange.endLineNumber) {
          return adjustedRange;
        }
        return;
      }
    }

    return activeRange;
  });

  import("../lsp-client").then(async ({ connectionWebSocket }) => {
    let messageHandler: ((msg: MessageEvent) => void) | null = null;

    watchEffect(() => {
      if (!connectionWebSocket.value) {
        return;
      }

      connectionWebSocket.value?.then((ws) => {
        // Remove old listener if it exists
        if (messageHandler) {
          ws.removeEventListener("message", messageHandler);
        }

        // Create optimized message handler
        messageHandler = (msg: MessageEvent) => {
          try {
            if (!msg || !msg.data) {
              return;
            }

            // Early exit if not our message type (avoid parsing)
            if (!msg.data.includes(rangeChangeEvent)) {
              return;
            }

            const data = JSON.parse(msg.data) as WebSocketMessage;
            if (data.method !== rangeChangeEvent) {
              return;
            }
            const rangeMessage = data.params as StatementRangeMessage;
            if (!rangeMessage.uri || !Array.isArray(rangeMessage.ranges)) {
              return;
            }

            // Process and cache the ranges
            const processedRanges = orderBy(
              rangeMessage.ranges,
              (range) => range.start,
              "asc"
            ).map((range) => {
              // The position starts from 1 in the editor.
              return {
                startLineNumber: range.start.line + 1,
                endLineNumber: range.end.line + 1,
                startColumn: range.start.character + 1,
                endColumn: range.end.character + 1,
              };
            });

            // Update the ref (which serves as our cache)
            statementRangeByUri.value.set(rangeMessage.uri, processedRanges);
          } catch {
            // nothing
          }
        };

        ws.addEventListener("message", messageHandler);
      });
    });
  });

  return activeRange;
};
