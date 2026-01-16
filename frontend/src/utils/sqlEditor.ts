import dayjs from "dayjs";
import { head } from "lodash-es";
import { v1 as uuidv1 } from "uuid";
import { useDatabaseV1Store, useDataSourceRestrictionPolicy } from "@/store";
import type {
  ComposedDatabase,
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
import {
  DataSourceType,
  type InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import { QueryDataPolicy_Restriction } from "@/types/proto-es/v1/org_policy_service_pb";
import { wrapRefAsPromise } from "@/utils";
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
    database: ComposedDatabase | undefined;
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
    target.instance = database.instanceResource;
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
  if (isValidDatabaseName(database.name)) {
    parts.push(database.databaseName);
  } else if (isValidInstanceName(database.instance)) {
    parts.push(database.instanceResource.title);
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

const getDataSourceBehavior = async (database: ComposedDatabase) => {
  const { ready, dataSourceRestriction } =
    useDataSourceRestrictionPolicy(database);
  await wrapRefAsPromise(ready, /* expected */ true);

  let behavior: "RO" | "FALLBACK" | "ALLOW_ADMIN";
  if (
    dataSourceRestriction.value.environmentPolicy ===
      QueryDataPolicy_Restriction.DISALLOW ||
    dataSourceRestriction.value.projectPolicy ===
      QueryDataPolicy_Restriction.DISALLOW
  ) {
    behavior = "RO";
  } else if (
    dataSourceRestriction.value.environmentPolicy ===
      QueryDataPolicy_Restriction.FALLBACK ||
    dataSourceRestriction.value.projectPolicy ===
      QueryDataPolicy_Restriction.FALLBACK
  ) {
    behavior = "FALLBACK";
  } else {
    behavior = "ALLOW_ADMIN";
  }
  return behavior;
};

export const getValidDataSourceByPolicy = async (
  database: ComposedDatabase,
  type?: QueryDataSourceType
) => {
  const adminDataSource = database.instanceResource.dataSources.find(
    (ds) => ds.type === DataSourceType.ADMIN
  )!;
  const readonlyDataSources = database.instanceResource.dataSources.filter(
    (ds) => ds.type === DataSourceType.READ_ONLY
  );

  const behavior = await getDataSourceBehavior(database);

  switch (behavior) {
    case "ALLOW_ADMIN":
    // ALLOW_ADMIN means no policy restriction.
    case "FALLBACK": {
      // FALLBACK means try to use read-only data source, it can also accept admin data source if no read-only data source exists.
      switch (type) {
        case DataSourceType.ADMIN:
          return adminDataSource.id;
        default:
          // try to use read-only data source first.
          return head(readonlyDataSources)?.id ?? adminDataSource.id;
      }
    }
    case "RO": {
      // RO only accept the read-only data source.
      return head(readonlyDataSources)?.id;
    }
  }
};
