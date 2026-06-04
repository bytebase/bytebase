import { v1 as uuidv1 } from "uuid";
import type { SQLEditorConnection, SQLEditorTab } from "@/types";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  defaultViewState,
  UNKNOWN_DATABASE_NAME,
} from "@/types";

export const defaultSQLEditorTab = (): SQLEditorTab => {
  return {
    id: uuidv1(),
    // Tabs are created untitled. The UI renders a localized "Untitled"
    // placeholder when the title is empty; users name worksheets explicitly
    // when (and if) they save.
    title: "",
    connection: emptySQLEditorConnection(),
    statement: "",
    selectedStatement: "",
    status: "CLEAN",
    mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    worksheet: "",
    treeState: {
      database: UNKNOWN_DATABASE_NAME,
      keys: [],
    },
    editorState: {
      selection: null,
    },
    viewState: defaultViewState(),
    batchQueryContext: {
      databases: [],
    },
  };
};

export const emptySQLEditorConnection = (): SQLEditorConnection => {
  return {
    instance: "",
    database: "",
  };
};

export const isSameSQLEditorConnection = (
  a: SQLEditorConnection,
  b: SQLEditorConnection
): boolean => {
  return a.instance === b.instance && a.database === b.database;
};

// `getConnectionForSQLEditorTab` / `isConnectedSQLEditorTab` and
// `getValidDataSourceByPolicy` moved to `@/react/lib/*` so the database and
// policy lookups read the React app store directly without dragging
// `@/react/stores/app` into the `@/utils` import graph (a static ESM cycle).
