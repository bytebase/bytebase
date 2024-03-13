import { SQLResultSetV1 } from "../v1/sql";
import { SQLEditorConnection, SQLEditorQueryParams } from "./editor";

export type SQLEditorEditMode = "SQL-EDITOR" | "CHAT-TO-SQL";

export type SQLEditorTabStatus =
  | "NEW" // just created and untouched
  | "DIRTY" // edited
  | "CLEAN"; // saved to a remote sheet

// STANDARD not supported by backend so far
export type SQLEditorTabMode = "STANDARD" | "READONLY" | "ADMIN";
export const DEFAULT_SQL_EDITOR_TAB_MODE: SQLEditorTabMode = "READONLY";

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
  sheet: string; // if ref to a local or remote sheet
  connection: SQLEditorConnection;
  status: SQLEditorTabStatus;
  statement: string; // local editing statement, might be out-of-sync to ref sheet's statement
  selectedStatement: string;
  mode: SQLEditorTabMode;
  editMode: SQLEditorEditMode;

  // SQL query related fields
  // won't be saved to localStorage
  queryContext?: SQLEditorTabQueryContext;
  batchQueryContext?: BatchQueryContext;
};

export type CoreSQLEditorTab = Pick<
  SQLEditorTab,
  "sheet" | "connection" | "mode"
>;
