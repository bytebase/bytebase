import { create } from "@bufbuild/protobuf";
import { Code } from "@connectrpc/connect";
import { cloneDeep, isEmpty } from "lodash-es";
import { useCallback, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { isConnectedSQLEditorTab } from "@/react/lib/sqlEditorConnection";
import { getValidDataSourceByPolicy } from "@/react/lib/sqlEditorDataSource";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorStore as useSQLEditorReactStore } from "@/react/stores/sqlEditor";
import { getSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import {
  getDatabaseQueryContext,
  getSQLEditorTabsState,
} from "@/react/stores/sqlEditor/tab";
import type {
  BBNotificationStyle,
  QueryContextStatus,
  SQLEditorDatabaseQueryContext,
  SQLEditorQueryParams,
  SQLResultSetV1,
} from "@/types";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  type QueryOption,
  QueryOptionSchema,
  QueryRequestSchema,
  QueryResult_CommandError_Type,
} from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  getDatabaseProject,
  getInstanceResource,
  hasPermissionToCreateChangeDatabaseIssueInProject,
} from "@/utils";
import { flattenNoSQLResult } from "@/utils/sqlResult";
import { sqlEditorEvents } from "@/views/sql-editor/events";

// QUERY_INTERVAL_LIMIT is the minimal gap between two queries
const QUERY_INTERVAL_LIMIT = 1000;

export const useExecuteSQL = () => {
  const { t } = useTranslation();
  const lastQueryTimeRef = useRef<number | undefined>(undefined);
  // Eagerly fetch the subscription so the batch-query / database-group
  // gates below see the licensed plan when the user clicks Run. The hook
  // is mounted by `EditorMain` on every tab open, so this fires once per
  // tab; `loadSubscription` itself dedupes via an in-flight request.
  const loadSubscription = useAppStore((s) => s.loadSubscription);
  useEffect(() => {
    void loadSubscription();
  }, [loadSubscription]);
  const notify = (
    type: BBNotificationStyle,
    title: string,
    description?: string
  ) => {
    useAppStore.getState().notify({
      module: "bytebase",
      style: type,
      title,
      description,
    });
  };

  const preflight = useCallback(
    async (params: SQLEditorQueryParams) => {
      lastQueryTimeRef.current = Date.now();

      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabsState.currentTabId);
      if (!tab) {
        return false;
      }

      // Mirrors `useIsDisconnected` selector logic without a hook
      // (preflight runs inside a callback, not during render).
      if (!isConnectedSQLEditorTab(tab)) {
        notify("CRITICAL", t("sql-editor.select-connection"));
        return false;
      }

      if (isEmpty(params.statement)) {
        notify("CRITICAL", t("sql-editor.notify-empty-statement"));
        return false;
      }

      if (!tab.databaseQueryContexts) {
        tabsState.updateTab(tab.id, { databaseQueryContexts: new Map() });
      }
      return true;
    },
    [t]
  );

  // Propagates the status change to the Zustand store via
  // `updateDatabaseQueryContext` (the path that drives React re-renders).
  // For ad-hoc contexts that aren't tracked under `(database, ctx.id)` —
  // e.g. the ephemeral context in `getExplainTokenForMSSQL` — the store
  // action returns `undefined`; we fall back to mutating the passed-in
  // `ctx` directly so the caller still sees the new fields after
  // `await runQuery(...)`.
  //
  // Mutating `ctx` BEFORE the immer update would defeat the update:
  // immer compares the patched draft to the (already-mutated) original
  // state, detects no change, and returns the same state reference —
  // Zustand then skips firing subscribers, and the UI never re-renders.
  const changeContextStatus = (
    database: string,
    ctx: SQLEditorDatabaseQueryContext,
    status: QueryContextStatus
  ) => {
    const patch: Partial<SQLEditorDatabaseQueryContext> = { status };
    switch (status) {
      case "EXECUTING": {
        patch.abortController = new AbortController();
        patch.beginTimestampMS = Date.now();
        break;
      }
      case "CANCELLED":
        ctx.abortController?.abort();
        break;
      case "DONE":
        break;
    }
    const next = getSQLEditorTabsState().updateDatabaseQueryContext({
      database,
      contextId: ctx.id,
      context: patch,
    });
    if (!next) {
      // Ad-hoc context (not in the tab's map): mirror the patch onto
      // the passed-in object so awaiters can still observe the result.
      Object.assign(ctx, patch);
    }
  };

  const preExecute = useCallback(
    async (params: SQLEditorQueryParams) => {
      const now = Date.now();
      if (
        lastQueryTimeRef.current &&
        now - lastQueryTimeRef.current < QUERY_INTERVAL_LIMIT
      ) {
        return;
      }

      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabsState.currentTabId);
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

      // Re-read the tab after preflight, which may have initialized
      // `databaseQueryContexts` via `updateTab`.
      const freshState = getSQLEditorTabsState();
      const freshTab = freshState.tabsById.get(freshState.currentTabId);
      if (!freshTab) {
        return;
      }
      const existingContexts: Map<string, SQLEditorDatabaseQueryContext[]> =
        freshTab.databaseQueryContexts ?? new Map();
      const batchQueryDatabaseSet = new Set<string /* database name */>([
        params.connection.database,
      ]);

      // Check if the user selects multiple databases to query.
      const appStore = useAppStore.getState();
      if (
        freshTab.batchQueryContext &&
        appStore.hasFeature(PlanFeature.FEATURE_BATCH_QUERY)
      ) {
        const { databases = [], databaseGroups = [] } =
          freshTab.batchQueryContext;
        for (const databaseResourceName of databases) {
          if (!isValidDatabaseName(databaseResourceName)) {
            continue;
          }
          if (batchQueryDatabaseSet.has(databaseResourceName)) {
            continue;
          }
          batchQueryDatabaseSet.add(databaseResourceName);
        }

        if (appStore.hasFeature(PlanFeature.FEATURE_DATABASE_GROUPS)) {
          for (const databaseGroupName of databaseGroups) {
            try {
              const databaseGroup = await useAppStore
                .getState()
                .fetchDBGroup(databaseGroupName, DatabaseGroupView.FULL);
              if (!databaseGroup) continue;
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

      // Cancel and drop contexts whose database is no longer in the batch.
      // Cancellation happens first (while contexts are still in the store)
      // so the abort + status update propagate to subscribers; then the
      // database key is removed via the store action.
      for (const [database, contexts] of existingContexts.entries()) {
        if (!batchQueryDatabaseSet.has(database)) {
          for (const context of contexts) {
            changeContextStatus(database, context, "CANCELLED");
          }
          freshState.deleteDatabaseQueryContext(database);
        }
      }

      const isBatch = batchQueryDatabaseSet.size > 1;
      await useAppStore
        .getState()
        .batchGetOrFetchDatabases([...batchQueryDatabaseSet.keys()]);

      for (const databaseName of batchQueryDatabaseSet.values()) {
        // Re-read the latest tab snapshot inside the loop so each
        // iteration sees the prior iteration's writes.
        const loopState = getSQLEditorTabsState();
        const loopTab = loopState.tabsById.get(loopState.currentTabId);
        if (!loopTab) {
          break;
        }
        const currentMap: Map<string, SQLEditorDatabaseQueryContext[]> =
          loopTab.databaseQueryContexts ?? new Map();
        const currentList = currentMap.get(databaseName) ?? [];

        // If at capacity, cancel + drop the oldest entry first.
        let trimmedList = currentList;
        if (currentList.length >= 50) {
          const oldest = currentList[currentList.length - 1];
          if (oldest) {
            changeContextStatus(databaseName, oldest, "CANCELLED");
            trimmedList = currentList.slice(0, currentList.length - 1);
          }
        }

        const database = useAppStore.getState().getDatabaseByName(databaseName);
        const resolvedDataSourceId =
          isBatch && loopTab.batchQueryContext.dataSourceType
            ? ((await getValidDataSourceByPolicy(
                database,
                loopTab.batchQueryContext.dataSourceType
              )) ?? "")
            : params.connection.dataSourceId;
        const context: SQLEditorDatabaseQueryContext = {
          id: uuidv4(),
          params: Object.assign(cloneDeep(params), {
            connection: {
              ...params.connection,
              ...(resolvedDataSourceId
                ? { dataSourceId: resolvedDataSourceId }
                : {}),
            },
          }),
          status: "PENDING",
        };

        const nextMap = new Map(currentMap);
        nextMap.set(databaseName, [context, ...trimmedList]);
        loopState.updateTab(loopTab.id, { databaseQueryContexts: nextMap });
      }
    },
    [preflight]
  );

  const runQuery = useCallback(
    async (database: Database, context: SQLEditorDatabaseQueryContext) => {
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

      changeContextStatus(database.name, context, "EXECUTING");

      const finish = (resultSet: SQLResultSetV1) => {
        const next = getSQLEditorTabsState().updateDatabaseQueryContext({
          database: database.name,
          contextId: context.id,
          context: { resultSet },
        });
        if (!next) {
          // Ad-hoc context: mirror onto the local object.
          context.resultSet = resultSet;
        }
        changeContextStatus(database.name, context, "DONE");
      };

      // After the EXECUTING transition, re-read the live context from
      // the store so the abortController set during the transition is
      // visible. Resolve by (database, contextId) across tabs — not the
      // current tab — so a query whose tab was switched away still finds
      // its own context. For ad-hoc contexts the store action bails and
      // the `changeContextStatus` fallback wrote directly into `context`.
      const liveContext =
        getDatabaseQueryContext(database.name, context.id) ?? context;
      const { abortController } = liveContext;
      if (!abortController) {
        return;
      }

      const dataSourceId = context.params.connection.dataSourceId;

      if (abortController.signal.aborted) {
        // Once any one of the batch queries is aborted, don't go further
        // and mock an "Aborted" result for the rest queries.
        return finish({
          error: t("sql-editor.request-aborted"),
          results: [],
          status: Code.Aborted,
        });
      }

      const editorState = getSQLEditorEditorState();
      const queryOption = create(QueryOptionSchema, {
        ...(context.params.queryOption ?? ({} as QueryOption)),
        redisRunCommandsOn: editorState.redisCommandOption,
      });
      const resultSet = await useAppStore.getState().query(
        create(QueryRequestSchema, {
          name: database.name,
          ...(dataSourceId ? { dataSourceId } : {}),
          statement: context.params.statement,
          limit: context.params.limit ?? editorState.resultRowsLimit,
          explain: context.params.explain,
          schema: context.params.connection.schema,
          container: context.params.connection.table,
          queryOption: queryOption,
        }),
        abortController.signal
      );

      // Merge the freshly-executed statement into the history cache
      // WITHOUT resetting pagination — the user keeps whatever pages
      // they had already loaded ("Load more"d), and the new entry just
      // gets prepended. After the cache update lands, emit the event so
      // the HistoryPane re-renders from it (store reactivity alone
      // doesn't reliably propagate into the React subscriber, so we
      // trigger the re-render explicitly).
      useSQLEditorReactStore
        .getState()
        .mergeLatest({
          project: getSQLEditorEditorState().project,
          database: database.name,
        })
        .catch(() => {
          /* nothing */
        })
        .finally(() => {
          void sqlEditorEvents.emit("query-executed");
        });

      const instanceResource = getInstanceResource(database);
      if (instanceResource.engine === Engine.COSMOSDB) {
        flattenNoSQLResult(resultSet);
      }

      if (isDisallowChangeDatabaseError(resultSet)) {
        // Show a tips to navigate to issue creation
        // if the user is allowed to create issue in the project.
        if (
          hasPermissionToCreateChangeDatabaseIssueInProject(
            getDatabaseProject(database)
          )
        ) {
          const liveEditorState = getSQLEditorEditorState();
          liveEditorState.setShowExecutingHint(true);
          liveEditorState.setExecutingHintDatabase(database);
        }
        return finish(resultSet);
      }

      return finish(resultSet);
    },
    [t]
  );

  const execute = useCallback(
    async (params: SQLEditorQueryParams) => {
      return preExecute(params);
    },
    [preExecute]
  );

  return {
    execute,
    runQuery,
  };
};

export const isDisallowChangeDatabaseError = (resultSet: SQLResultSetV1) => {
  const isCommandError = resultSet.results.some((result) => {
    return (
      result.detailedError.case === "commandError" &&
      [
        QueryResult_CommandError_Type.DDL,
        QueryResult_CommandError_Type.DML,
        QueryResult_CommandError_Type.NON_READ_ONLY,
      ].includes(result.detailedError.value.commandType)
    );
  });
  if (isCommandError) {
    return true;
  }

  return resultSet.results.some((result) => {
    if (result.detailedError.case === "permissionDenied") {
      return result.detailedError.value.requiredPermissions.some((p) => {
        return p === "bb.sql.ddl" || p === "bb.sql.dml";
      });
    }
    return false;
  });
};
