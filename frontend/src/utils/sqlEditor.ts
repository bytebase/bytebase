import dayjs from "dayjs";
import { head } from "lodash-es";
import { v1 as uuidv1 } from "uuid";
import { useDatabaseV1Store, useQueryDataPolicy } from "@/store";
import type {
  QueryDataSourceType,
  SQLEditorConnection,
  SQLEditorTab,
} from "@/types";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  defaultViewState,
  isValidDatabaseName,
  isValidInstanceName,
  UNKNOWN_DATABASE_NAME,
} from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  DataSourceType,
  type InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import { wrapRefAsPromise } from "@/utils";
import {
  extractDatabaseResourceName,
  getInstanceResource,
} from "./v1/database";
import { instanceV1AllowsCrossDatabaseQuery } from "./v1/instance";

export const NEW_WORKSHEET_TITLE = "new worksheet";

export const defaultSQLEditorTab = (): SQLEditorTab => {
  return {
    id: uuidv1(),
    title: defaultSQLEditorTabTitle(),
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

const defaultSQLEditorTabTitle = () => {
  return dayjs().format("YYYY-MM-DD HH:mm");
};

export const emptySQLEditorConnection = (): SQLEditorConnection => {
  return {
    instance: "",
    database: "",
  };
};

export const getConnectionForSQLEditorTab = (tab?: SQLEditorTab) => {
  const target: {
    instance: InstanceResource | undefined;
    database: Database | undefined;
  } = {
    instance: undefined,
    database: undefined,
  };
  if (!tab) {
    return target;
  }
  const { connection } = tab;
  if (connection.database) {
    const database = useDatabaseV1Store().getDatabaseByName(
      connection.database
    );
    target.database = database;
    target.instance = getInstanceResource(database);
  }
  return target;
};

export const isSameSQLEditorConnection = (
  a: SQLEditorConnection,
  b: SQLEditorConnection
): boolean => {
  return a.instance === b.instance && a.database === b.database;
};

export const isSimilarDefaultSQLEditorTabTitle = (title: string) => {
  if (!title || title === NEW_WORKSHEET_TITLE) {
    return true;
  }
  const regex = /(^|\s)(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2})$/;
  return regex.test(title);
};

export const suggestedTabTitleForSQLEditorConnection = (
  conn: SQLEditorConnection
) => {
  const database = useDatabaseV1Store().getDatabaseByName(conn.database);
  const parts: string[] = [];
  const { databaseName, instance } = extractDatabaseResourceName(database.name);
  if (isValidDatabaseName(database.name)) {
    parts.push(databaseName);
  } else if (isValidInstanceName(instance)) {
    parts.push(getInstanceResource(database).title);
  }
  parts.push(defaultSQLEditorTabTitle());
  return parts.join(" ");
};

export const isConnectedSQLEditorTab = (tab: SQLEditorTab) => {
  const { instance, database } = getConnectionForSQLEditorTab(tab);
  if (!instance) {
    return false;
  }
  if (!isValidInstanceName(instance.name)) {
    return false;
  }

  if (instanceV1AllowsCrossDatabaseQuery(instance)) {
    // Connecting to instance directly.
    return true;
  }
  return database && isValidDatabaseName(database.name);
};

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
