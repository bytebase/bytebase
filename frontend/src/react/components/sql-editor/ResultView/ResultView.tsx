import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import dayjs from "dayjs";
import { InfoIcon, LoaderCircle } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  DataExportButton,
  type DataExportRequest,
} from "@/react/components/DataExportButton";
import { RequestQueryButton } from "@/react/components/sql-editor/RequestQueryButton";
import { Button } from "@/react/components/ui/button";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Tooltip } from "@/react/components/ui/tooltip";
import { isDisallowChangeDatabaseError } from "@/react/hooks/useExecuteSQL";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useSQLStore,
} from "@/store";
import { usePolicyV1Store } from "@/store/modules/v1/policy";
import type { SQLEditorQueryParams, SQLResultSetV1 } from "@/types";
import { isValidDatabaseName } from "@/types";
import {
  ExportFormat,
  type PermissionDeniedDetail,
} from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { ExportRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import { buildDownloadBlob } from "@/utils/sql-download";
import { SQL_ENGINE_QUOTES } from "@/utils/sql-download/engines";
import { downloadErrorMessage } from "@/utils/sql-download/error-messages";
import { isDev } from "@/utils/util";
import {
  extractDatabaseResourceName,
  getDatabaseProject,
  getInstanceResource,
} from "@/utils/v1/database";
import { EmptyView } from "./EmptyView";
import { ErrorView } from "./ErrorView";
import { SingleResultView } from "./SingleResultView";

export interface ResultViewProps {
  executeParams: SQLEditorQueryParams;
  database: Database;
  resultSet?: SQLResultSetV1;
  loading?: boolean;
  dark?: boolean;
}

type ViewMode = "SINGLE-RESULT" | "MULTI-RESULT" | "EMPTY" | "ERROR";

/**
 * Top-level wrapper for one database's query result. Routes to a single
 * `SingleResultView`, a multi-tab list of them, an empty placeholder, or
 * a result-set-level error view (with optional access-request /
 * sync-database affordances). Phase 7's caller swap mounts this via
 * `<ReactPageMount page="ResultView" ...>`.
 */
export function ResultView({
  executeParams,
  database,
  resultSet,
  loading,
  dark = false,
}: ResultViewProps) {
  const { t } = useTranslation();
  const policyStore = usePolicyV1Store();
  // Hoist the editor state singleton so the useVueState getters below don't
  // appear to call a React Hook inside a callback (Sonar S6440).
  const editorState = useSQLEditorVueState();
  const queryDataPolicy = useVueState(() => editorState.queryDataPolicy);
  // The editor's per-execution row limit. A result whose .rows.length
  // exactly equals this is probably truncated; fewer rows is probably
  // complete. Used by the dev-path export to WARN only when the user's
  // requested limit can't be satisfied from the cached result.
  const resultRowsLimit = useVueState(() => editorState.resultRowsLimit);
  const tabStore = useSQLEditorTabStore();

  const permissionDeniedError = useMemo<
    PermissionDeniedDetail | undefined
  >(() => {
    if (!resultSet || isDisallowChangeDatabaseError(resultSet)) {
      return undefined;
    }
    for (const result of resultSet.results) {
      if (result.detailedError.case === "permissionDenied") {
        return result.detailedError.value;
      }
    }
    return undefined;
  }, [resultSet]);

  const viewMode: ViewMode = useMemo(() => {
    if (!resultSet) return "EMPTY";
    if (resultSet.error) return "ERROR";
    const results = resultSet.results ?? [];
    if (results.length === 0) return "EMPTY";
    if (results.length === 1) return "SINGLE-RESULT";
    return "MULTI-RESULT";
  }, [resultSet]);

  const showPlaceholder = useMemo(() => {
    if (viewMode === "ERROR") return false;
    if (!resultSet) return true;
    if (loading) return true;
    return false;
  }, [viewMode, resultSet, loading]);

  const disallowCopyingData = useMemo(() => {
    if (queryDataPolicy?.disableCopyData) return true;
    const environment = database.effectiveEnvironment;
    if (
      environment &&
      policyStore.getQueryDataPolicyByParent(environment).disableCopyData
    ) {
      return true;
    }
    return false;
  }, [queryDataPolicy, database, policyStore]);

  const filteredResults = useMemo(() => {
    if (!resultSet) return [];
    return resultSet.results.filter(
      (r) => !r.statement.trim().toUpperCase().startsWith("SET")
    );
  }, [resultSet]);

  const tabName = (index: number) => `${t("common.query")} #${index + 1}`;

  // Format list for the export drawer. Under the isDev() gate, drop SQL when
  // the database's engine can't be serialized as INSERTs — otherwise
  // selecting SQL would reach serializeSQL and throw UnsupportedFormat at
  // runtime. Production (backend Export RPC) handles all engines.
  const supportFormats = useMemo(() => {
    const all = [
      ExportFormat.CSV,
      ExportFormat.JSON,
      ExportFormat.SQL,
      ExportFormat.XLSX,
    ];
    if (!isDev()) return all;
    const engine = getInstanceResource(database).engine;
    return SQL_ENGINE_QUOTES.has(engine)
      ? all
      : [ExportFormat.CSV, ExportFormat.JSON, ExportFormat.XLSX];
  }, [database]);

  const handleExport = async (
    req: DataExportRequest & { statement: string }
  ) => {
    const { options, resolve, reject, statement } = req;

    // === Dev path: client-side ZIP via buildDownloadBlob ===
    // Production builds keep using the backend Export RPC below until the
    // client-side download module ships GA.
    if (isDev()) {
      try {
        // Guard against stale format selection: DataExportButton keeps its
        // `format` state across engine changes, so a user who picked SQL
        // before an engine switch would still submit SQL here.
        if (!supportFormats.includes(options.format)) {
          reject(
            "The selected format is not supported for the current database engine. Pick a different format."
          );
          return;
        }
        const { databaseName, instanceName } = extractDatabaseResourceName(
          database.name
        );
        const engine = getInstanceResource(database).engine;
        const candidates =
          viewMode === "MULTI-RESULT"
            ? filteredResults
            : (resultSet?.results?.slice(0, 1) ?? []);
        // Abort the whole export if any sub-result errored, matching the
        // prod RPC's behavior in doExport (sql_service.go) which aborts on
        // the first errored result. Skipping would produce a "successful"
        // download silently missing the failed statement's data.
        const erroredResult = candidates.find((r) => r.error);
        if (erroredResult) {
          reject(erroredResult.error);
          return;
        }
        if (candidates.length === 0) {
          reject(t("sql-editor.batch-export.no-results"));
          return;
        }
        const sourceResults = candidates;
        const baseFilename = `${databaseName}.${dayjs().format(
          "YYYY-MM-DDTHH-mm-ss"
        )}`;
        // Apply the export drawer's row limit per-result, mirroring the
        // server-side LIMIT clause semantics. The dev path operates on
        // already-fetched rows; when the user-requested limit exceeds the
        // cached count we can only serve what we have.
        const limit = options.limit > 0 ? options.limit : Infinity;
        // Probable-truncation inference: compare cached row count against
        // the limit IN EFFECT AT EXECUTION TIME, captured in
        // `executeParams.limit`. The current editor `resultRowsLimit` is a
        // snapshot of when the user clicked Export (they may have changed
        // it between Run and Export), so it can't be the ground truth.
        const executedLimit = executeParams.limit ?? resultRowsLimit;
        if (
          options.limit > 0 &&
          sourceResults.some(
            (r) =>
              r.rows.length === executedLimit && options.limit > r.rows.length
          )
        ) {
          pushNotification({
            module: "bytebase",
            style: "WARN",
            title: t("sql-editor.batch-export.failed-for-db", {
              db: databaseName,
            }),
            description: `Export limit ${options.limit} exceeds the executed query's row limit (${executedLimit}); exporting cached rows only.`,
          });
        }
        const out = await buildDownloadBlob({
          groups: [
            {
              instanceId: instanceName,
              databaseName,
              engine,
              statements: sourceResults.map((r) => ({
                result:
                  r.rows.length > limit
                    ? { ...r, rows: r.rows.slice(0, limit) }
                    : r,
                // For multi-result, `r.statement` is the substatement the
                // backend split out — that's what the user should see. For
                // single-result, prefer the caller's verbatim `statement`
                // (the per-result form may have an auto-appended LIMIT).
                statement:
                  viewMode === "MULTI-RESULT"
                    ? r.statement || statement
                    : statement || r.statement,
              })),
            },
          ],
          format: options.format,
          baseFilename,
          password: options.password,
        });
        const content = new Uint8Array(await out.blob.arrayBuffer());
        resolve([{ content, filename: out.filename }]);
      } catch (e) {
        reject(downloadErrorMessage(e, t));
      }
      return;
    }

    // === Prod path: backend Export RPC ===
    const admin = tabStore.currentTab?.mode === "ADMIN";
    try {
      const content = await useSQLStore().exportData(
        create(ExportRequestSchema, {
          name: database.name,
          ...(executeParams.connection.dataSourceId
            ? { dataSourceId: executeParams.connection.dataSourceId }
            : {}),
          format: options.format,
          statement,
          limit: options.limit,
          admin,
          password: options.password,
          schema: executeParams.connection.schema,
        })
      );
      resolve([
        {
          content,
          filename: `${extractDatabaseResourceName(database.name).databaseName}.${dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss")}.zip`,
        },
      ]);
    } catch (e) {
      reject(e);
    }
  };

  const errorViewSuffix = (statement?: string): ReactNode => {
    if (permissionDeniedError) {
      return (
        <RequestQueryButton
          text={false}
          statement={statement}
          permissionDeniedDetail={permissionDeniedError}
        />
      );
    }
    if (
      viewMode === "ERROR" &&
      resultSet?.error.includes("resource not found")
    ) {
      return <SyncDatabaseButton database={database} />;
    }
    return null;
  };

  return (
    <div
      className={cn(
        "relative flex flex-col justify-start items-start pb-1 overflow-y-auto h-full w-full",
        dark && "dark bg-dark-bg"
      )}
    >
      {executeParams && resultSet && !showPlaceholder && (
        <>
          {viewMode === "SINGLE-RESULT" &&
            (resultSet.results[0]?.error ? (
              <ErrorView
                dark={dark}
                error={resultSet.results[0].error}
                executeParams={executeParams}
                resultSet={resultSet}
                suffix={errorViewSuffix(executeParams.statement)}
              />
            ) : (
              <SingleResultView
                dark={dark}
                disallowCopyingData={disallowCopyingData}
                params={executeParams}
                database={database}
                result={resultSet.results[0]}
                showExport={!queryDataPolicy?.disableExport}
                maximumExportCount={queryDataPolicy?.maximumResultRows}
                onExport={handleExport}
              />
            ))}

          {viewMode === "MULTI-RESULT" && (
            <Tabs
              defaultValue={tabName(0)}
              className="flex-1 flex flex-col overflow-hidden w-full"
            >
              <div className="flex items-center justify-between gap-x-2">
                <TabsList>
                  {filteredResults.map((result, i) => (
                    <Tooltip key={i} content={result.statement}>
                      <TabsTrigger value={tabName(i)}>
                        <div className="flex items-center gap-x-2 mb-1">
                          <span>{tabName(i)}</span>
                          {result.error && (
                            <InfoIcon className="size-4 text-warning" />
                          )}
                        </div>
                      </TabsTrigger>
                    </Tooltip>
                  ))}
                </TabsList>
                {!queryDataPolicy?.disableExport && (
                  <div className="mb-1">
                    <DataExportButton
                      size="sm"
                      disabled={false}
                      supportFormats={supportFormats}
                      viewMode="DRAWER"
                      supportPassword
                      maximumExportCount={queryDataPolicy?.maximumResultRows}
                      onExport={(req) =>
                        handleExport({
                          ...req,
                          statement: executeParams.statement,
                        })
                      }
                    />
                  </div>
                )}
              </div>
              {filteredResults.map((result, i) => (
                <TabsPanel
                  key={i}
                  value={tabName(i)}
                  className="flex-1 flex flex-col overflow-hidden"
                >
                  {result.error ? (
                    <ErrorView
                      dark={dark}
                      error={result.error}
                      executeParams={executeParams}
                      resultSet={resultSet}
                      suffix={errorViewSuffix(result.statement)}
                    />
                  ) : (
                    <SingleResultView
                      dark={dark}
                      disallowCopyingData={disallowCopyingData}
                      params={executeParams}
                      database={database}
                      result={result}
                      showExport={false}
                      maximumExportCount={queryDataPolicy?.maximumResultRows}
                      onExport={handleExport}
                    />
                  )}
                </TabsPanel>
              ))}
            </Tabs>
          )}

          {viewMode === "EMPTY" && <EmptyView dark={dark} />}

          {viewMode === "ERROR" && (
            <ErrorView
              dark={dark}
              error={resultSet.error}
              executeParams={executeParams}
              resultSet={resultSet}
              suffix={errorViewSuffix(resultSet.results[0]?.statement)}
            />
          )}
        </>
      )}

      {showPlaceholder && (
        <div
          className={cn(
            "absolute inset-0 flex flex-col justify-center items-center z-10",
            loading && "bg-overlay/50"
          )}
        >
          {loading ? (
            <>
              <LoaderCircle className="size-6 animate-spin text-control-light" />
              {t("sql-editor.loading-data")}
            </>
          ) : !resultSet ? (
            t("sql-editor.table-empty-placeholder")
          ) : null}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Inline SyncDatabaseButton — replaces frontend/src/components/DatabaseDetail/SyncDatabaseButton.vue.
// Only used by the result-set-level "resource not found" branch above.
// ---------------------------------------------------------------------------

function SyncDatabaseButton({ database }: { database: Database }) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const dbSchemaStore = useDBSchemaV1Store();
  const [syncing, setSyncing] = useState(false);

  if (!isValidDatabaseName(database.name)) return null;

  const project = getDatabaseProject(database);
  const canSync = hasProjectPermissionV2(project, "bb.databases.sync");
  if (!canSync) return null;

  const handleSync = async () => {
    setSyncing(true);
    const { databaseName } = extractDatabaseResourceName(database.name);
    try {
      await databaseStore.syncDatabase(database.name);
      await dbSchemaStore.getOrFetchDatabaseMetadata({
        database: database.name,
        skipCache: true,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "db.successfully-synced-schema-for-database-database-value-name",
          { name: databaseName }
        ),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("db.failed-to-sync-schema-for-database-database-value-name", {
          name: databaseName,
        }),
        description:
          error instanceof ConnectError
            ? error.message
            : (error as Error)?.message,
      });
    } finally {
      setSyncing(false);
    }
  };

  return (
    <Button size="sm" variant="link" disabled={syncing} onClick={handleSync}>
      {syncing && <LoaderCircle className="size-3 animate-spin" />}
      {t("database.sync-database")}
    </Button>
  );
}
