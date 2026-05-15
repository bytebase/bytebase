import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import dayjs from "dayjs";
import { InfoIcon, LoaderCircle } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { isDisallowChangeDatabaseError } from "@/composables/useExecuteSQL";
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
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  pushNotification,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
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
  const policyStore = usePolicyV1Store();
  const queryDataPolicy = useVueState(
    () => useSQLEditorStore().queryDataPolicy
  );
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

  const handleExport = async (
    req: DataExportRequest & { statement: string }
  ) => {
    const { options, resolve, reject, statement } = req;
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
                      supportFormats={[
                        ExportFormat.CSV,
                        ExportFormat.JSON,
                        ExportFormat.SQL,
                        ExportFormat.XLSX,
                      ]}
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
