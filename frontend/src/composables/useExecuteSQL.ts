import Emittery from "emittery";
import { head, isEmpty } from "lodash-es";
import { Status, ClientError } from "nice-grpc-common";
import { markRaw, reactive } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBGroupStore,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLStore,
  useSQLEditorQueryHistoryStore,
  useAppFeature,
  hasFeature,
  getSqlReviewReports,
} from "@/store";
import type {
  ComposedDatabase,
  SQLResultSetV1,
  BBNotificationStyle,
  SQLEditorQueryParams,
  SQLEditorTab,
} from "@/types";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseGroupView } from "@/types/proto/v1/database_group_service";
import {
  Advice,
  Advice_Status,
  advice_StatusToJSON,
  CheckRequest_ChangeType,
  type BatchQueryRequest,
} from "@/types/proto/v1/sql_service";
import {
  emptySQLEditorTabQueryContext,
  ensureDataSourceSelection,
  hasPermissionToCreateChangeDatabaseIssue,
} from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { flattenNoSQLResult } from "./utils";

// SKIP_CHECK_THRESHOLD is the MaxSheetCheckSize in the backend.
const SKIP_CHECK_THRESHOLD = 2 * 1024 * 1024;
// QUERY_INTERVAL_LIMIT is the minimal gap between two queries
const QUERY_INTERVAL_LIMIT = 1000;

const events = new Emittery<{
  "update:advices": {
    tab: SQLEditorTab;
    params: SQLEditorQueryParams;
    advices: Advice[];
  };
}>();

type SQLCheckResult = { passed: boolean; advices?: Advice[] };

const useExecuteSQL = () => {
  const state = reactive<{
    lastQueryTime?: number;
  }>({});
  const databaseStore = useDatabaseV1Store();
  const dbGroupStore = useDBGroupStore();
  const tabStore = useSQLEditorTabStore();
  const sqlEditorStore = useSQLEditorStore();
  const queryHistoryStore = useSQLEditorQueryHistoryStore();
  const sqlCheckStyle = useAppFeature("bb.feature.sql-editor.sql-check-style");

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

  const preflight = async (params: SQLEditorQueryParams) => {
    state.lastQueryTime = Date.now();

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

    const emptyContext = emptySQLEditorTabQueryContext();
    tab.queryContext = {
      ...emptyContext,
      status: "EXECUTING",
      results: tab.queryContext?.results ?? emptyContext.results,
    };
    return true;
  };

  const check = async (
    params: SQLEditorQueryParams
  ): Promise<SQLCheckResult> => {
    const tab = tabStore.currentTab;
    if (!tab) {
      return { passed: false };
    }

    const abortController = tab.queryContext?.abortController;
    if (!params) {
      return { passed: false };
    }
    if (new Blob([params.statement]).size > SKIP_CHECK_THRESHOLD) {
      return { passed: true };
    }
    const response = await sqlServiceClient.check(
      {
        name: params.connection.database,
        statement: params.statement,
        changeType: CheckRequest_ChangeType.SQL_EDITOR,
      },
      {
        ignoredCodes: [Status.PERMISSION_DENIED],
        signal: abortController?.signal,
      }
    );
    const { advices } = response;
    events.emit("update:advices", { tab, params, advices });
    return { passed: advices.length === 0, advices };
  };

  const notifyAdvices = (advices: Advice[]) => {
    let adviceStatus: "SUCCESS" | "ERROR" | "WARNING" = "SUCCESS";
    let adviceNotifyMessage = "";
    for (const advice of advices) {
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
  };

  const cleanup = () => {
    const tab = tabStore.currentTab;
    if (!tab) return;
    if (!tab.queryContext) return;
    tab.queryContext.status = "IDLE";
  };

  const execute = async (params: SQLEditorQueryParams) => {
    const now = Date.now();
    if (
      state.lastQueryTime &&
      now - state.lastQueryTime < QUERY_INTERVAL_LIMIT
    ) {
      return;
    }

    if (!preflight(params)) {
      return cleanup();
    }

    const tab = tabStore.currentTab;
    if (!tab) {
      return cleanup();
    }
    const { mode } = tab;
    if (mode === "ADMIN") {
      return cleanup();
    }

    const queryContext = tab.queryContext!;
    const selectedDatabase = await databaseStore.getOrFetchDatabaseByName(
      params.connection.database
    );
    if (!isValidDatabaseName(selectedDatabase.name)) {
      return cleanup();
    }
    const batchQueryDatabaseMap: Map<string, ComposedDatabase> = new Map([
      [selectedDatabase.name, selectedDatabase],
    ]);

    // Check if the user selects multiple databases to query.
    if (tab.batchQueryContext && hasFeature("bb.feature.batch-query")) {
      const { databases, databaseGroups } = tab.batchQueryContext;
      for (const databaseResourceName of databases) {
        if (batchQueryDatabaseMap.has(databaseResourceName)) {
          continue;
        }
        const database =
          await databaseStore.getOrFetchDatabaseByName(databaseResourceName);
        if (!isValidDatabaseName(database.name)) {
          continue;
        }
        batchQueryDatabaseMap.set(database.name, database);
      }

      if (hasFeature("bb.feature.database-grouping")) {
        for (const databaseGroupName of databaseGroups) {
          try {
            const databaseGroup = await dbGroupStore.getOrFetchDBGroupByName(
              databaseGroupName,
              {
                skipCache: false,
                silent: true,
                view: DatabaseGroupView.DATABASE_GROUP_VIEW_FULL,
              }
            );
            for (const matchedDatabase of databaseGroup.matchedDatabases) {
              if (batchQueryDatabaseMap.has(matchedDatabase.name)) {
                continue;
              }
              const database = await databaseStore.getOrFetchDatabaseByName(
                matchedDatabase.name
              );
              if (!isValidDatabaseName(database.name)) {
                continue;
              }
              batchQueryDatabaseMap.set(database.name, database);
            }
          } catch {
            // skip
          }
        }
      }
    }

    const beginTimestampMS = Date.now();

    for (const database of queryContext.results.keys()) {
      if (!batchQueryDatabaseMap.has(database)) {
        queryContext.results.delete(database);
      }
    }

    const unshiftQueryResult = (resultSet: SQLResultSetV1) => {
      if (!queryContext.results.has(resultSet.name)) {
        queryContext.results.set(resultSet.name, []);
      }
      if (queryContext.results.get(resultSet.name)!.length >= 10) {
        queryContext.results.get(resultSet.name)!.pop();
      }
      queryContext.results.get(resultSet.name)!.unshift({
        params,
        beginTimestampMS,
        resultSet,
      });
    };

    const fail = ({
      databases,
      error,
      status,
      advices = [],
    }: {
      databases: string[];
      error: string;
      status: Status;
      advices?: Advice[];
    }) => {
      for (const database of databases) {
        unshiftQueryResult({
          error,
          results: [],
          advices,
          status,
          name: database,
        });
      }
      return cleanup();
    };

    const { abortController } = queryContext;

    const sqlStore = useSQLStore();
    const checkBehavior = params.skipCheck ? "SKIP" : sqlCheckStyle.value;
    let checkResult: SQLCheckResult = { passed: true };
    if (checkBehavior !== "SKIP") {
      try {
        checkResult = await check(params);
      } catch (error) {
        return fail({
          databases: [params.connection.database],
          error: extractGrpcErrorMessage(error),
          status: Status.ABORTED,
        });
      }
    }
    if (checkBehavior === "PREFLIGHT" && !checkResult.passed) {
      return fail({
        databases: [params.connection.database],
        error: head(checkResult.advices)?.content ?? "",
        status: Status.ABORTED,
        advices: checkResult.advices,
      });
    }
    if (
      checkBehavior === "NOTIFICATION" &&
      !checkResult.passed &&
      checkResult.advices
    ) {
      const { advices } = checkResult;
      const errorAdvice = advices.find(
        (advice) => advice.status === Advice_Status.ERROR
      );
      if (errorAdvice) {
        notifyAdvices(advices);
        return fail({
          databases: [params.connection.database],
          error: errorAdvice.content,
          status: Status.ABORTED,
          advices,
        });
      }
    }

    const queryParams: BatchQueryRequest = {
      statement: params.statement,
      limit: sqlEditorStore.resultRowsLimit,
      explain: params.explain,
      requests: [...batchQueryDatabaseMap.values()].map((database) => {
        const instance = isValidDatabaseName(database.name)
          ? database.instance
          : params.connection.instance;
        const dataSourceId =
          ensureDataSourceSelection(
            instance === params.connection.instance
              ? params.connection.dataSourceId
              : undefined,
            database
          ) ?? "";

        return {
          name: database.name,
          dataSourceId,
          statement: "",
          limit: 0,
          explain: false,
          schema: params.connection.schema,
          container: params.connection.table,
          queryOption: {
            redisRunCommandsOn: sqlEditorStore.redisCommandOption,
          },
        };
      }),
    };

    try {
      if (abortController.signal.aborted) {
        // Once any one of the batch queries is aborted, don't go further
        // and mock an "Aborted" result for the rest queries.
        return fail({
          databases: queryParams.requests.map((r) => r.name),
          error: "AbortError: The user aborted a request.",
          status: Status.ABORTED,
        });
      }

      const response = await sqlStore.batchQuery(
        queryParams,
        abortController.signal
      );

      for (const queryResult of response) {
        const database = databaseStore.getDatabaseByName(queryResult.name);
        const resultSet: SQLResultSetV1 = {
          error: "",
          advices: [],
          ...queryResult,
        };
        if (
          database.instanceResource.engine === Engine.MONGODB ||
          database.instanceResource.engine === Engine.COSMOSDB
        ) {
          flattenNoSQLResult(resultSet);
        }

        if (checkBehavior === "NOTIFICATION") {
          notifyAdvices(checkResult.advices ?? []);
        }

        if (resultSet.error) {
          // The error message should be consistent with the one from the backend.
          if (isOnlySelectError(resultSet)) {
            // Show a tips to navigate to issue creation
            // if the user is allowed to create issue in the project.
            if (hasPermissionToCreateChangeDatabaseIssue(database)) {
              sqlEditorStore.isShowExecutingHint = true;
              sqlEditorStore.executingHintDatabase = database;
              cleanup();
            }
            unshiftQueryResult(resultSet);
          } else {
            unshiftQueryResult(resultSet);
          }
        } else {
          unshiftQueryResult(markRaw(resultSet));
          // After all the queries are executed, we update the tab with the latest query result map.
          // Refresh the query history list when the query executed successfully
          // (with or without warnings).
          queryHistoryStore.resetPageToken({
            project: sqlEditorStore.project,
            database: database.name,
          });
          queryHistoryStore.fetchQueryHistoryList({
            project: sqlEditorStore.project,
            database: database.name,
          });
        }
      }
    } catch (err) {
      return fail({
        error: extractGrpcErrorMessage(err),
        status: err instanceof ClientError ? err.code : Status.UNKNOWN,
        advices: getSqlReviewReports(err),
        databases: queryParams.requests.map((r) => r.name),
      });
    }

    cleanup();
  };

  return {
    events,
    execute,
  };
};

const isOnlySelectError = (resultSet: SQLResultSetV1) => {
  if (
    resultSet.error === "Support SELECT sql statement only" &&
    resultSet.status === Status.INVALID_ARGUMENT
  ) {
    return true;
  }
  if (
    resultSet.error.match(/disallow execute (DML|DDL) statement/) &&
    resultSet.status === Status.PERMISSION_DENIED
  ) {
    return true;
  }
  return false;
};

export { useExecuteSQL };
