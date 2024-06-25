import { isEmpty } from "lodash-es";
import { Status } from "nice-grpc-common";
import { markRaw } from "vue";
import { parseSQL } from "@/components/MonacoEditor/sqlParser";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLStore,
  useSQLEditorQueryHistoryStore,
} from "@/store";
import type {
  ComposedDatabase,
  SQLResultSetV1,
  BBNotificationStyle,
  SQLEditorQueryParams,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import {
  Advice_Status,
  advice_StatusToJSON,
} from "@/types/proto/v1/sql_service";
import {
  emptySQLEditorTabQueryContext,
  hasPermissionToCreateChangeDatabaseIssue,
} from "@/utils";

const useExecuteSQL = () => {
  const currentUser = useCurrentUserV1();
  const databaseStore = useDatabaseV1Store();
  const tabStore = useSQLEditorTabStore();
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

  const preflight = (params: SQLEditorQueryParams) => {
    const tab = tabStore.currentTab;
    if (!tab) {
      return false;
    }

    if (tab.queryContext?.status === "EXECUTING") {
      notify("INFO", t("common.tips"), t("sql-editor.can-not-execute-query"));
      return false;
    }

    if (tabStore.isDisconnected) {
      notify("CRITICAL", t("sql-editor.select-connection"));
      return false;
    }

    if (isEmpty(params.statement)) {
      notify("CRITICAL", t("sql-editor.notify-empty-statement"));
      return false;
    }

    tab.queryContext = {
      ...emptySQLEditorTabQueryContext(),
      params,
      status: "EXECUTING",
    };
    return true;
  };

  const cleanup = () => {
    const tab = tabStore.currentTab;
    if (!tab) return;
    if (!tab.queryContext) return;
    tab.queryContext.status = "IDLE";
  };

  const execute = async (params: SQLEditorQueryParams) => {
    if (!preflight(params)) {
      return cleanup();
    }

    const tab = tabStore.currentTab;
    if (!tab) {
      return;
    }
    const { mode } = tab;
    if (mode === "ADMIN") {
      return;
    }
    const queryContext = tab.queryContext!;
    const batchQueryContext = tab.batchQueryContext;
    const { data } = await parseSQL(params.statement);

    if (data === undefined) {
      notify("CRITICAL", t("sql-editor.notify-invalid-sql-statement"));
      return cleanup();
    }

    const selectedDatabase = useDatabaseV1Store().getDatabaseByName(
      params.connection.database
    );
    const databaseName =
      selectedDatabase.uid === String(UNKNOWN_ID)
        ? ""
        : selectedDatabase.databaseName;
    const batchQueryDatabases: ComposedDatabase[] = [selectedDatabase];

    // Check if the user selects multiple databases to query.
    if (
      databaseName &&
      batchQueryContext &&
      batchQueryContext.databases.length > 0
    ) {
      for (const databaseResourceName of batchQueryContext.databases) {
        const database = databaseStore.getDatabaseByName(databaseResourceName);
        if (database.name === selectedDatabase.name) {
          continue;
        }
        batchQueryDatabases.push(database);
      }
    }

    const queryResultMap = new Map<string, SQLResultSetV1>();
    for (const database of batchQueryDatabases) {
      queryResultMap.set(database.name, {
        error: "",
        results: [],
        advices: [],
        allowExport: false,
      });
    }
    queryContext.results = queryResultMap;

    const fail = (database: ComposedDatabase, result: SQLResultSetV1) => {
      queryResultMap.set(database.name, {
        error: result.error,
        results: [],
        advices: result.advices,
        status: result.status,
        allowExport: false,
      });
    };

    const { abortController } = queryContext;
    const sqlStore = useSQLStore();
    queryContext.beginTimestampMS = Date.now();
    for (const database of batchQueryDatabases) {
      if (abortController.signal.aborted) {
        // Once any one of the batch queries is aborted, don't go further
        // and mock an "Aborted" result for the rest queries.
        fail(database, {
          advices: [],
          allowExport: false,
          error: "AbortError: The user aborted a request.",
          results: [],
          status: Status.ABORTED,
        });
        continue;
      }

      try {
        let resultSet: SQLResultSetV1;
        if (mode === "READONLY") {
          const isUnknownDatabase = database.uid === String(UNKNOWN_ID);
          const instance = isUnknownDatabase
            ? params.connection.instance
            : database.instance;
          const dataSourceId =
            instance === params.connection.instance
              ? params.connection.dataSourceId ?? ""
              : "";
          resultSet = await sqlStore.queryReadonly(
            {
              name: database.name,
              connectionDatabase: database.databaseName, // deprecated field, remove me later
              dataSourceId: dataSourceId,
              statement: params.statement,
              limit: sqlEditorStore.resultRowsLimit,
              explain: params.explain,
              timeout: undefined, // TODO: make this param configurable
            },
            abortController.signal
          );
        } else {
          resultSet = await sqlStore.executeStandard(
            {
              name: database.name,
              statement: params.statement,
              limit: sqlEditorStore.resultRowsLimit,
              timeout: undefined, // TODO: make this param configurable
            },
            abortController.signal
          );
        }

        let adviceStatus: "SUCCESS" | "ERROR" | "WARNING" = "SUCCESS";
        let adviceNotifyMessage = "";
        for (const advice of resultSet.advices) {
          if (advice.status === Advice_Status.SUCCESS) {
            continue;
          }

          if (advice.status === Advice_Status.ERROR) {
            adviceStatus = "ERROR";
          } else if (adviceStatus !== "ERROR") {
            adviceStatus = "WARNING";
          }

          adviceNotifyMessage += `${advice_StatusToJSON(advice.status)}: ${
            advice.title
          }\n`;
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

        if (resultSet.error) {
          // The error message should be consistent with the one from the backend.
          if (
            resultSet.error === "Support SELECT sql statement only" &&
            resultSet.status === Status.INVALID_ARGUMENT
          ) {
            const database = databaseStore.getDatabaseByName(
              params.connection.database
            );
            // Show a tips to navigate to issue creation
            // if the user is allowed to create issue in the project.
            if (
              hasPermissionToCreateChangeDatabaseIssue(
                database,
                currentUser.value
              )
            ) {
              sqlEditorStore.isShowExecutingHint = true;
              cleanup();
            }
            fail(database, resultSet);
          } else {
            fail(database, resultSet);
          }
        } else {
          queryResultMap.set(database.name, markRaw(resultSet));
        }
      } catch (error: any) {
        fail(database, error.response?.data?.message ?? String(error));
      }
    }

    // After all the queries are executed, we update the tab with the latest query result map.
    // Refresh the query history list when the query executed successfully
    // (with or without warnings).
    useSQLEditorQueryHistoryStore().fetchQueryHistoryList();
    cleanup();
  };

  return {
    execute,
  };
};

export { useExecuteSQL };
