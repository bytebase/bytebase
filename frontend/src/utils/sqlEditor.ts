import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import {
  useDatabaseV1Store,
  useInstanceV1Store,
  useSQLEditorTabStore,
} from "@/store";
import type {
  ComposedDatabase,
  ComposedInstance,
  CoreSQLEditorTab,
  SQLEditorConnection,
  SQLEditorTab,
  SQLEditorTabQueryContext,
} from "@/types";
import { DEFAULT_SQL_EDITOR_TAB_MODE, UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
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
    sheet: "",
    editMode: "SQL-EDITOR",
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
    instance: ComposedInstance | undefined;
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
    target.instance = database.instanceEntity;
  } else if (connection.instance) {
    const instance = useInstanceV1Store().getInstanceByUID(connection.instance);
    target.instance = instance;
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
  b: CoreSQLEditorTab
): boolean => {
  return (
    isSameSQLEditorConnection(a.connection, b.connection) &&
    a.sheet === b.sheet &&
    a.mode === b.mode
  );
};

export const isSimilarToDefaultSQLEditorTabTitle = (title: string) => {
  const regex = /(^|\s)(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2})$/;
  return regex.test(title);
};

export const suggestedTabTitleForSQLEditorConnection = (
  conn: SQLEditorConnection
) => {
  const instance = useInstanceV1Store().getInstanceByName(conn.instance);
  const database = useDatabaseV1Store().getDatabaseByName(conn.database);
  const parts: string[] = [];
  if (database.uid !== String(UNKNOWN_ID)) {
    parts.push(database.databaseName);
  } else if (instance.uid !== String(UNKNOWN_ID)) {
    parts.push(instance.title);
  }
  parts.push(defaultSQLEditorTabTitle());
  return parts.join(" ");
};

export const isDisconnectedSQLEditorTab = (tab: SQLEditorTab) => {
  const { connection } = tab;
  if (!connection.instance) {
    return true;
  }
  const instance = useInstanceV1Store().getInstanceByName(connection.instance);
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
    if (currentTab.status === "NEW" || !currentTab.sheet) {
      // If the current tab is "fresh new", update its connection directly.
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
  },
});
