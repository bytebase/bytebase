import { markRaw } from "vue";
import { isEmpty } from "lodash-es";
import { useI18n } from "vue-i18n";

import {
  parseSQL,
  isSelectStatement,
  isMultipleStatements,
  isDDLStatement,
  isDMLStatement,
} from "../components/MonacoEditor/sqlParser";
import { pushNotification, useTabStore, useSQLEditorStore } from "@/store";
import { BBNotificationStyle } from "@/bbkit/types";
import {
  ExecuteConfig,
  ExecuteOption,
  SQLResultSet,
  TabMode,
  SQLAdviceCodeGeneral,
} from "@/types";

const useExecuteSQL = () => {
  const { t } = useI18n();
  const tabStore = useTabStore();
  const sqlEditorStore = useSQLEditorStore();

  const notify = (
    type: BBNotificationStyle,
    title: string,
    description?: string
  ) => {
    pushNotification({
      module: "bytebase",
      style: type,
      title,
      description,
    });
  };

  const executeReadonly = async (
    query: string,
    config: ExecuteConfig,
    option?: Partial<ExecuteOption>
  ) => {
    const tab = tabStore.currentTab;
    const { data } = parseSQL(query);

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
        sqlEditorStore.setSQLEditorState({
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

    let selectStatement = query;
    if (option?.explain) {
      selectStatement = `EXPLAIN ${selectStatement}`;
    }

    try {
      const sqlResultSet = await sqlEditorStore.executeQuery({
        statement: selectStatement,
      });
      // TODO(steven): use BBModel instead of notify to show the advice from SQL review.
      let adviceStatus = "SUCCESS";
      let adviceNotifyMessage = "";
      for (const advice of sqlResultSet.adviceList) {
        if (advice.code === SQLAdviceCodeGeneral.NotFound) {
          // Hide the "SQL review policy is not configured or disabled" warning
          // since it's too annoying.
          continue;
        }

        if (advice.status === "ERROR") {
          adviceStatus = "ERROR";
        } else if (adviceStatus !== "ERROR") {
          adviceStatus = advice.status;
        }

        adviceNotifyMessage += `${advice.status}: ${advice.title}\n`;
        if (advice.content) {
          adviceNotifyMessage += `${advice.content}\n`;
        }
      }
      if (adviceStatus !== "SUCCESS") {
        const notifyStyle = adviceStatus === "ERROR" ? "CRITICAL" : "WARN";
        notify(
          notifyStyle,
          t("sql-editor.sql-review-result"),
          adviceNotifyMessage
        );
      }

      // use `markRaw` to prevent vue from monitoring the object change deeply
      const queryResult = sqlResultSet.data
        ? markRaw(sqlResultSet.data)
        : undefined;
      Object.assign(tab, {
        queryResult,
        adviceList: sqlResultSet.adviceList,
        executeParams: {
          query,
          config,
          option,
        },
      });
      if (queryResult) {
        // Refresh the query history list when the query executed successfully
        // (with or without warnings).
        sqlEditorStore.fetchQueryHistoryList();
      }

      return sqlResultSet;
    } catch (error) {
      Object.assign(tab, {
        queryResult: undefined,
        adviceList: undefined,
        executeParams: {
          query,
          config,
          option,
        },
      });
      notify("CRITICAL", error as string);
    }
  };

  const executeAdmin = async (
    query: string,
    config: ExecuteConfig,
    option?: Partial<ExecuteOption>
  ) => {
    const tab = tabStore.currentTab;

    let statement = query;
    if (option?.explain) {
      statement = `EXPLAIN ${statement}`;
    }

    try {
      const sqlResultSet = await sqlEditorStore.executeAdminQuery({
        statement,
      });

      // use `markRaw` to prevent vue from monitoring the object change deeply
      const queryResult = sqlResultSet.data
        ? markRaw(sqlResultSet.data)
        : undefined;
      Object.assign(tab, {
        queryResult,
        adviceList: sqlResultSet.adviceList,
        executeParams: {
          query,
          config,
          option,
        },
      });
      if (queryResult) {
        // Refresh the query history list when the query executed successfully
        // (with or without warnings).
        sqlEditorStore.fetchQueryHistoryList();
      }

      return sqlResultSet;
    } catch (error) {
      Object.assign(tab, {
        queryResult: undefined,
        adviceList: undefined,
        executeParams: {
          query,
          config,
          option,
        },
      });
      notify("CRITICAL", error as string);
    }
  };

  const execute = async (
    query: string,
    config: ExecuteConfig,
    option?: Partial<ExecuteOption>
  ) => {
    const tab = tabStore.currentTab;

    if (tab.isExecutingSQL) {
      notify("INFO", t("common.tips"), t("sql-editor.can-not-execute-query"));
      return;
    }

    const isDisconnected = tabStore.isDisconnected;
    if (isDisconnected) {
      notify("CRITICAL", t("sql-editor.select-connection"));
      return;
    }

    if (isEmpty(query)) {
      notify("CRITICAL", t("sql-editor.notify-empty-statement"));
      return;
    }

    tab.isExecutingSQL = true;

    let result: SQLResultSet | undefined = undefined;
    if (tab.mode === TabMode.ReadOnly) {
      result = await executeReadonly(query, config, option);
    } else if (tab.mode === TabMode.Admin) {
      result = await executeAdmin(query, config, option);
    }

    tab.isExecutingSQL = false;

    return result;
  };

  return {
    execute,
  };
};

export { useExecuteSQL };
