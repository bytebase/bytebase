import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import { SQLEditorConnection, SQLEditorTab } from "@/types";

export const defaultSQLEditorTab = (): SQLEditorTab => {
  return {
    id: uuidv1(),
    title: defaultSQLEditorTabTitle(),
    connection: emptySQLEditorConnection(),
    statement: "",
    status: "NEW",
    mode: "STANDARD",
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

// export const connectionForTab = (tab: TabInfo) => {
//   const target: {
//     instance: ComposedInstance | undefined;
//     database: ComposedDatabase | undefined;
//   } = {
//     instance: undefined,
//     database: undefined,
//   };
//   const { instanceId, databaseId } = tab.connection;
//   if (databaseId !== String(UNKNOWN_ID)) {
//     const database = useDatabaseV1Store().getDatabaseByUID(databaseId);
//     target.database = database;
//     target.instance = database.instanceEntity;
//   } else if (instanceId !== String(UNKNOWN_ID)) {
//     const instance = useInstanceV1Store().getInstanceByUID(instanceId);
//     target.instance = instance;
//   }
//   return target;
// };

// const isSameConnection = (a: Connection, b: Connection): boolean => {
//   return a.instanceId === b.instanceId && a.databaseId === b.databaseId;
// };

// export const isSimilarTab = (a: CoreTabInfo, b: CoreTabInfo): boolean => {
//   return (
//     isSameConnection(a.connection, b.connection) &&
//     a.sheetName === b.sheetName &&
//     a.mode === b.mode
//   );
// };

// export const getSuggestedTabNameFromConnection = (conn: Connection) => {
//   const instance = useInstanceV1Store().getInstanceByUID(conn.instanceId);
//   const database = useDatabaseV1Store().getDatabaseByUID(conn.databaseId);
//   const parts: string[] = [];
//   if (database.uid !== String(UNKNOWN_ID)) {
//     parts.push(database.databaseName);
//   } else if (instance.uid !== String(UNKNOWN_ID)) {
//     parts.push(instance.title);
//   }
//   parts.push(getDefaultTabName());
//   return parts.join(" ");
// };

// export const isDisconnectedTab = (tab: TabInfo) => {
//   const { instanceId, databaseId } = tab.connection;
//   if (instanceId === String(UNKNOWN_ID)) {
//     return true;
//   }
//   const instance = useInstanceV1Store().getInstanceByUID(instanceId);
//   if (instanceV1AllowsCrossDatabaseQuery(instance)) {
//     // Connecting to instance directly.
//     return false;
//   }
//   return databaseId === String(UNKNOWN_ID);
// };

// export const tryConnectToCoreTab = (tab: CoreTabInfo) => {
//   const tabStore = useTabStore();
//   if (isSimilarTab(tab, tabStore.currentTab)) {
//     // Don't go further if the connection doesn't change.
//     return;
//   }
//   if (tabStore.currentTab.isFreshNew) {
//     // If the current tab is "fresh new", update its connection directly.
//     tabStore.updateCurrentTab(tab);
//   } else {
//     // Otherwise select or add a new tab and set its connection.
//     const name = getSuggestedTabNameFromConnection(tab.connection);
//     tabStore.selectOrAddSimilarTab(
//       tab,
//       false /* beside */,
//       name /* defaultTabName */
//     );
//     tabStore.updateCurrentTab(tab);
//   }
// };
