import { head } from "lodash-es";
import { v1 as uuidv1 } from "uuid";
import { useQueryDataPolicy } from "@/store";
import type {
  QueryDataSourceType,
  SQLEditorConnection,
  SQLEditorTab,
} from "@/types";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  defaultViewState,
  UNKNOWN_DATABASE_NAME,
} from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { wrapRefAsPromise } from "@/utils";
import { getInstanceResource } from "./v1/database";

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

// `getConnectionForSQLEditorTab` and `isConnectedSQLEditorTab` moved to
// `@/react/lib/sqlEditorConnection` so the database lookup can go through
// the React app store without dragging `@/react/stores/app` into the
// `@/utils` import graph (which would create a static ESM cycle).

export const getValidDataSourceByPolicy = async (
  database: Database,
  type?: QueryDataSourceType
) => {
  const instanceResource = getInstanceResource(database);
  const adminDataSource = instanceResource.dataSources.find(
    (ds) => ds.type === DataSourceType.ADMIN
  )!;
  const readonlyDataSources = instanceResource.dataSources.filter(
    (ds) => ds.type === DataSourceType.READ_ONLY
  );

  const { ready, policy } = useQueryDataPolicy(database.project);
  await wrapRefAsPromise(ready, /* expected */ true);

  if (policy.value.allowAdminDataSource) {
    switch (type) {
      case DataSourceType.ADMIN:
        return adminDataSource.id;
      default:
        // try to use read-only data source first.
        return head(readonlyDataSources)?.id ?? adminDataSource.id;
    }
  }

  return head(readonlyDataSources)?.id ?? adminDataSource.id;
};
