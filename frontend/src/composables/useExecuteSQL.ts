import { reactive } from "vue";
import { isEmpty } from "lodash-es";
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";

import {
  parseSQL,
  transformSQL,
  isSelectStatement,
  isMultipleStatements,
} from "../components/MonacoEditor/sqlParser";

const useExecuteSQL = () => {
  const store = useStore();
  const { t } = useI18n();
  const state = reactive({
    isLoadingData: false,
  });

  const notify = (type: string, title: string, description?: string) => {
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: type,
      title,
      description,
    });
  };

  const execute = async () => {
    if (state.isLoadingData) {
      notify("INFO", t("common.tips"), t("sql-editor.can-not-execute-query"));
    }

    const currentTab = store.getters["tab/currentTab"];
    const isDisconnected = store.getters["sqlEditor/isDisconnected"];
    const queryStatement = currentTab.queryStatement;
    const selectedStatement = currentTab.selectedStatement;
    const sqlStatement = selectedStatement || queryStatement;

    if (isDisconnected) {
      notify("CRITICAL", t("sql-editor.select-connection"));
      return;
    }

    const { data } = parseSQL(sqlStatement);

    if (isEmpty(sqlStatement)) {
      notify("CRITICAL", t("sql-editor.notify-empty-statement"));
      return;
    }

    if (data !== null && !isSelectStatement(data)) {
      store.dispatch("sqlEditor/setSqlEditorState", {
        isShowExecutingHint: true,
      });
      return;
    }

    if (isMultipleStatements(data)) {
      notify(
        "INFO",
        t("common.tips"),
        t("sql-editor.notify-multiple-statements")
      );
    }

    try {
      state.isLoadingData = true;
      store.dispatch("sqlEditor/setIsExecuting", true);
      // remove the comment from the sql statement in front-end
      const statement = data !== null ? transformSQL(data) : sqlStatement;
      const res = await store.dispatch("sqlEditor/executeQuery", {
        statement,
      });
      state.isLoadingData = false;
      store.dispatch("sqlEditor/setIsExecuting", false);

      if (res.error) {
        notify("CRITICAL", res.error);
        return;
      }
    } catch (error) {
      state.isLoadingData = false;
      store.dispatch("sqlEditor/setIsExecuting", false);
      notify("CRITICAL", error as string);
    }
  };

  return {
    state,
    execute,
  };
};

export { useExecuteSQL };
