import dayjs from "dayjs";
import { v1 as uuidv1 } from "uuid";
import { t } from "../plugins/i18n";

export const getDefaultTab = () => {
  return {
    id: uuidv1(),
    name: t("sql-editor.untitled-sheet"),
    isSaved: true,
    savedAt: dayjs().format("YYYY-MM-DD HH:mm:ss"),
    statement: "",
    selectedStatement: "",
  };
};
