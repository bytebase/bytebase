import { computed } from "vue";
import { v1 as uuidv1 } from "uuid";
import { t } from "../plugins/i18n";
import { TabInfo } from "@/types";

export const defaultTabName = computed(() => t("sql-editor.untitled-sheet"));

export const getDefaultTab = (): TabInfo => {
  return {
    id: uuidv1(),
    name: defaultTabName.value,
    isModified: false,
    statement: "",
    selectedStatement: "",
  };
};

export const isReplaceableTab = (tab: TabInfo): boolean => {
  if (tab.sheetId) return false;
  if (tab.isModified) return false;
  if (tab.statement) return false;
  return true;
};
