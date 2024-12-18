import dayjs from "dayjs";
import { head } from "lodash-es";
import { v1 as uuidv1 } from "uuid";
import {
  useDatabaseV1Store,
  useInstanceResourceByName,
  usePolicyV1Store,
  useSQLEditorTabStore,
} from "@/store";
import type {
  ComposedDatabase,
  CoreSQLEditorTab,
  SQLEditorConnection,
  SQLEditorTab,
  SQLEditorTabQueryContext,
} from "@/types";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  isValidDatabaseName,
  isValidInstanceName,
  UNKNOWN_DATABASE_NAME,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DataSourceType,
  type InstanceResource,
} from "@/types/proto/v1/instance_service";
import {
  DataSourceQueryPolicy_Restriction,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { instanceV1AllowsCrossDatabaseQuery } from "./v1/instance";

export const defaultSQLEditorTab = (): SQLEditorTab => {
  return {
    id: uuidv1(),
    title: defaultSQLEditorTabTitle(),
    connection: emptySQLEditorConnection(),
    statement: "",
    selectedStatement: "",
    status: "NEW",
    mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    worksheet: "",
    treeState: {
      database: UNKNOWN_DATABASE_NAME,
      keys: [],
    },
    editorState: {
      selection: null,
      advices: [],
    },
  };
};

export const defaultSQLEditorTabTitle = () => {
  return dayjs().format("YYYY-MM-DD HH:mm");
};
export const emptySQLEditorConnection = (): SQLEditorConnection => {
  return {
    instance: "",
    database: "",
  };
};

export const connectionForSQLEditorTab = (tab: SQLEditorTab) => {
  const target: {
    instance: InstanceResource | undefined;
    database: ComposedDatabase | undefined;
  } = {
    instance: undefined,
    database: undefined,
  };
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

const isSameSQLEditorConnection = (
  a: SQLEditorConnection,
  b: SQLEditorConnection
): boolean => {
  return a.instance === b.instance && a.database === b.database;
};

export const isSimilarSQLEditorTab = (
  a: CoreSQLEditorTab,
  b: CoreSQLEditorTab,
  ignoreMode?: boolean
): boolean => {
  if (!isSameSQLEditorConnection(a.connection, b.connection)) return false;
  if (a.worksheet !== b.worksheet) return false;
  if (!ignoreMode && a.mode !== b.mode) return false;
  return true;
};

export const isSimilarToDefaultSQLEditorTabTitle = (title: string) => {
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

export const isDisconnectedSQLEditorTab = (tab: SQLEditorTab) => {
  const { connection } = tab;
  if (!connection.instance) {
    return true;
  }
  const instance = useInstanceResourceByName(connection.instance);
  if (instanceV1AllowsCrossDatabaseQuery(instance)) {
    // Connecting to instance directly.
    return false;
  }
  return connection.database === "";
};

export const tryConnectToCoreSQLEditorTab = (
  tab: CoreSQLEditorTab,
  overrideTitle = true,
  newTab = false
) => {
  const tabStore = useSQLEditorTabStore();
  if (newTab) {
    tabStore.addTab({}, true);
  }

  const currentTab = tabStore.currentTab;
  if (currentTab) {
    if (isSimilarSQLEditorTab(tab, currentTab)) {
      // Don't go further if the connection doesn't change.
      return;
    }
    tabStore.updateCurrentTab(tab);
    if (
      overrideTitle &&
      isSimilarToDefaultSQLEditorTabTitle(currentTab.title)
    ) {
      const title = suggestedTabTitleForSQLEditorConnection(tab.connection);
      tabStore.updateCurrentTab({
        title,
      });
    }
    return;
  }

  // Otherwise select or add a new tab and set its connection.
  const title = suggestedTabTitleForSQLEditorConnection(tab.connection);
  tabStore.selectOrAddSimilarNewTab(
    tab,
    false /* beside */,
    title /* defaultTabTitle */
  );
  tabStore.updateCurrentTab(tab);
};

export const emptySQLEditorTabQueryContext = (): SQLEditorTabQueryContext => ({
  beginTimestampMS: Date.now(),
  abortController: new AbortController(),
  status: "IDLE",
  results: new Map(),
  params: {
    connection: emptySQLEditorConnection(),
    engine: Engine.MYSQL,
    explain: false,
    statement: "",
    selection: null,
  },
});

export const getAdminDataSourceRestrictionOfDatabase = (
  database: ComposedDatabase
) => {
  const policyStore = usePolicyV1Store();
  const projectLevelPolicy = policyStore.getPolicyByParentAndType({
    parentPath: database.project,
    policyType: PolicyType.DATA_SOURCE_QUERY,
  });
  const projectLevelAdminDSRestriction =
    projectLevelPolicy?.dataSourceQueryPolicy?.adminDataSourceRestriction;
  const envLevelPolicy = policyStore.getPolicyByParentAndType({
    parentPath: database.effectiveEnvironment,
    policyType: PolicyType.DATA_SOURCE_QUERY,
  });
  const envLevelAdminDSRestriction =
    envLevelPolicy?.dataSourceQueryPolicy?.adminDataSourceRestriction;
  return {
    environmentPolicy:
      envLevelAdminDSRestriction ??
      DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
    projectPolicy:
      projectLevelAdminDSRestriction ??
      DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED,
  };
};

export const ensureDataSourceSelection = (
  current: string | undefined,
  database: ComposedDatabase
) => {
  const restriction = getAdminDataSourceRestrictionOfDatabase(database);
  const adminDataSource = database.instanceResource.dataSources.find(
    (ds) => ds.type === DataSourceType.ADMIN
  )!;
  const readonlyDataSources = database.instanceResource.dataSources.filter(
    (ds) => ds.type === DataSourceType.READ_ONLY
  );

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
  if (behavior === "ALLOW_ADMIN") {
    if (current) {
      return current;
    }
    return adminDataSource.id;
  }
  if (behavior === "FALLBACK") {
    if (current) {
      return current;
    }
    return head(readonlyDataSources)?.id ?? adminDataSource.id;
  }
  if (behavior === "RO") {
    if (
      current &&
      readonlyDataSources.findIndex((ds) => ds.id === current) >= 0
    ) {
      return current;
    }
    return head(readonlyDataSources)?.id;
  }
  console.warn(
    "[SQL Editor] failed to ensureDataSourceSelection",
    current,
    behavior,
    database,
    restriction
  );
  return undefined;
};
