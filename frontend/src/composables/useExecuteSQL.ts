import Emittery from "emittery";
import { head, isEmpty } from "lodash-es";
import { Status } from "nice-grpc-common";
import { v4 as uuidv4 } from "uuid";
import { markRaw, reactive } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useDBGroupStore,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLStore,
  useSQLEditorQueryHistoryStore,
  useAppFeature,
  hasFeature,
} from "@/store";
import type {
  ComposedDatabase,
  SQLResultSetV1,
  BBNotificationStyle,
  SQLEditorQueryParams,
  SQLEditorTab,
  SQLEditorDatabaseQueryContext,
} from "@/types";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseGroupView } from "@/types/proto/v1/database_group_service";
import {
  Advice,
  Advice_Status,
  advice_StatusToJSON,
  CheckRequest_ChangeType,
} from "@/types/proto/v1/sql_service";
import {
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

    if (tabStore.isDisconnected) {
      notify("CRITICAL", t("sql-editor.select-connection"));
      return false;
    }

    if (isEmpty(params.statement)) {
      notify("CRITICAL", t("sql-editor.notify-empty-statement"));
      return false;
    }

    if (!tab.databaseQueryContexts) {
      tab.databaseQueryContexts = new Map();
    }
    return true;
  };

  const check = async (
    abortController: AbortController,
    params: SQLEditorQueryParams
  ): Promise<SQLCheckResult> => {
    const tab = tabStore.currentTab;
    if (!tab) {
      return { passed: false };
    }

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

  const preExecute = async (params: SQLEditorQueryParams) => {
    const now = Date.now();
    if (
      state.lastQueryTime &&
      now - state.lastQueryTime < QUERY_INTERVAL_LIMIT
    ) {
      return;
    }

    const tab = tabStore.currentTab;
    if (!tab) {
      return;
    }
    const { mode } = tab;
    if (mode === "ADMIN") {
      return;
    }

    if (!preflight(params)) {
      return;
    }

    if (!isValidDatabaseName(params.connection.database)) {
      return;
    }

    const databaseQueryContexts = tab.databaseQueryContexts!;
    const batchQueryDatabaseSet = new Set<string /* database name */>([
      params.connection.database,
    ]);

    // Check if the user selects multiple databases to query.
    if (tab.batchQueryContext && hasFeature("bb.feature.batch-query")) {
      const { databases, databaseGroups } = tab.batchQueryContext;
      for (const databaseResourceName of databases) {
        if (!isValidDatabaseName(databaseResourceName)) {
          continue;
        }
        if (batchQueryDatabaseSet.has(databaseResourceName)) {
          continue;
        }
        batchQueryDatabaseSet.add(databaseResourceName);
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
              if (!isValidDatabaseName(matchedDatabase.name)) {
                continue;
              }
              if (batchQueryDatabaseSet.has(matchedDatabase.name)) {
                continue;
              }
              batchQueryDatabaseSet.add(matchedDatabase.name);
            }
          } catch {
            // skip
          }
        }
      }
    }

    for (const [database, contexts] of databaseQueryContexts.entries()) {
      if (!batchQueryDatabaseSet.has(database)) {
        for (const context of contexts) {
          if (context.status === "EXECUTING") {
            context.abortController?.abort();
          }
        }
        databaseQueryContexts.delete(database);
      }
    }

    for (const databaseName of batchQueryDatabaseSet.values()) {
      if (!databaseQueryContexts.has(databaseName)) {
        databaseQueryContexts.set(databaseName, []);
      }

      if ((databaseQueryContexts.get(databaseName)?.length ?? 0) >= 10) {
        databaseQueryContexts.get(databaseName)?.pop();
      }

      databaseQueryContexts.get(databaseName)?.unshift({
        id: uuidv4(),
        params,
        status: "PENDING",
      });
    }
  };

  const runQuery = async (
    database: ComposedDatabase,
    context: SQLEditorDatabaseQueryContext
  ) => {
    if (context.status === "EXECUTING") {
      notify("INFO", t("common.tips"), t("sql-editor.can-not-execute-query"));
      return;
    }

    if (!isValidDatabaseName(database.name)) {
      notify(
        "CRITICAL",
        t("common.error"),
        t("sql-editor.invalid-database", { database: database.name })
      );
      return;
    }

    context.abortController = new AbortController();
    context.status = "EXECUTING";
    context.beginTimestampMS = Date.now();

    const finish = (resultSet: SQLResultSetV1) => {
      context.resultSet = resultSet;
      context.status = "DONE";
    };

    const abort = (error: string, advices: Advice[] = []) => {
      notify("WARN", t("sql-editor.request-cancelled"));
      return finish({
        error,
        results: [],
        advices,
        status: Status.ABORTED,
      });
    };

    const { abortController } = context;
    const sqlStore = useSQLStore();

    const checkBehavior = context.params.skipCheck
      ? "SKIP"
      : sqlCheckStyle.value;
    let checkResult: SQLCheckResult = { passed: true };
    if (checkBehavior !== "SKIP") {
      try {
        checkResult = await check(context.abortController, context.params);
      } catch (error) {
        return abort(extractGrpcErrorMessage(error));
      }
    }
    if (checkBehavior === "PREFLIGHT" && !checkResult.passed) {
      return abort(
        head(checkResult.advices)?.content ?? "",
        checkResult.advices
      );
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
        return abort(errorAdvice.content, advices);
      }
    }

    const instance = isValidDatabaseName(database.name)
      ? database.instance
      : context.params.connection.instance;
    const dataSourceId =
      ensureDataSourceSelection(
        instance === context.params.connection.instance
          ? context.params.connection.dataSourceId
          : undefined,
        database
      ) ?? "";

    if (!dataSourceId) {
      return finish({
        advices: [],
        error: t("sql-editor.no-data-source"),
        results: [],
        status: Status.NOT_FOUND,
      });
    }

    if (abortController.signal.aborted) {
      // Once any one of the batch queries is aborted, don't go further
      // and mock an "Aborted" result for the rest queries.
      return finish({
        advices: [],
        error: t("sql-editor.request-aborted"),
        results: [],
        status: Status.ABORTED,
      });
    }

    const resultSet = await sqlStore.query(
      {
        name: database.name,
        dataSourceId: dataSourceId,
        statement: context.params.statement,
        limit: sqlEditorStore.resultRowsLimit,
        explain: context.params.explain,
        schema: context.params.connection.schema,
        container: context.params.connection.table,
        queryOption: {
          redisRunCommandsOn: sqlEditorStore.redisCommandOption,
        },
      },
      abortController.signal
    );

    // After all the queries are executed, we update the tab with the latest query result map.
    // Refresh the query history list when the query executed successfully
    // (with or without warnings).
    queryHistoryStore.resetPageToken({
      project: sqlEditorStore.project,
      database: database.name,
    });
    queryHistoryStore
      .fetchQueryHistoryList({
        project: sqlEditorStore.project,
        database: database.name,
      })
      .catch(() => {
        /* nothing */
      });

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
        }
      }
      return finish(resultSet);
    }

    return finish(markRaw(resultSet));
  };

  const execute = async (params: SQLEditorQueryParams) => {
    return preExecute(params);
  };

  return {
    events,
    execute,
    runQuery,
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
