import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import type {
  ComposedDatabase,
  ComposedInstance,
  Connection,
  ConnectionAtom,
  CoreTabInfo,
  TabInfo,
  TabSheetType,
} from "@/types";
import { UNKNOWN_ID, TabMode } from "@/types";
import { instanceV1AllowsCrossDatabaseQuery } from "./v1/instance";

export const getDefaultTabName = () => {
  return dayjs().format("YYYY-MM-DD HH:mm");
};

export const isSimilarDefaultTabName = (name: string) => {
  const regex = /^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2})$/;
  return regex.test(name);
};

export const emptyConnection = (): Connection => {
  return {
    instanceId: String(UNKNOWN_ID),
    databaseId: String(UNKNOWN_ID),
  };
};

export const getDefaultTab = (): TabInfo => {
  return {
    id: uuidv1(),
    name: getDefaultTabName(),
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

export const sheetTypeForTab = (tab: TabInfo): TabSheetType => {
  if (!tab.sheetName) {
    return "TEMP";
  }
  if (tab.isSaved) {
    return "CLEAN";
  }
  return "DIRTY";
};

export const connectionForTab = (tab: TabInfo) => {
  const target: {
    instance: ComposedInstance | undefined;
    database: ComposedDatabase | undefined;
  } = {
    instance: undefined,
    database: undefined,
  };
  const { instanceId, databaseId } = tab.connection;
  if (databaseId !== String(UNKNOWN_ID)) {
    const database = useDatabaseV1Store().getDatabaseByUID(databaseId);
    target.database = database;
    target.instance = database.instanceEntity;
  } else if (instanceId !== String(UNKNOWN_ID)) {
    const instance = useInstanceV1Store().getInstanceByUID(instanceId);
    target.instance = instance;
  }
  return target;
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
  return getDefaultTabName();
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

export const isDisconnectedTab = (tab: TabInfo) => {
  const { instanceId, databaseId } = tab.connection;
  if (instanceId === String(UNKNOWN_ID)) {
    return true;
  }
  const instance = useInstanceV1Store().getInstanceByUID(instanceId);
  if (instanceV1AllowsCrossDatabaseQuery(instance)) {
    // Connecting to instance directly.
    return false;
  }
  return databaseId === String(UNKNOWN_ID);
};
