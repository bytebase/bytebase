import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import { computed } from "vue";
import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import type { Connection, ConnectionAtom, CoreTabInfo, TabInfo } from "@/types";
import { UNKNOWN_ID, TabMode } from "@/types";
import { t } from "../plugins/i18n";

export const defaultTabName = computed(() => t("sql-editor.untitled-sheet"));

export const emptyConnection = (): Connection => {
  return {
    instanceId: String(UNKNOWN_ID),
    databaseId: String(UNKNOWN_ID),
  };
};

export const getDefaultTab = (): TabInfo => {
  return {
    id: uuidv1(),
    name: defaultTabName.value,
    connection: emptyConnection(),
    isSaved: true,
    savedAt: dayjs().format("YYYY-MM-DD HH:mm:ss"),
    statement: "",
    selectedStatement: "",
    mode: TabMode.ReadOnly,
    editMode: "SQL-EDITOR",
    isExecutingSQL: false,
  };
};

export const INITIAL_TAB = getDefaultTab();

export const isTempTab = (tab: TabInfo): boolean => {
  if (tab.sheetName) return false;
  if (!tab.isSaved) return false;
  if (tab.statement) return false;
  return true;
};

export const isSameConnection = (a: Connection, b: Connection): boolean => {
  return a.instanceId === b.instanceId && a.databaseId === b.databaseId;
};

export const isSimilarTab = (a: CoreTabInfo, b: CoreTabInfo): boolean => {
  return (
    isSameConnection(a.connection, b.connection) &&
    a.sheetName === b.sheetName &&
    a.mode === b.mode
  );
};

export const getDefaultTabNameFromConnection = (conn: Connection) => {
  const instance = useInstanceV1Store().getInstanceByUID(conn.instanceId);
  const database = useDatabaseV1Store().getDatabaseByUID(conn.databaseId);
  if (database.uid !== String(UNKNOWN_ID)) {
    return `${database.databaseName}`;
  }
  if (instance.uid !== String(UNKNOWN_ID)) {
    return `${instance.title}`;
  }
  return defaultTabName.value;
};

export const instanceOfConnectionAtom = (atom: ConnectionAtom) => {
  if (atom.type === "instance") {
    return useInstanceV1Store().getInstanceByUID(atom.id);
  }
  if (atom.type === "database") {
    return useDatabaseV1Store().getDatabaseByUID(atom.id).instanceEntity;
  }
  return undefined;
};
