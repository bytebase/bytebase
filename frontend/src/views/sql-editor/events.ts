import Emittery from "emittery";
import type { IRange } from "monaco-editor";
import type {
  BatchQueryContext,
  SQLEditorConnection,
  SQLEditorTab,
} from "@/types";

export type SQLEditorEvents = {
  "save-sheet": { tab: SQLEditorTab; editTitle?: boolean };
  "alter-schema": { databaseName: string; schema: string; table: string };
  "execute-sql": {
    connection: SQLEditorConnection;
    statement: string;
    batchQueryContext?: BatchQueryContext;
  };
  "format-content": undefined;
  "tree-ready": undefined;
  "project-context-ready": { project: string };
  "set-editor-selection": IRange;
  "append-editor-content": { content: string; select: boolean };
  "insert-at-caret": { content: string };
  // Fired after a SQL statement has finished executing — by both the
  // worksheet (`useExecuteSQL`) and admin/terminal (`webTerminal`)
  // paths. `HistoryPane` listens and refetches. Bypasses store
  // reactivity that doesn't reliably propagate the post-exec
  // mutations into the React `useVueState` subscriber.
  "query-executed": undefined;
};

export const sqlEditorEvents: Emittery<SQLEditorEvents> =
  new Emittery<SQLEditorEvents>();
