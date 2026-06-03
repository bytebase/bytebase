import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import dayjs from "dayjs";
import { InfoIcon, LoaderCircle } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  DataExportButton,
  type DataExportRequest,
} from "@/react/components/DataExportButton";
import { RequestExportButton } from "@/react/components/sql-editor/RequestExportButton";
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
import { useSQLEditorQueryDataPolicy } from "@/react/hooks/useSQLEditorBridge";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { getSQLEditorTabsState } from "@/react/stores/sqlEditor/tab";
import type { SQLEditorQueryParams, SQLResultSetV1 } from "@/types";
import { isValidDatabaseName } from "@/types";
import {
  ExportFormat,
  type PermissionDeniedDetail,
} from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { ExportRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import {
  extractDatabaseResourceName,
  getDatabaseProject,
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
  const project = useSQLEditorEditorState((s) => s.project);
  const queryDataPolicy = useSQLEditorQueryDataPolicy(project);
  // Env-level data-query policy via the app store. Subscribe to the
  // derived `QueryDataPolicy` directly — the slice returns a stable empty
  // singleton when nothing is cached, so this is safe for
  // `useSyncExternalStore` snapshot comparisons.
  const environment = database.effectiveEnvironment;
  const envQueryDataPolicy = useAppStore((s) =>
    environment ? s.getQueryDataPolicyByParent(environment) : undefined
  );
  const getOrFetchPolicyByParentAndType = useAppStore(
    (s) => s.getOrFetchPolicyByParentAndType
  );
  // Settings pages populate the env policy in Pinia, but the SQL editor
  // route doesn't fetch it on its own — self-fetch so the read above
  // resolves to a real policy (not the empty fallback) and copy-disable
  // gates fire even on a fresh editor visit.
  useEffect(() => {
    if (!environment) return;
    void getOrFetchPolicyByParentAndType({
      parentPath: environment,
      policyType: PolicyType.DATA_QUERY,
    });
  }, [environment, getOrFetchPolicyByParentAndType]);

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
    if (envQueryDataPolicy?.disableCopyData) return true;
    return false;
  }, [queryDataPolicy, envQueryDataPolicy]);

  // Show the real export button when the policy allows export, OR when the
  // user holds any active export-capable grant for this statement — even if
  // it's not the grant the Query path applied (Query prefers Unmask, so the
  // applied grant may be unmask-only while a separate export grant exists).
  // Without this independent search the UI would hide the Export button and
  // direct the user to "Request export" despite already having a grant — see
  // PR #20491 bot review (#3349086832).
  const searchMyAccessGrants = useAppStore((s) => s.searchMyAccessGrants);
  const databaseProjectName = database.project;
  const [exportGrantName, setExportGrantName] = useState<string>("");

  useEffect(() => {
    if (
      !queryDataPolicy?.disableExport ||
      !executeParams?.statement ||
      !databaseProjectName
    ) {
      setExportGrantName("");
      return;
    }
    let canceled = false;
    void (async () => {
      const result = await searchMyAccessGrants({
        parent: databaseProjectName,
        filter: {
          target: database.name,
          // Exact match — the backend's JIT authorization path uses
          // `query == ...` (preCheckAccess in sql_service.go). A
          // substring match (`statement: ...`) would expose Export for
          // queries that don't actually match any grant. PR #20491 bot
          // review #3349385091.
          statementExact: executeParams.statement,
          status: ["ACTIVE"],
          export: true,
        },
      });
      if (!canceled) {
        setExportGrantName(result.accessGrants[0]?.name ?? "");
      }
    })();
    return () => {
      canceled = true;
    };
  }, [
    queryDataPolicy?.disableExport,
    executeParams?.statement,
    databaseProjectName,
    database.name,
    searchMyAccessGrants,
  ]);

  // Pull display fields from the cache populated by the search above.
  const exportGrantIssue = useAppStore((s) =>
    exportGrantName ? (s.accessGrantsByName[exportGrantName]?.issue ?? "") : ""
  );
  const exportGrantReason = useAppStore((s) =>
    exportGrantName ? (s.accessGrantsByName[exportGrantName]?.reason ?? "") : ""
  );

  const showExport = !queryDataPolicy?.disableExport || !!exportGrantName;

  // Surface a tooltip explaining the grant-based bypass only when the policy
  // itself would normally block export — in the everyday "policy allows
  // export" case, no tooltip is needed. Attributes to the export-capable
  // grant (which may differ from the Query-applied grant when the user has
  // separate unmask + export grants for the same statement).
  const exportTooltip = useMemo<ReactNode>(() => {
    if (!queryDataPolicy?.disableExport || !exportGrantName) {
      return undefined;
    }
    const issueHref = exportGrantIssue
      ? exportGrantIssue.startsWith("/")
        ? exportGrantIssue
        : `/${exportGrantIssue}`
      : undefined;
    return (
      <div className="flex flex-col gap-y-1">
        <span>{t("sql-editor.export-enabled-by-grant")}</span>
        {issueHref ? (
          <a
            href={issueHref}
            target="_blank"
            rel="noreferrer"
            className="break-all underline"
            onClick={(e) => e.stopPropagation()}
          >
            {exportGrantName}
          </a>
        ) : (
          <span className="break-all">{exportGrantName}</span>
        )}
        {exportGrantReason && (
          <span className="text-xs opacity-80">{exportGrantReason}</span>
        )}
      </div>
    );
  }, [
    t,
    queryDataPolicy?.disableExport,
    exportGrantName,
    exportGrantIssue,
    exportGrantReason,
  ]);

  // When direct export is unavailable, offer a "Request export" affordance that
  // opens the access-grant drawer (pre-filled with this database, statement,
  // and unmask + export checked). The button self-hides when the project
  // doesn't allow just-in-time access.
  const requestExportButton = executeParams ? (
    <RequestExportButton
      statement={executeParams.statement}
      targets={[database.name]}
    />
  ) : null;

  const filteredResults = useMemo(() => {
    if (!resultSet) return [];
    return resultSet.results.filter(
      (r) => !r.statement.trim().toUpperCase().startsWith("SET")
    );
  }, [resultSet]);

  const tabName = (index: number) => `${t("common.query")} #${index + 1}`;

  const supportFormats = useMemo(
    () => [
      ExportFormat.CSV,
      ExportFormat.JSON,
      ExportFormat.SQL,
      ExportFormat.XLSX,
    ],
    []
  );

  const handleExport = async (
    req: DataExportRequest & { statement: string }
  ) => {
    const { options, resolve, reject, statement } = req;

    // === Prod path: backend Export RPC ===
    const tabsState = getSQLEditorTabsState();
    const admin =
      tabsState.tabsById.get(tabsState.currentTabId)?.mode === "ADMIN";
    try {
      const content = await useAppStore.getState().exportData(
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
                showExport={showExport}
                exportTooltip={exportTooltip}
                maximumExportCount={queryDataPolicy?.maximumResultRows}
                onExport={handleExport}
                requestExportSlot={
                  !showExport ? requestExportButton : undefined
                }
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
                {showExport ? (
                  <div className="mb-1">
                    <DataExportButton
                      size="sm"
                      disabled={false}
                      supportFormats={supportFormats}
                      viewMode="DRAWER"
                      supportPassword
                      tooltip={exportTooltip}
                      maximumExportCount={queryDataPolicy?.maximumResultRows}
                      onExport={(req) =>
                        handleExport({
                          ...req,
                          statement: executeParams.statement,
                        })
                      }
                    />
                  </div>
                ) : (
                  <div className="mb-1">{requestExportButton}</div>
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
  const [syncing, setSyncing] = useState(false);

  if (!isValidDatabaseName(database.name)) return null;

  const project = getDatabaseProject(database);
  const canSync = hasProjectPermissionV2(project, "bb.databases.sync");
  if (!canSync) return null;

  const handleSync = async () => {
    setSyncing(true);
    const { databaseName } = extractDatabaseResourceName(database.name);
    try {
      const appStore = useAppStore.getState();
      await appStore.syncDatabase(database.name);
      await appStore.getOrFetchDatabaseMetadata({
        database: database.name,
        skipCache: true,
      });
      useAppStore.getState().notify({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "db.successfully-synced-schema-for-database-database-value-name",
          { name: databaseName }
        ),
      });
    } catch (error) {
      useAppStore.getState().notify({
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
