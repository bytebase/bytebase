import dayjs from "dayjs";
import { computed } from "vue";
import { v1 as uuidv1 } from "uuid";
import { t } from "../plugins/i18n";

export const defaultTabName = computed(() => t("sql-editor.untitled-sheet"));

export const getDefaultTab = () => {
  return {
    id: uuidv1(),
    name: defaultTabName.value,
    isSaved: true,
    savedAt: dayjs().format("YYYY-MM-DD HH:mm:ss"),
    statement: "",
    selectedStatement: "",
  };
};
