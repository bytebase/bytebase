import dayjs from "dayjs";
import { computed } from "vue";
import { v1 as uuidv1 } from "uuid";
import { t } from "../plugins/i18n";
import type { Connection, TabInfo } from "@/types";
import { UNKNOWN_ID, TabMode } from "@/types";

export const defaultTabName = computed(() => t("sql-editor.untitled-sheet"));

export const emptyConnection = (): Connection => {
  return {
    instanceId: UNKNOWN_ID,
    databaseId: UNKNOWN_ID,
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
    isExecutingSQL: false,
  };
};

export const INITIAL_TAB = getDefaultTab();

export const isTempTab = (tab: TabInfo): boolean => {
  if (tab.sheetId) return false;
  if (!tab.isSaved) return false;
  if (tab.statement) return false;
  return true;
};

export const isSameConnection = (a: Connection, b: Connection): boolean => {
  return a.instanceId === b.instanceId && a.databaseId === b.databaseId;
};
