import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import {
  useDatabaseV1Store,
  useInstanceV1Store,
  useSQLEditorTabStore,
} from "@/store";
import {
  ComposedDatabase,
  ComposedInstance,
  CoreSQLEditorTab,
  DEFAULT_SQL_EDITOR_TAB_MODE,
  SQLEditorConnection,
  SQLEditorTab,
  UNKNOWN_ID,
} from "@/types";
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

// export const isSimilarDefaultTabName = (name: string) => {
//   const regex = /(^|\s)(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2})$/;
//   return regex.test(name);
// };

// export const INITIAL_TAB = getDefaultTab();

// export const isTempTab = (tab: TabInfo): boolean => {
//   if (tab.sheetName) return false;
//   if (!tab.isSaved) return false;
//   if (tab.statement) return false;
//   return true;
// };

// export const sheetTypeForTab = (tab: TabInfo): TabSheetType => {
//   if (!tab.sheetName) {
//     return "TEMP";
//   }
//   if (tab.isSaved) {
//     return "CLEAN";
//   }
//   return "DIRTY";
// };

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

export const tryConnectToCoreSQLEditorTab = (tab: CoreSQLEditorTab) => {
  const tabStore = useSQLEditorTabStore();
  const currentTab = tabStore.currentTab;
  if (currentTab) {
    if (isSimilarSQLEditorTab(tab, currentTab)) {
      // Don't go further if the connection doesn't change.
      return;
    }
    if (currentTab.status === "NEW") {
      // If the current tab is "fresh new", update its connection directly.
      tabStore.updateCurrentTab(tab);
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
