import { SQLResultSetV1 } from "./v1/sql";

export type SQLEditorEditMode = "SQL-EDITOR" | "CHAT-TO-SQL";

export interface SQLEditorConnection {
  instance: string; // instance resource name, empty if not connected
  database: string; // database resource name, empty if not connected to a database
  dataSourceId?: string;
  schema?: string;
  table?: string;
}

export type SQLEditorTabStatus =
  | "NEW" // just created and untouched
  | "DIRTY" // edited
  | "CLEAN"; // saved to a remote sheet

export type SQLEditorTabMode = "STANDARD" | "READONLY" | "ADMIN";

export type SQLEditorQuery = {
  connection: SQLEditorConnection; // the connection snapshot of the query
  result?: SQLResultSetV1;
};

export type BatchQueryContext = {
  // databases is used to store the selected database names.
  // Format: instances/{instance}/databases/{database}
  databases: string[];
};

export type SQLEditorTabQueryContext = {
  beginTimestampMS: number;
  abortController: AbortController;
  status: "IDLE" | "EXECUTING";

  statement: string; // the statement connection of the query
  explain: boolean;
  results: Map<string /* database or instance */, SQLResultSetV1>;
};

export interface SQLEditorTab {
  // basic fields
  id: string; // uuid
  title: string; // display title, should be synced with sheet's title once saved
  sheet: string; // if ref to a local or remote sheet
  connection: SQLEditorConnection;
  status: SQLEditorTabStatus;
  statement: string; // local editing statement, might be out-of-sync to ref sheet's statement
  mode: SQLEditorTabMode;
  editMode: SQLEditorEditMode;

  // SQL query related fields
  // won't be saved to localStorage
  queryContext?: SQLEditorTabQueryContext;
  batchContext?: BatchQueryContext;
}
