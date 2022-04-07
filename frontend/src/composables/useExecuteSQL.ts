import { reactive } from "vue";
import { isEmpty } from "lodash-es";
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";

import {
  parseSQL,
  transformSQL,
  isSelectStatement,
  isMultipleStatements,
  isDDLStatement,
  isDMLStatement,
} from "../components/MonacoEditor/sqlParser";
import { pushNotification, useTabStore } from "@/store";

type ExecuteConfig = {
  databaseType: string;
};

type ExecuteOption = {
  explain: boolean;
};

const useExecuteSQL = () => {
  const store = useStore();
  const { t } = useI18n();
  const tabStore = useTabStore();
  const state = reactive({
    isLoadingData: false,
  });

  const notify = (type: string, title: string, description?: string) => {
    pushNotification({
      module: "bytebase",
      style: type,
      title,
      description,
    });
  };

  const execute = async (
    config: ExecuteConfig,
    option?: Partial<ExecuteOption>
  ) => {
    if (state.isLoadingData) {
      notify("INFO", t("common.tips"), t("sql-editor.can-not-execute-query"));
    }

    const currentTab = tabStore.currentTab;
    const isDisconnected = store.getters["sqlEditor/isDisconnected"];
    const statement = currentTab.statement;
    const selectedStatement = currentTab.selectedStatement;
    const sqlStatement = selectedStatement || statement;

    if (isDisconnected) {
      notify("CRITICAL", t("sql-editor.select-connection"));
      return;
    }

    const { data } = parseSQL(sqlStatement);

    if (isEmpty(sqlStatement)) {
      notify("CRITICAL", t("sql-editor.notify-empty-statement"));
      return;
    }

    if (data === undefined) {
      notify("CRITICAL", t("sql-editor.notify-invalid-sql-statement"));
      return;
    }

    if (data !== null && !isSelectStatement(data)) {
      if (isMultipleStatements(data)) {
        notify(
          "INFO",
          t("common.tips"),
          t("sql-editor.notify-multiple-statements")
        );
        return;
      }
      // only DDL and DML statements are allowed
      if (isDDLStatement(data) || isDMLStatement(data)) {
        store.dispatch("sqlEditor/setSqlEditorState", {
          isShowExecutingHint: true,
        });
        return;
      }
    }

    if (isMultipleStatements(data)) {
      notify(
        "INFO",
        t("common.tips"),
        t("sql-editor.notify-multiple-statements")
      );
    }

    try {
      const isExplain = option?.explain || false;
      state.isLoadingData = true;
      store.dispatch("sqlEditor/setIsExecuting", true);
      // remove the comment from the sql statement in front-end
      const selectStatement =
        data !== null ? transformSQL(data, config.databaseType) : sqlStatement;
      const explainStatement = `EXPLAIN ${selectStatement}`;
      const queryResult = await store.dispatch("sqlEditor/executeQuery", {
        statement: isExplain ? explainStatement : selectStatement,
      });
      tabStore.updateCurrentTab({ queryResult });
      store.dispatch("sqlEditor/fetchQueryHistoryList");
    } catch (error) {
      tabStore.updateCurrentTab({ queryResult: undefined });
      notify("CRITICAL", error as string);
    } finally {
      state.isLoadingData = false;
      store.dispatch("sqlEditor/setIsExecuting", false);
    }
  };

  return {
    state,
    execute,
  };
};

export { useExecuteSQL };
