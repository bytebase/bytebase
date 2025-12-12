import dayjs from "dayjs";
import { head } from "lodash-es";
import { v1 as uuidv1 } from "uuid";
import { useDatabaseV1Store, usePolicyV1Store } from "@/store";
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
import type { Policy } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  DataSourceQueryPolicy_Restriction,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
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

export const getAdminDataSourceRestrictionOfDatabase = (
  database: ComposedDatabase
) => {
  const policyStore = usePolicyV1Store();
  const projectLevelPolicy = policyStore.getPolicyByParentAndType({
    parentPath: database.project,
    policyType: PolicyType.DATA_SOURCE_QUERY,
  });
  const projectLevelAdminDSRestriction =
    projectLevelPolicy?.policy?.case === "dataSourceQueryPolicy"
      ? projectLevelPolicy.policy.value.adminDataSourceRestriction
      : undefined;

  let envLevelPolicy: Policy | undefined;
  if (database.effectiveEnvironment) {
    envLevelPolicy = policyStore.getPolicyByParentAndType({
      parentPath: database.effectiveEnvironment,
      policyType: PolicyType.DATA_SOURCE_QUERY,
    });
  }

  const envLevelAdminDSRestriction =
    envLevelPolicy?.policy?.case === "dataSourceQueryPolicy"
      ? envLevelPolicy.policy.value.adminDataSourceRestriction
      : undefined;
  return {
    environmentPolicy:
      envLevelAdminDSRestriction ??
      DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
    projectPolicy:
      projectLevelAdminDSRestriction ??
      DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
  };
};

const getDataSourceBehavior = (database: ComposedDatabase) => {
  const restriction = getAdminDataSourceRestrictionOfDatabase(database);
  let behavior: "RO" | "FALLBACK" | "ALLOW_ADMIN";
  if (
    restriction.environmentPolicy ===
      DataSourceQueryPolicy_Restriction.DISALLOW ||
    restriction.projectPolicy === DataSourceQueryPolicy_Restriction.DISALLOW
  ) {
    behavior = "RO";
  } else if (
    restriction.environmentPolicy ===
      DataSourceQueryPolicy_Restriction.FALLBACK ||
    restriction.projectPolicy === DataSourceQueryPolicy_Restriction.FALLBACK
  ) {
    behavior = "FALLBACK";
  } else {
    behavior = "ALLOW_ADMIN";
  }
  return behavior;
};

export const getValidDataSourceByPolicy = (
  database: ComposedDatabase,
  type?: QueryDataSourceType
) => {
  const adminDataSource = database.instanceResource.dataSources.find(
    (ds) => ds.type === DataSourceType.ADMIN
  )!;
  const readonlyDataSources = database.instanceResource.dataSources.filter(
    (ds) => ds.type === DataSourceType.READ_ONLY
  );

  const behavior = getDataSourceBehavior(database);

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
