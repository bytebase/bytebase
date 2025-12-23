import { create } from "@bufbuild/protobuf";
import { Code } from "@connectrpc/connect";
import { cloneDeep, isEmpty } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { markRaw, reactive } from "vue";
import { t } from "@/plugins/i18n";
import {
  hasFeature,
  pushNotification,
  useDatabaseV1Store,
  useDBGroupStore,
  useSQLEditorQueryHistoryStore,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLStore,
} from "@/store";
import type {
  BBNotificationStyle,
  ComposedDatabase,
  QueryContextStatus,
  QueryDataSourceType,
  SQLEditorConnection,
  SQLEditorDatabaseQueryContext,
  SQLEditorQueryParams,
  SQLResultSetV1,
} from "@/types";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import {
  type QueryOption,
  QueryOptionSchema,
  QueryRequestSchema,
  QueryResult_PermissionDenied_CommandType,
} from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  getValidDataSourceByPolicy,
  hasPermissionToCreateChangeDatabaseIssue,
} from "@/utils";
import { flattenNoSQLResult } from "./utils";

// QUERY_INTERVAL_LIMIT is the minimal gap between two queries
const QUERY_INTERVAL_LIMIT = 1000;

const useExecuteSQL = () => {
  const state = reactive<{
    lastQueryTime?: number;
  }>({});
  const dbGroupStore = useDBGroupStore();
  const dbStore = useDatabaseV1Store();
  const tabStore = useSQLEditorTabStore();
  const sqlEditorStore = useSQLEditorStore();
  const queryHistoryStore = useSQLEditorQueryHistoryStore();

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
                view: DatabaseGroupView.FULL,
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
    await dbStore.batchGetOrFetchDatabases([...batchQueryDatabaseSet.keys()]);

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
              isBatch ? tab.batchQueryContext.dataSourceType : undefined
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

    const { abortController } = context;
    if (!abortController) {
      return;
    }
    const sqlStore = useSQLStore();

    const dataSourceId = context.params.connection.dataSourceId;
    if (!dataSourceId) {
      return finish({
        error: t("sql-editor.no-data-source"),
        results: [],
        status: Code.NotFound,
      });
    }

    if (abortController.signal.aborted) {
      // Once any one of the batch queries is aborted, don't go further
      // and mock an "Aborted" result for the rest queries.
      return finish({
        error: t("sql-editor.request-aborted"),
        results: [],
        status: Code.Aborted,
      });
    }

    const queryOption = create(QueryOptionSchema, {
      ...(context.params.queryOption ?? ({} as QueryOption)),
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

    if (isOnlySelectError(resultSet)) {
      // Show a tips to navigate to issue creation
      // if the user is allowed to create issue in the project.
      if (hasPermissionToCreateChangeDatabaseIssue(database)) {
        sqlEditorStore.isShowExecutingHint = true;
        sqlEditorStore.executingHintDatabase = database;
      }
      return finish(resultSet);
    }

    return finish(markRaw(resultSet));
  };

  const execute = async (params: SQLEditorQueryParams) => {
    return preExecute(params);
  };

  return {
    execute,
    runQuery,
  };
};

const isOnlySelectError = (resultSet: SQLResultSetV1) => {
  return resultSet.results.some((result) => {
    return (
      result.detailedError.case === "permissionDenied" &&
      [
        QueryResult_PermissionDenied_CommandType.DDL,
        QueryResult_PermissionDenied_CommandType.DML,
        QueryResult_PermissionDenied_CommandType.NON_READ_ONLY,
      ].includes(result.detailedError.value.commandType)
    );
  });
};

export { useExecuteSQL };
