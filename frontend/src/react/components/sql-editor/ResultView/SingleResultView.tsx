import { create } from "@bufbuild/protobuf";
import { isEmpty } from "lodash-es";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  ChevronRightIcon,
  CopyIcon,
  InfoIcon,
  XIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import {
  flattenElasticsearchSearchResult,
  flattenNoSQLQueryResult,
} from "@/composables/utils";
import {
  AdvancedSearch,
  type ScopeOption,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import {
  DataExportButton,
  type DataExportRequest,
} from "@/react/components/DataExportButton";
import { EngineIcon } from "@/react/components/EngineIcon";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import { Switch } from "@/react/components/ui/switch";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
import type {
  SQLEditorDatabaseQueryContext,
  SQLEditorQueryParams,
} from "@/types";
import { Engine, ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryOption_MSSQLExplainFormat,
  QueryOptionSchema,
  type QueryResult,
  type RowValue,
} from "@/types/proto-es/v1/sql_service_pb";
import { createExplainToken } from "@/utils/pev2";
import { STORAGE_KEY_SQL_EDITOR_NOSQL_TABLE_VIEW } from "@/utils/storage-keys";
import { isNullOrUndefined } from "@/utils/util";
import {
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getInstanceResource,
} from "@/utils/v1/database";
import { instanceV1Name } from "@/utils/v1/instance";
import { compareQueryRowValues, extractSQLRowValuePlain } from "@/utils/v1/sql";
import { SQLResultViewProvider, useSelectionContext } from "./context";
import { DetailPanel } from "./DetailPanel";
import { EmptyView } from "./EmptyView";
import { ErrorView } from "./ErrorView";
import { SelectionCopyTooltips } from "./SelectionCopyTooltips";
import type { ResultTableColumn, ResultTableRow, SortState } from "./types";
import {
  VirtualDataBlock,
  type VirtualDataBlockHandle,
} from "./VirtualDataBlock";
import {
  VirtualDataTable,
  type VirtualDataTableHandle,
} from "./VirtualDataTable";

export interface SingleResultViewProps {
  dark: boolean;
  disallowCopyingData: boolean;
  params: SQLEditorQueryParams;
  database: Database;
  result: QueryResult;
  showExport: boolean;
  maximumExportCount?: number;
  onExport?: (req: DataExportRequest & { statement: string }) => void;
}

type ViewMode = "RESULT" | "EMPTY" | "AFFECTED-ROWS" | "ERROR";

/**
 * Per-result-set view. Owns the per-mount `<SQLResultViewProvider>` plus
 * the result toolbar (search, switches, export, copy-all), the
 * VirtualDataTable / VirtualDataBlock body, the floating scroll +
 * search-candidate buttons, the bottom status bar, and the DetailPanel
 * sheet. The wrapper does the per-result derivations once (active result,
 * columns, base row wrappers, NoSQL flatten + toggle) so the inner
 * component and the provider share the same arrays — earlier we derived
 * them twice and doubled the 100k-row allocation on large result sets.
 */
export function SingleResultView(props: SingleResultViewProps) {
  const { dark, disallowCopyingData, database, result } = props;
  const engine = getInstanceResource(database).engine;
  const [noSQLTableView, setNoSQLTableView] = useLocalStorageBoolean(
    STORAGE_KEY_SQL_EDITOR_NOSQL_TABLE_VIEW,
    true
  );
  // Sort state lives at this level (not inside the inner component) so the
  // sorted `rows` we feed into the provider are the same array the user
  // sees in the table. The provider's selection-copy logic dereferences
  // `rows[selectionIndex]` — if the provider held the unsorted array but
  // the table rendered sorted rows, Cmd+C would copy the wrong row.
  const [sortState, setSortState] = useState<SortState | undefined>();

  const flattened = useMemo(() => {
    if (engine === Engine.ELASTICSEARCH) {
      return flattenElasticsearchSearchResult(result);
    }
    if (engine === Engine.MONGODB) {
      return flattenNoSQLQueryResult(result);
    }
    return undefined;
  }, [engine, result]);
  const supportsTableViewToggle = flattened !== undefined;
  const activeResult =
    supportsTableViewToggle && noSQLTableView ? flattened : result;

  const columns: ResultTableColumn[] = useMemo(
    () =>
      activeResult.columnNames.map((columnName, index) => ({
        id: columnName,
        name: columnName,
        columnType: activeResult.columnTypeNames[index] ?? "",
      })),
    [activeResult]
  );

  const baseRows: ResultTableRow[] = useMemo(
    () =>
      activeResult.rows.map((item, index) => ({
        key: index,
        item,
      })),
    [activeResult]
  );

  const rows: ResultTableRow[] = useMemo(() => {
    if (!sortState || !sortState.direction) return baseRows;
    const { columnIndex, direction } = sortState;
    const columnType = columns[columnIndex]?.columnType ?? "";
    return [...baseRows].sort((a, b) => {
      const cmp = compareQueryRowValues(
        columnType,
        a.item.values[columnIndex],
        b.item.values[columnIndex]
      );
      return direction === "asc" ? cmp : -cmp;
    });
  }, [baseRows, sortState, columns]);

  // Reset sort whenever the user toggles the NoSQL table-view switch:
  // column indices don't carry over between flattened/raw shapes.
  useEffect(() => {
    setSortState(undefined);
  }, [noSQLTableView]);

  const toggleSort = useCallback((columnIndex: number) => {
    setSortState((current) => {
      if (!current || current.columnIndex !== columnIndex) {
        return { columnIndex, direction: "desc" };
      }
      if (current.direction === "desc") {
        return { columnIndex, direction: "asc" };
      }
      return undefined;
    });
  }, []);

  return (
    <SQLResultViewProvider
      dark={dark}
      disallowCopyingData={disallowCopyingData}
      rows={rows}
      columns={columns}
    >
      <SingleResultViewInner
        {...props}
        engine={engine}
        activeResult={activeResult}
        columns={columns}
        rows={rows}
        sortState={sortState}
        toggleSort={toggleSort}
        supportsTableViewToggle={supportsTableViewToggle}
        noSQLTableView={noSQLTableView}
        setNoSQLTableView={setNoSQLTableView}
      />
    </SQLResultViewProvider>
  );
}

interface SingleResultViewInnerProps extends SingleResultViewProps {
  engine: Engine;
  activeResult: QueryResult;
  columns: ResultTableColumn[];
  rows: ResultTableRow[];
  sortState: SortState | undefined;
  toggleSort: (columnIndex: number) => void;
  supportsTableViewToggle: boolean;
  noSQLTableView: boolean;
  setNoSQLTableView: (next: boolean) => void;
}

function SingleResultViewInner({
  dark,
  disallowCopyingData,
  params,
  database,
  result,
  showExport,
  maximumExportCount,
  onExport,
  engine,
  activeResult: _activeResult,
  columns,
  rows,
  sortState,
  toggleSort,
  supportsTableViewToggle,
  noSQLTableView,
  setNoSQLTableView,
}: SingleResultViewInnerProps) {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const editorStore = useSQLEditorVueState();
  const currentTabMode = useVueState(() => tabStore.currentTab?.mode);
  const resultRowsLimit = useVueState(() => editorStore.resultRowsLimit);
  const policyMaxRows = useVueState(
    () => editorStore.queryDataPolicy.maximumResultRows
  );
  const { runQuery } = useExecuteSQL();
  const { copyAll } = useSelectionContext();

  const dataTableRef = useRef<
    VirtualDataTableHandle | VirtualDataBlockHandle | null
  >(null);

  const [vertical, setVertical] = useState(false);
  const [searchParams, setSearchParams] = useState<SearchParams>({
    query: "",
    scopes: [],
  });
  const [searchCandidateActiveIndex, setSearchCandidateActiveIndex] =
    useState(-1);
  const [searchCandidateRowIndexs, setSearchCandidateRowIndexs] = useState<
    number[]
  >([]);

  // Reset search state when toggling NoSQL table view (sort reset is
  // handled by the wrapper, where sortState lives).
  useEffect(() => {
    setSearchParams({ query: "", scopes: [] });
    setSearchCandidateActiveIndex(-1);
    setSearchCandidateRowIndexs([]);
  }, [noSQLTableView]);

  const viewMode: ViewMode = useMemo(() => {
    if (result.error && rows.length === 0) return "ERROR";
    const columnNames = result.columnNames;
    if (columnNames?.length === 0) return "EMPTY";
    if (columnNames?.length === 1 && columnNames[0] === "Affected Rows") {
      return "AFFECTED-ROWS";
    }
    return "RESULT";
  }, [result, rows.length]);

  const searchScopeOptions: ScopeOption[] = useMemo(() => {
    const opts: ScopeOption[] = [
      {
        id: "row-number",
        title: t("sql-editor.search-scope-row-number-title"),
        description: t("sql-editor.search-scope-row-number-description"),
      },
    ];
    for (const column of columns) {
      opts.push({
        id: column.id,
        title: column.name,
        description: t("sql-editor.search-scope-column-description", {
          type: column.columnType,
        }),
      });
    }
    return opts;
  }, [columns, t]);

  // Recompute candidates when search params change.
  const cellValueMatches = useCallback(
    (cell: RowValue, query: string): boolean => {
      const value = extractSQLRowValuePlain(cell);
      if (isNullOrUndefined(value)) return false;
      return String(value).toLowerCase().includes(query.toLowerCase());
    },
    []
  );

  const getNextCandidateRowIndex = useCallback(
    (from: number, sp: SearchParams): number => {
      if (sp.scopes.length === 0 && !sp.query) return -1;
      for (let i = from; i < rows.length; i++) {
        const row = rows[i];
        const scopeOk = sp.scopes.every((scope) => {
          if (!scope.value) return false;
          if (scope.id === "row-number") {
            return i + 1 === Number.parseInt(scope.value, 10);
          }
          const columnIndex = columns.findIndex((c) => c.name === scope.id);
          if (columnIndex < 0) return false;
          return cellValueMatches(row.item.values[columnIndex], scope.value);
        });
        if (!scopeOk) continue;
        if (sp.query) {
          const queryOk = row.item.values.some((cell) =>
            cellValueMatches(cell, sp.query)
          );
          if (!queryOk) continue;
        }
        return i;
      }
      return -1;
    },
    [rows, columns, cellValueMatches]
  );

  useEffect(() => {
    const next = getNextCandidateRowIndex(0, searchParams);
    const indexes: number[] = [];
    if (next >= 0) {
      indexes.push(next);
      const another = getNextCandidateRowIndex(next + 1, searchParams);
      if (another >= 0) indexes.push(another);
    }
    setSearchCandidateRowIndexs(indexes);
    setSearchCandidateActiveIndex(0);
  }, [searchParams, getNextCandidateRowIndex]);

  const activeRowIndex =
    searchCandidateRowIndexs[searchCandidateActiveIndex] ?? -1;

  const scrollToRow = useCallback((index: number | undefined) => {
    if (index === undefined || index < 0) return;
    requestAnimationFrame(() => dataTableRef.current?.scrollTo(index));
  }, []);

  // Re-scroll when the active candidate or vertical mode changes.
  useEffect(() => {
    scrollToRow(activeRowIndex);
  }, [activeRowIndex, vertical, scrollToRow]);

  const scrollToNextCandidate = () => {
    if (searchCandidateActiveIndex >= searchCandidateRowIndexs.length - 1) {
      return;
    }
    const next = searchCandidateActiveIndex + 1;
    setSearchCandidateActiveIndex(next);
    if (next === searchCandidateRowIndexs.length - 1) {
      const cur = searchCandidateRowIndexs[next];
      const more = getNextCandidateRowIndex(cur + 1, searchParams);
      if (more >= 0) {
        setSearchCandidateRowIndexs((prev) => [...prev, more]);
      }
    }
  };

  const scrollToPreviousCandidate = () => {
    if (searchCandidateActiveIndex <= 0) return;
    setSearchCandidateActiveIndex(searchCandidateActiveIndex - 1);
  };

  const clearSearchCandidate = () => {
    setSearchParams({ query: "", scopes: [] });
  };

  const reachQueryLimit =
    currentTabMode !== "ADMIN" &&
    (rows.length === resultRowsLimit || rows.length === policyMaxRows);

  const isSensitiveColumn = useCallback(
    (columnIndex: number): boolean => {
      if (supportsTableViewToggle && noSQLTableView) return false;
      const reason = result.masked?.[columnIndex];
      return (
        reason !== null &&
        reason !== undefined &&
        reason.semanticTypeId !== undefined &&
        reason.semanticTypeId !== ""
      );
    },
    [result.masked, supportsTableViewToggle, noSQLTableView]
  );

  const getMaskingReason = useCallback(
    (columnIndex: number) => {
      if (supportsTableViewToggle && noSQLTableView) return undefined;
      if (!result.masked || columnIndex >= result.masked.length) {
        return undefined;
      }
      const reason = result.masked[columnIndex];
      if (!reason || !reason.semanticTypeId) return undefined;
      return reason;
    },
    [result.masked, supportsTableViewToggle, noSQLTableView]
  );

  const showVisualizeButton =
    (engine === Engine.POSTGRES ||
      engine === Engine.MSSQL ||
      engine === Engine.SPANNER) &&
    !!params.explain;

  const visualizeExplain = async () => {
    let token: string | undefined;
    try {
      if (engine === Engine.POSTGRES || engine === Engine.SPANNER) {
        token = getExplainTokenFromResult(result, engine);
      } else if (engine === Engine.MSSQL) {
        token = await getExplainTokenForMSSQL(database, params, runQuery);
      }
      if (!token) return;
      window.open(`/explain-visualizer.html?token=${token}`, "_blank");
    } catch {
      // ignore
    }
  };

  const queryTime = useMemo(() => {
    const { latency } = result;
    if (!latency) return "-";
    const totalSeconds = Number(latency.seconds) + latency.nanos / 1e9;
    if (totalSeconds < 1) {
      return `${Math.round(totalSeconds * 1000)} ms`;
    }
    return `${totalSeconds.toFixed(2)} s`;
  }, [result]);

  const resultRowsText = `${rows.length} ${t("sql-editor.rows.self")}`;

  const handleExport = (req: DataExportRequest) => {
    // Forward the user-typed query (`params.statement`) rather than
    // `result.statement`. The backend may rewrite the result statement
    // with an auto-appended LIMIT for non-admin reads — re-running that
    // rewritten SQL on the export path silently caps the exported rows
    // even when the user asks for more. This matches the Vue
    // multi-result export, which already used `executeParams.statement`.
    onExport?.({ ...req, statement: params.statement });
  };

  // ---- Render branches by viewMode ----

  if (viewMode === "AFFECTED-ROWS") {
    return (
      <div
        className={cn(
          "text-md font-normal flex items-center gap-x-1",
          dark ? "text-matrix-green-hover" : "text-control-light"
        )}
      >
        <span>{String(extractSQLRowValuePlain(result.rows[0].values[0]))}</span>
        <span>{t("sql-editor.rows-affected")}</span>
      </div>
    );
  }

  if (viewMode === "EMPTY") {
    return <EmptyView dark={dark} />;
  }

  return (
    <>
      {/* Pre-toolbar messages */}
      {result.messages.length > 0 && (
        <>
          {result.messages.map((message, i) => (
            <div key={`message-${i}`} className="text-control-light">
              <div>{`[${message.level}] ${message.content}`}</div>
            </div>
          ))}
        </>
      )}

      {viewMode === "RESULT" && (
        <>
          {result.error && (
            <Alert variant="error" className="w-full mb-2">
              <ErrorView dark={dark} error={result.error} />
            </Alert>
          )}

          {/* Toolbar */}
          <div className="result-toolbar relative w-full shrink-0 flex flex-row gap-x-4 justify-between items-center mb-2 hide-scrollbar">
            <div className="flex flex-row justify-start items-center gap-x-2 mr-2 flex-1">
              <AdvancedSearch
                params={searchParams}
                scopeOptions={searchScopeOptions}
                placeholder=""
                onParamsChange={setSearchParams}
                onEnter={scrollToNextCandidate}
              />
              <Tooltip
                content={
                  reachQueryLimit ? t("sql-editor.rows-upper-limit") : ""
                }
              >
                <div className="flex items-center gap-x-1 whitespace-nowrap text-sm text-control-light">
                  {reachQueryLimit && (
                    <InfoIcon className="size-4 text-warning" />
                  )}
                  {resultRowsText}
                </div>
              </Tooltip>
            </div>
            <div className="flex justify-between items-center shrink-0 gap-x-2">
              {supportsTableViewToggle && (
                <div className="flex items-center gap-x-1">
                  <Switch
                    checked={noSQLTableView}
                    onCheckedChange={setNoSQLTableView}
                  />
                  <span className="whitespace-nowrap text-sm text-control-light">
                    {t("sql-editor.table-view")}
                  </span>
                </div>
              )}
              <div className="flex items-center gap-x-1">
                <Switch checked={vertical} onCheckedChange={setVertical} />
                <span className="whitespace-nowrap text-sm text-control-light">
                  {t("sql-editor.vertical-display")}
                </span>
              </div>
              {!disallowCopyingData && rows.length > 0 && (
                // `variant="outline"` is `bg-transparent + text-control`, which
                // disappears inside the admin-mode dark backdrop. Force an
                // opaque light-on-dark surface in `.dark` to match the Vue
                // toolbar's contrast (light gray bg + dark text).
                <Button
                  size="sm"
                  variant="outline"
                  className="h-7 px-2 dark:bg-gray-700 dark:text-gray-100 dark:border-zinc-600 dark:hover:bg-gray-600"
                  onClick={copyAll}
                >
                  <CopyIcon className="size-4" />
                  {t("common.copy")}
                </Button>
              )}
              {showExport && (
                <DataExportButton
                  size="sm"
                  disabled={!result || isEmpty(result)}
                  supportFormats={[
                    ExportFormat.CSV,
                    ExportFormat.JSON,
                    ExportFormat.SQL,
                    ExportFormat.XLSX,
                  ]}
                  viewMode="DRAWER"
                  supportPassword
                  maximumExportCount={maximumExportCount}
                  formContent={<DatabaseInfo database={database} />}
                  onExport={handleExport}
                />
              )}
            </div>
            <SelectionCopyTooltips />
          </div>

          {/* Body */}
          <div
            className={cn(
              "w-full flex flex-col relative",
              dark ? "h-80 overflow-hidden" : "flex-1 min-h-0 overflow-y-auto"
            )}
          >
            {vertical ? (
              <VirtualDataBlock
                ref={dataTableRef as React.RefObject<VirtualDataBlockHandle>}
                rows={rows}
                columns={columns}
                isSensitiveColumn={isSensitiveColumn}
                getMaskingReason={getMaskingReason}
                database={database}
                statement={params.statement}
                activeRowIndex={activeRowIndex}
                search={searchParams}
              />
            ) : (
              <VirtualDataTable
                ref={dataTableRef as React.RefObject<VirtualDataTableHandle>}
                rows={rows}
                columns={columns}
                isSensitiveColumn={isSensitiveColumn}
                getMaskingReason={getMaskingReason}
                database={database}
                statement={params.statement}
                sortState={sortState}
                activeRowIndex={activeRowIndex}
                search={searchParams}
                onToggleSort={toggleSort}
              />
            )}

            {/* Floating buttons */}
            <div className="absolute bottom-2 right-4 flex items-end gap-x-2">
              {searchCandidateRowIndexs.length > 0 && (
                <div className="flex flex-row gap-x-2 border shadow rounded bg-background py-1 px-2">
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={searchCandidateActiveIndex <= 0}
                    onClick={scrollToPreviousCandidate}
                  >
                    <ArrowUpIcon className="size-4" />
                    {t("sql-editor.previous-row")}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={
                      searchCandidateActiveIndex >=
                      searchCandidateRowIndexs.length - 1
                    }
                    onClick={scrollToNextCandidate}
                  >
                    <ArrowDownIcon className="size-4" />
                    {t("sql-editor.next-row")}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="size-7 p-0"
                    onClick={clearSearchCandidate}
                  >
                    <XIcon className="size-4" />
                  </Button>
                </div>
              )}

              <div className="flex flex-col gap-y-2 result-scroll-buttons">
                <Tooltip content={t("sql-editor.scroll-to-top")}>
                  <div className="rounded-full shadow bg-background">
                    <Button
                      size="sm"
                      variant="ghost"
                      className="size-9 p-0 rounded-full"
                      onClick={() => scrollToRow(0)}
                    >
                      <ArrowUpIcon className="size-4" />
                    </Button>
                  </div>
                </Tooltip>
                <Tooltip content={t("sql-editor.scroll-to-bottom")}>
                  <div className="rounded-full shadow bg-background">
                    <Button
                      size="sm"
                      variant="ghost"
                      className="size-9 p-0 rounded-full"
                      onClick={() => scrollToRow(rows.length - 1)}
                    >
                      <ArrowDownIcon className="size-4" />
                    </Button>
                  </div>
                </Tooltip>
              </div>
            </div>
          </div>

          {/* Status bar */}
          <div className="w-full flex items-center justify-between text-xs mt-1 gap-x-4 text-control-light">
            <div className="flex items-center gap-x-2">
              <RichDatabaseName database={database} />
              <div className="flex items-center gap-x-1 min-w-0">
                <EllipsisText
                  text={result.statement ?? ""}
                  className="truncate"
                />
                <CopyInlineButton text={result.statement ?? ""} />
              </div>
            </div>
            <div className="flex shrink-0 items-center gap-x-2">
              {showVisualizeButton && (
                <Button
                  size="sm"
                  variant="link"
                  className="h-auto px-0 text-xs"
                  onClick={visualizeExplain}
                >
                  {t("sql-editor.visualize-explain")}
                </Button>
              )}
              <span>
                {t("sql-editor.query-time")}: {queryTime}
              </span>
            </div>
          </div>
        </>
      )}

      <DetailPanel rows={rows} columns={columns} />
    </>
  );
}

// ---------------------------------------------------------------------------
// Inline helpers — small enough to live in this file.
// ---------------------------------------------------------------------------

function useLocalStorageBoolean(
  key: string,
  defaultValue: boolean
): [boolean, (next: boolean) => void] {
  const [value, setValue] = useState<boolean>(() => {
    try {
      const raw = localStorage.getItem(key);
      if (raw === "true") return true;
      if (raw === "false") return false;
    } catch {
      // ignore
    }
    return defaultValue;
  });
  const update = useCallback(
    (next: boolean) => {
      setValue(next);
      try {
        localStorage.setItem(key, String(next));
      } catch {
        // ignore
      }
    },
    [key]
  );
  return [value, update];
}

/**
 * Inline simplified renderer mirroring the visible output of Vue
 * `RichDatabaseName` with default props: engine icon + instance title +
 * chevron + environment + database. Skips the popover-on-hover branch
 * (only used by `tooltip="instance"` callers; result-view doesn't set it).
 */
function RichDatabaseName({ database }: { database: Database }) {
  const instance = getInstanceResource(database);
  const environment = getDatabaseEnvironment(database);
  const { databaseName } = extractDatabaseResourceName(database.name);
  return (
    <div className="flex flex-row justify-start items-center gap-x-1">
      <EngineIcon engine={instance.engine} className="size-4" />
      <span>{instanceV1Name(instance)}</span>
      <ChevronRightIcon className="size-3" />
      <span className="flex flex-row items-center gap-x-1">
        <span className="text-control-light">{environment.title}</span>
        <span>{databaseName}</span>
      </span>
    </div>
  );
}

/**
 * Inline simplified `DatabaseInfo` for the export drawer's form-prefix slot.
 * Vue version showed `[engine] instance > [env] database`; we mirror that.
 */
function DatabaseInfo({ database }: { database: Database }) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-1 mb-3">
      <span className="text-xs text-control-light">{t("common.database")}</span>
      <RichDatabaseName database={database} />
    </div>
  );
}

function CopyInlineButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };
  return (
    <button
      type="button"
      onClick={handleCopy}
      className="text-control-light hover:text-control"
      aria-label="copy"
    >
      <CopyIcon className={cn("size-3", copied && "text-accent")} />
    </button>
  );
}

function getExplainTokenFromResult(
  result: QueryResult,
  engine: Engine
): string | undefined {
  const { statement } = result;
  if (!statement) return undefined;
  const lines = result.rows.map((row) =>
    row.values.map((value) => String(extractSQLRowValuePlain(value)))
  );
  const explain = lines.map((line) => line[0]).join("\n");
  if (!explain) return undefined;
  return createExplainToken({ statement, explain, engine });
}

async function getExplainTokenForMSSQL(
  database: Database,
  params: SQLEditorQueryParams,
  runQuery: ReturnType<typeof useExecuteSQL>["runQuery"]
): Promise<string | undefined> {
  const context: SQLEditorDatabaseQueryContext = {
    id: uuidv4(),
    params: {
      ...params,
      queryOption: create(QueryOptionSchema, {
        mssqlExplainFormat:
          QueryOption_MSSQLExplainFormat.MSSQL_EXPLAIN_FORMAT_XML,
      }),
    },
    status: "PENDING",
  };
  await runQuery(database, context);
  const result = context.resultSet?.results[0];
  if (!result) return undefined;
  return getExplainTokenFromResult(result, Engine.MSSQL);
}
