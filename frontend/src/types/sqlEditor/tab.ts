import type { Selection as MonacoSelection } from "@/components/MonacoEditor";
import { t } from "@/plugins/i18n";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import type { SQLResultSetV1 } from "../v1/sql";
import type { SQLEditorConnection, SQLEditorQueryParams } from "./editor";
import type { EditorPanelViewState } from "./tabViewState";

export type SQLEditorTabStatus =
  | "DIRTY" // edited
  | "CLEAN"; // saved to a remote sheet

export type SQLEditorTabMode = "WORKSHEET" | "ADMIN";
export const DEFAULT_SQL_EDITOR_TAB_MODE: SQLEditorTabMode = "WORKSHEET";
export type QueryDataSourceType =
  | DataSourceType.ADMIN
  | DataSourceType.READ_ONLY;

export const getDataSourceTypeI18n = (dsType?: DataSourceType) => {
  switch (dsType) {
    case DataSourceType.ADMIN:
      return t("sql-editor.batch-query.select-data-source.admin");
    case DataSourceType.READ_ONLY:
      return t("sql-editor.batch-query.select-data-source.readonly");
    default:
      return "Unknown";
  }
};

export type BatchQueryContext = {
  // databases is used to store the selected database names.
  // Format: instances/{instance}/databases/{database}
  databases: string[];

  // databaseGroups is used to store the selected database group names for batch request.
  // Format: projects/{project}/databaseGroups/{databaseGroup}
  databaseGroups?: string[];

  dataSourceType?: QueryDataSourceType;
};

// PENDING: ready, pending to request.
// EXECUTING: requesting.
// DONE: request finished.
// CANCELLED: request cancelled.
export type QueryContextStatus = "PENDING" | "EXECUTING" | "DONE" | "CANCELLED";

export type SQLEditorDatabaseQueryContext = {
  id: string;
  // we will generate a new abortController when status changed to EXECUTING.
  abortController?: AbortController;
  // request params in the history.
  params: SQLEditorQueryParams;
  // PENDING: ready, pending to request.
  // EXECUTING: requesting.
  // DONE: request finished.
  // CANCELLED: request cancelled.
  status: QueryContextStatus;
  // beginTimestampMS will store the start request time when status changed to EXECUTING.
  beginTimestampMS?: number;
  // query result.
  resultSet?: SQLResultSetV1;
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
  databaseQueryContexts?: Map<
    string /* database or instance */,
    SQLEditorDatabaseQueryContext[]
  >;
  batchQueryContext: BatchQueryContext;

  // extended fields
  treeState: {
    database: string;
    keys: string[];
  };
  editorState: {
    selection: MonacoSelection | null;
  };
  viewState: EditorPanelViewState;
};

export type CoreSQLEditorTab = Pick<
  SQLEditorTab,
  "worksheet" | "connection" | "mode"
>;
