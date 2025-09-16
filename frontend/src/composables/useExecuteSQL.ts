import { create } from "@bufbuild/protobuf";
import { Code } from "@connectrpc/connect";
import { createContextValues } from "@connectrpc/connect";
import Emittery from "emittery";
import { head, isEmpty, cloneDeep } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { markRaw, reactive } from "vue";
import { STATEMENT_SKIP_CHECK_THRESHOLD } from "@/components/SQLCheck/common";
import { sqlServiceClientConnect } from "@/grpcweb";
import { ignoredCodesContextKey } from "@/grpcweb/context-key";
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
  batchGetOrFetchDatabases,
  useDatabaseV1Store,
} from "@/store";
import type {
  ComposedDatabase,
  SQLResultSetV1,
  BBNotificationStyle,
  SQLEditorQueryParams,
  SQLEditorConnection,
  SQLEditorTab,
  QueryContextStatus,
  SQLEditorDatabaseQueryContext,
  QueryDataSourceType,
} from "@/types";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import {
  CheckRequestSchema,
  CheckRequest_ChangeType as NewCheckRequest_ChangeType,
  QueryRequestSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import type { Advice } from "@/types/proto-es/v1/sql_service_pb";
import {
  Advice_Status,
  QueryOptionSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  getValidDataSourceByPolicy,
  hasPermissionToCreateChangeDatabaseIssue,
} from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { flattenNoSQLResult } from "./utils";

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
  const dbStore = useDatabaseV1Store();
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
    if (new Blob([params.statement]).size > STATEMENT_SKIP_CHECK_THRESHOLD) {
      return { passed: true };
    }
    const request = create(CheckRequestSchema, {
      name: params.connection.database,
      statement: params.statement,
      changeType: NewCheckRequest_ChangeType.SQL_EDITOR,
    });
    const response = await sqlServiceClientConnect.check(request, {
      contextValues: createContextValues().set(ignoredCodesContextKey, [
        Code.PermissionDenied,
      ]),
      signal: abortController?.signal,
    });
    const advices = response.advices;
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

      adviceNotifyMessage += `${Advice_Status[advice.status]}: ${
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

  const getDataSourceId = (
    database: ComposedDatabase,
    connection: SQLEditorConnection,
    mode?: QueryDataSourceType
  ) => {
    if (
      database.instance === connection.instance &&
      !!connection.dataSourceId
    ) {
      return connection.dataSourceId;
    }

    return getValidDataSourceByPolicy(database, mode) ?? "";
  };

  const changeContextStatus = (
    ctx: SQLEditorDatabaseQueryContext,
    status: QueryContextStatus
  ) => {
    switch (status) {
      case "EXECUTING":
        ctx.abortController = new AbortController();
        ctx.beginTimestampMS = Date.now();
        break;
      case "CANCELLED":
        ctx.abortController?.abort();
        break;
      case "DONE":
        break;
    }
    ctx.status = status;
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
    if (tab.batchQueryContext && hasFeature(PlanFeature.FEATURE_BATCH_QUERY)) {
      const { databases = [], databaseGroups = [] } = tab.batchQueryContext;
      for (const databaseResourceName of databases) {
        if (!isValidDatabaseName(databaseResourceName)) {
          continue;
        }
        if (batchQueryDatabaseSet.has(databaseResourceName)) {
          continue;
        }
        batchQueryDatabaseSet.add(databaseResourceName);
      }

      if (hasFeature(PlanFeature.FEATURE_DATABASE_GROUPS)) {
        for (const databaseGroupName of databaseGroups) {
          try {
            const databaseGroup = await dbGroupStore.getOrFetchDBGroupByName(
              databaseGroupName,
              {
                skipCache: false,
                silent: true,
                view: DatabaseGroupView.MATCHED,
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
          changeContextStatus(context, "CANCELLED");
        }
        databaseQueryContexts.delete(database);
      }
    }

    const isBatch = batchQueryDatabaseSet.size > 1;
    await batchGetOrFetchDatabases([...batchQueryDatabaseSet.keys()]);

    for (const databaseName of batchQueryDatabaseSet.values()) {
      if (!databaseQueryContexts.has(databaseName)) {
        databaseQueryContexts.set(databaseName, []);
      }

      if ((databaseQueryContexts.get(databaseName)?.length ?? 0) >= 50) {
        const ctx = databaseQueryContexts.get(databaseName)?.pop();
        if (ctx) {
          changeContextStatus(ctx, "CANCELLED");
        }
      }

      const database = dbStore.getDatabaseByName(databaseName);
      const context: SQLEditorDatabaseQueryContext = {
        id: uuidv4(),
        params: Object.assign(cloneDeep(params), {
          connection: {
            ...params.connection,
            dataSourceId: getDataSourceId(
              database,
              params.connection,
              isBatch ? tab.batchQueryContext?.dataSourceType : undefined
            ),
          },
        }),
        status: "PENDING",
      };
      databaseQueryContexts.get(databaseName)?.unshift(context);
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

    changeContextStatus(context, "EXECUTING");

    const finish = (resultSet: SQLResultSetV1) => {
      context.resultSet = resultSet;
      changeContextStatus(context, "DONE");
    };

    const abort = (error: string, advices: Advice[] = []) => {
      notify("WARN", t("sql-editor.request-cancelled"));
      return finish({
        error,
        results: [],
        advices,
        status: Code.Aborted,
      });
    };

    const { abortController } = context;
    if (!abortController) {
      return;
    }
    const sqlStore = useSQLStore();

    const checkBehavior = context.params.skipCheck
      ? "SKIP"
      : sqlCheckStyle.value;
    let checkResult: SQLCheckResult = { passed: true };
    if (checkBehavior !== "SKIP") {
      try {
        checkResult = await check(abortController, context.params);
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

    const dataSourceId = context.params.connection.dataSourceId;
    if (!dataSourceId) {
      return finish({
        advices: [],
        error: t("sql-editor.no-data-source"),
        results: [],
        status: Code.NotFound,
      });
    }

    if (abortController.signal.aborted) {
      // Once any one of the batch queries is aborted, don't go further
      // and mock an "Aborted" result for the rest queries.
      return finish({
        advices: [],
        error: t("sql-editor.request-aborted"),
        results: [],
        status: Code.Aborted,
      });
    }

    const queryOption = create(QueryOptionSchema, {
      redisRunCommandsOn: sqlEditorStore.redisCommandOption,
    });
    const resultSet = await sqlStore.query(
      create(QueryRequestSchema, {
        name: database.name,
        dataSourceId: dataSourceId,
        statement: context.params.statement,
        limit: sqlEditorStore.resultRowsLimit,
        explain: context.params.explain,
        schema: context.params.connection.schema,
        container: context.params.connection.table,
        queryOption: queryOption,
      }),
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
    resultSet.status === Code.InvalidArgument
  ) {
    return true;
  }
  if (
    resultSet.error.match(/disallow execute (DML|DDL) statement/) &&
    resultSet.status === Code.PermissionDenied
  ) {
    return true;
  }
  return false;
};

export { useExecuteSQL };
