import type {
  AdviceOption,
  Selection as MonacoSelection,
} from "@/components/MonacoEditor";
import type { SQLResultSetV1 } from "../v1/sql";
import type { SQLEditorConnection, SQLEditorQueryParams } from "./editor";

export type SQLEditorTabStatus =
  | "NEW" // just created and untouched
  | "DIRTY" // edited
  | "CLEAN"; // saved to a remote sheet

export type SQLEditorTabMode = "WORKSHEET" | "ADMIN";
export const DEFAULT_SQL_EDITOR_TAB_MODE: SQLEditorTabMode = "WORKSHEET";

export type SQLEditorQueryMode = "QUERY" | "EXECUTE";

export type BatchQueryContext = {
  // databases is used to store the selected database names.
  // Format: instances/{instance}/databases/{database}
  databases: string[];
};

export type SQLEditorTabQueryContext = {
  beginTimestampMS: number;
  abortController: AbortController;
  status: "IDLE" | "EXECUTING";

  params: SQLEditorQueryParams;
  results: Map<string /* database or instance */, SQLResultSetV1>;
};

export type SQLEditorTab = {
  // basic fields
  id: string; // uuid
  title: string; // display title, should be synced with sheet's title once saved
  worksheet: string; // if ref to a local or remote sheet
  connection: SQLEditorConnection;
  status: SQLEditorTabStatus;
  statement: string; // local editing statement, might be out-of-sync to ref sheet's statement
  selectedStatement: string;
  mode: SQLEditorTabMode;

  // SQL query related fields
  // won't be saved to localStorage
  queryContext?: SQLEditorTabQueryContext;
  batchQueryContext?: BatchQueryContext;

  // extended fields
  treeState: {
    database: string;
    keys: string[];
  };
  editorState: {
    selection: MonacoSelection | null;
    advices: AdviceOption[];
  };
  queryMode: SQLEditorQueryMode;
};

export type CoreSQLEditorTab = Pick<
  SQLEditorTab,
  "worksheet" | "connection" | "mode"
>;
