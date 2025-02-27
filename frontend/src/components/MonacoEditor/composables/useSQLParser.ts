import { useLocalStorage } from "@vueuse/core";
import { orderBy } from "lodash-es";
import type monaco from "monaco-editor";
import { watchEffect } from "vue";
import { MonacoEditorReady } from "../editor";

interface WebSocketMessage {
  method: string;
  params: any;
}

interface StatementRange {
  end: {
    line: number;
    character: number;
  };
  start: {
    line: number;
    character: number;
  };
}

interface StatementRangeMessage {
  uri: string;
  ranges: StatementRange[];
}

export const useSQLParser = () => {
  const statementRangeByUri = useLocalStorage<Map<string, monaco.IRange[]>>(
    "bb.sql-editor.statement-range",
    new Map()
  );

  watchEffect(() => {
    MonacoEditorReady.then(() => {
      import("../lsp-client").then(({ connectionWebSocket }) => {
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
    });
  });

  const getActiveStatementRange = (
    uri: string,
    line: number
  ): monaco.IRange | undefined => {
    const ranges = statementRangeByUri.value.get(uri);
    if (!ranges || !Array.isArray(ranges)) {
      return undefined;
    }

    for (const range of ranges) {
      if (range.startLineNumber <= line && range.endLineNumber >= line) {
        return range;
      }
    }
    return undefined;
  };

  return {
    getActiveStatementRange,
  };
};
