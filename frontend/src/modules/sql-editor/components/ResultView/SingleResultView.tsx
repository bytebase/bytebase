import { create } from "@bufbuild/protobuf";
import { isEmpty } from "lodash-es";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  BracesIcon,
  CopyIcon,
  InfoIcon,
  type LucideIcon,
  NetworkIcon,
  Rows3Icon,
  Table2Icon,
  XIcon,
} from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { AdvancedSearch } from "@/components/AdvancedSearch";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import {
  DataExportButton,
  type DataExportRequest,
} from "@/components/DataExportButton";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  SegmentedControl,
  type SegmentedControlOption,
} from "@/components/ui/segmented-control";
import { Switch } from "@/components/ui/switch";
import { Tooltip } from "@/components/ui/tooltip";
import { useExecuteSQL } from "@/hooks/useExecuteSQL";
import { writeTextToClipboard } from "@/lib/clipboard";
import { cn } from "@/lib/utils";
import { useSQLEditorQueryDataPolicy } from "@/modules/sql-editor/hooks/useSQLEditorState";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { useSQLEditorTabState } from "@/modules/sql-editor/store/tab";
import type {
  ResultLayout,
  SQLEditorDatabaseQueryContext,
  SQLEditorQueryParams,
} from "@/types";
import { Engine, ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryOption_ExplainFormat,
  QueryOptionSchema,
  type QueryResult,
} from "@/types/proto-es/v1/sql_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import {
  isVisualizerEngine,
  VISUALIZER_EXPLAIN_FORMATS,
} from "@/utils/queryPlanEngines";
import {
  flattenElasticsearchSearchResult,
  flattenNoSQLQueryResult,
  formatNoSQLQueryResultAsJSON,
  isNoSQLQueryResult,
} from "@/utils/sqlResult";
import { STORAGE_KEY_SQL_EDITOR_NOSQL_TABLE_VIEW } from "@/utils/storage-keys";
import { getDatabaseProject, getInstanceResource } from "@/utils/v1/database";
import { instanceV1AllowsExplain } from "@/utils/v1/instance";
import { compareQueryRowValues, extractSQLRowValuePlain } from "@/utils/v1/sql";
import { CopyAllButton } from "./CopyAllButton";
import { SQLResultViewProvider } from "./context";
import { DataExplorerResultView } from "./DataExplorerResultView";
import { DetailPanel } from "./DetailPanel";
import { DocumentJSONView } from "./DocumentJSONView";
import { EmptyView } from "./EmptyView";
import { ErrorView } from "./ErrorView";
import { canReplayStatement, planAvailability } from "./planAvailability";
import {
  type InlineQueryPlan,
  QueryPlanResultView,
} from "./QueryPlanResultView";
import { formatQueryTime, ResultStatusBar } from "./ResultStatusBar";
import { SelectionCopyTooltips } from "./SelectionCopyTooltips";
import { TextSearchControl } from "./TextSearchControl";
import type {
  ResultTableColumn,
  ResultTableRow,
  ResultViewPresentation,
  SortState,
} from "./types";
import { useResultTableSearch } from "./useResultTableSearch";
import {
  VirtualDataBlock,
  type VirtualDataBlockHandle,
} from "./VirtualDataBlock";
import {
  VirtualDataTable,
  type VirtualDataTableHandle,
} from "./VirtualDataTable";

export interface SingleResultViewProps {
  disallowCopyingData: boolean;
  params: SQLEditorQueryParams;
  database: Database;
  result: QueryResult;
  // Every result of the run, and which one this view shows. Visualize uses
  // earlier results to decide whether replaying this statement is safe.
  results?: QueryResult[];
  resultIndex?: number;
  showExport: boolean;
  // Optional tooltip shown on the export button — used to explain when the
  // export is enabled by a JIT access grant despite the policy disabling it.
  exportTooltip?: ReactNode;
  maximumExportCount?: number;
  onExport?: (req: DataExportRequest & { statement: string }) => void;
  // Rendered in the toolbar when `showExport` is false — e.g. a
  // "Request export" affordance when the policy disables direct export.
  requestExportSlot?: ReactNode;
  // Compact layout (fixed-height, non-growing body) used by the terminal /
  // admin result panel. Defaults to the flex-grow saved query layout.
  compact?: boolean;
  presentation?: ResultViewPresentation;
}

type ViewMode = "RESULT" | "EMPTY" | "AFFECTED-ROWS" | "ERROR";
type DocumentViewMode = "TABLE" | "JSON";

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
  const { disallowCopyingData, database, result } = props;
  const engine = getInstanceResource(database).engine;
  const [noSQLTableView, setNoSQLTableView] = useLocalStorageBoolean(
    STORAGE_KEY_SQL_EDITOR_NOSQL_TABLE_VIEW,
    true
  );
  const [documentViewMode, setDocumentViewMode] =
    useState<DocumentViewMode>("JSON");
  // Sort state lives at this level (not inside the inner component) so the
  // sorted `rows` we feed into the provider are the same array the user
  // sees in the table. The provider's selection-copy logic dereferences
  // `rows[selectionIndex]` — if the provider held the unsorted array but
  // the table rendered sorted rows, Cmd+C would copy the wrong row.
  const [sortState, setSortState] = useState<SortState | undefined>();

  const isDocumentEngine =
    engine === Engine.COSMOSDB || engine === Engine.MONGODB;
  const supportsDocumentJSONView = useMemo(
    () => isDocumentEngine && isNoSQLQueryResult(result),
    [isDocumentEngine, result]
  );
  const activeDocumentViewMode =
    props.presentation === "DATA_EXPLORER"
      ? "TABLE"
      : supportsDocumentJSONView
        ? documentViewMode
        : "TABLE";
  const flattened = useMemo(() => {
    if (engine === Engine.ELASTICSEARCH) {
      return flattenElasticsearchSearchResult(result);
    }
    if (isDocumentEngine && activeDocumentViewMode === "TABLE") {
      return flattenNoSQLQueryResult(result);
    }
    return undefined;
  }, [activeDocumentViewMode, engine, isDocumentEngine, result]);
  const supportsTableViewToggle =
    engine === Engine.ELASTICSEARCH && flattened !== undefined;
  const activeResult = isDocumentEngine
    ? (flattened ?? result)
    : supportsTableViewToggle && noSQLTableView
      ? flattened
      : result;
  const flattenedTableView =
    flattened !== undefined && activeResult === flattened;

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

  // Reset sort whenever the user toggles a NoSQL view:
  // column indices don't carry over between flattened/raw shapes.
  useEffect(() => {
    setSortState(undefined);
  }, [activeDocumentViewMode, noSQLTableView]);

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
      disallowCopyingData={disallowCopyingData}
      engine={engine}
      schema={props.params.connection.schema}
      rows={rows}
      columns={columns}
    >
      {props.presentation === "DATA_EXPLORER" ? (
        <DataExplorerResultView
          rows={rows}
          columns={columns}
          database={database}
          result={result}
          sortState={sortState}
          onToggleSort={toggleSort}
        />
      ) : (
        <SingleResultViewInner
          {...props}
          engine={engine}
          columns={columns}
          rows={rows}
          sortState={sortState}
          toggleSort={toggleSort}
          flattenedTableView={flattenedTableView}
          supportsTableViewToggle={supportsTableViewToggle}
          noSQLTableView={noSQLTableView}
          setNoSQLTableView={setNoSQLTableView}
          supportsDocumentJSONView={supportsDocumentJSONView}
          documentViewMode={activeDocumentViewMode}
          setDocumentViewMode={setDocumentViewMode}
        />
      )}
    </SQLResultViewProvider>
  );
}

interface SingleResultViewInnerProps extends SingleResultViewProps {
  engine: Engine;
  columns: ResultTableColumn[];
  rows: ResultTableRow[];
  sortState: SortState | undefined;
  toggleSort: (columnIndex: number) => void;
  flattenedTableView: boolean;
  supportsTableViewToggle: boolean;
  noSQLTableView: boolean;
  setNoSQLTableView: (next: boolean) => void;
  supportsDocumentJSONView: boolean;
  documentViewMode: DocumentViewMode;
  setDocumentViewMode: (mode: DocumentViewMode) => void;
}

function SingleResultViewInner({
  disallowCopyingData,
  params,
  database,
  result,
  results = [],
  resultIndex = 0,
  showExport,
  exportTooltip,
  maximumExportCount,
  onExport,
  requestExportSlot,
  engine,
  columns,
  rows,
  sortState,
  toggleSort,
  flattenedTableView,
  supportsTableViewToggle,
  noSQLTableView,
  setNoSQLTableView,
  supportsDocumentJSONView,
  documentViewMode,
  setDocumentViewMode,
  compact = false,
}: SingleResultViewInnerProps) {
  const { t } = useTranslation();
  const project = useSQLEditorEditorState((s) => s.project);
  const queryDataPolicy = useSQLEditorQueryDataPolicy(project);
  const currentTabMode = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.mode
  );
  const resultRowsLimit = useSQLEditorEditorState((s) => s.resultRowsLimit);
  const policyMaxRows = queryDataPolicy.maximumResultRows;
  const { runQuery } = useExecuteSQL();

  const supportFormats = useMemo(
    () => [
      ExportFormat.CSV,
      ExportFormat.JSON,
      ExportFormat.SQL,
      ExportFormat.XLSX,
    ],
    []
  );

  const dataTableRef = useRef<
    VirtualDataTableHandle | VirtualDataBlockHandle | null
  >(null);

  // The view switch's choice is remembered per editor tab, so the next run in
  // this tab opens the same way.
  const rememberedLayout = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.resultLayout
  );
  const updateCurrentTab = useSQLEditorTabState((s) => s.updateCurrentTab);
  const [chosenLayout, setChosenLayout] = useState<ResultLayout>(
    rememberedLayout ?? "TABLE"
  );
  const setLayout = useCallback(
    (next: ResultLayout) => {
      setChosenLayout(next);
      updateCurrentTab({ resultLayout: next });
    },
    [updateCurrentTab]
  );
  const planOption = planAvailability({
    result,
    results,
    resultIndex,
    engineExplains: instanceV1AllowsExplain(engine),
    canExplain: hasProjectPermissionV2(
      getDatabaseProject(database),
      "bb.sql.explain"
    ),
  });
  // A remembered Plan choice falls back to the table where this result offers
  // no plan.
  const layout: ResultLayout =
    chosenLayout === "PLAN" && planOption !== "offered"
      ? "TABLE"
      : chosenLayout;
  const vertical = layout === "VERTICAL";
  // The plan fetched for this result, kept so switching views does not fetch
  // it again.
  const [planCache, setPlanCache] = useState<{
    result: QueryResult;
    plan: InlineQueryPlan;
  }>();
  const resultPlan = planCache?.result === result ? planCache.plan : undefined;
  const loadResultPlan = useCallback(
    async (signal: AbortSignal) => {
      const loaded = await explainStatement(
        database,
        params,
        result.statement,
        runQuery,
        engine,
        signal
      );
      if (loaded) setPlanCache({ result, plan: loaded });
      return loaded;
    },
    [database, engine, params, result, runQuery]
  );
  // Document and search engines keep their own view controls; the SQL
  // engines get the one switch with Table, Vertical and Plan.
  const sqlLayoutSwitch = !supportsDocumentJSONView && !supportsTableViewToggle;
  const layoutOptions = useMemo(() => {
    const options: SegmentedControlOption<ResultLayout>[] = [
      {
        value: "TABLE",
        label: (
          <SegmentIcon icon={Table2Icon} label={t("sql-editor.table-view")} />
        ),
        tooltip: t("sql-editor.table-view"),
      },
      {
        value: "VERTICAL",
        label: (
          <SegmentIcon
            icon={Rows3Icon}
            label={t("sql-editor.vertical-display")}
          />
        ),
        tooltip: t("sql-editor.vertical-display"),
      },
    ];
    if (planOption !== "absent") {
      const needsPermission = planOption === "needs-permission";
      options.push({
        value: "PLAN",
        label: (
          <SegmentIcon icon={NetworkIcon} label={t("sql-editor.plan-view")} />
        ),
        tooltip: needsPermission
          ? t("sql-editor.query-plan-requires-explain")
          : t("sql-editor.plan-view"),
        disabled: needsPermission,
      });
    }
    return options;
  }, [planOption, t]);

  const {
    params: searchParams,
    setParams: setSearchParams,
    scopeOptions: searchScopeOptions,
    candidateActiveIndex: searchCandidateActiveIndex,
    candidateRowIndexes: searchCandidateRowIndexes,
    activeRowIndex,
    next: scrollToNextCandidate,
    previous: scrollToPreviousCandidate,
    clear: clearSearchCandidate,
  } = useResultTableSearch(rows, columns, noSQLTableView);
  const [documentSearchQuery, setDocumentSearchQuery] = useState("");
  const [documentSearchActiveIndex, setDocumentSearchActiveIndex] = useState(0);
  const [documentSearchMatchCount, setDocumentSearchMatchCount] = useState(0);

  useEffect(() => {
    setDocumentSearchQuery("");
    setDocumentSearchActiveIndex(0);
    setDocumentSearchMatchCount(0);
  }, [result]);

  useEffect(() => {
    setDocumentSearchActiveIndex(0);
  }, [documentSearchQuery]);

  useEffect(() => {
    setDocumentSearchActiveIndex((current) => {
      if (documentSearchMatchCount === 0) {
        return 0;
      }
      return Math.min(current, documentSearchMatchCount - 1);
    });
  }, [documentSearchMatchCount]);

  const viewMode: ViewMode = useMemo(() => {
    if (result.error && rows.length === 0) return "ERROR";
    const columnNames = result.columnNames;
    if (columnNames?.length === 0) return "EMPTY";
    if (columnNames?.length === 1 && columnNames[0] === "Affected Rows") {
      return "AFFECTED-ROWS";
    }
    return "RESULT";
  }, [result, rows.length]);

  const scrollToRow = useCallback((index: number | undefined) => {
    if (index === undefined || index < 0) return;
    requestAnimationFrame(() => dataTableRef.current?.scrollTo(index));
  }, []);

  // Re-scroll when the active candidate or vertical mode changes.
  useEffect(() => {
    scrollToRow(activeRowIndex);
  }, [activeRowIndex, vertical, scrollToRow]);

  const reachQueryLimit =
    currentTabMode !== "ADMIN" &&
    (rows.length === resultRowsLimit || rows.length === policyMaxRows);

  const getMaskingReason = useCallback(
    (columnIndex: number) => {
      if (flattenedTableView) return undefined;
      if (!result.masked || columnIndex >= result.masked.length) {
        return undefined;
      }
      const reason = result.masked[columnIndex];
      if (!reason || !reason.semanticTypeId) return undefined;
      return reason;
    },
    [flattenedTableView, result.masked]
  );

  const plan = result.queryPlan;
  const planInRows =
    isVisualizerEngine(engine) &&
    plan?.format ===
      QueryOption_ExplainFormat[VISUALIZER_EXPLAIN_FORMATS[engine]];
  const planInResult = useMemo(() => getInlineQueryPlan(result), [result]);
  const canReplay =
    isVisualizerEngine(engine) &&
    plan?.format === QueryOption_ExplainFormat.TEXT &&
    !plan.executed &&
    canReplayStatement(results, resultIndex);
  // Multi-column plans (SQL Server SHOWPLAN_ALL) keep their table unless the
  // XML plan can be loaded for the visualizer.
  const showInlinePlan =
    !!plan && (result.columnNames.length === 1 || canReplay);
  const loadPlan = useCallback(
    (signal: AbortSignal) =>
      isVisualizerEngine(engine)
        ? explainStatement(
            database,
            params,
            result.statement,
            runQuery,
            engine,
            signal
          )
        : Promise.resolve(undefined),
    [database, engine, params, result.statement, runQuery]
  );

  const queryTime = formatQueryTime(result.latency);

  const resultRowsText = `${rows.length} ${t("sql-editor.rows.self")}`;
  const isJSONView = supportsDocumentJSONView && documentViewMode === "JSON";
  const moveDocumentSearchMatch = (offset: number) => {
    if (documentSearchMatchCount === 0) {
      return;
    }
    setDocumentSearchActiveIndex(
      (current) =>
        (current + offset + documentSearchMatchCount) % documentSearchMatchCount
    );
  };

  const handleExport = (req: DataExportRequest) => {
    // Forward the user-typed query (`params.statement`) rather than
    // `result.statement`. The backend may rewrite the result statement
    // with an auto-appended LIMIT for non-admin reads — re-running that
    // rewritten SQL on the export path silently caps the exported rows
    // even when the user asks for more.
    onExport?.({ ...req, statement: params.statement });
  };

  const handleCopyJSON = () => {
    if (disallowCopyingData) {
      return;
    }
    const content = formatNoSQLQueryResultAsJSON(result);
    if (content) {
      void writeTextToClipboard(content);
    }
  };

  // ---- Render branches by viewMode ----

  if (viewMode === "AFFECTED-ROWS") {
    return (
      <div
        className={cn(
          "text-md font-normal flex items-center gap-x-1",
          "text-control-light"
        )}
      >
        <span>{String(extractSQLRowValuePlain(result.rows[0].values[0]))}</span>
        <span>{t("sql-editor.rows-affected")}</span>
      </div>
    );
  }

  if (viewMode === "EMPTY") {
    return <EmptyView />;
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

      {viewMode === "RESULT" && showInlinePlan && (
        <>
          <div
            className={cn(
              "flex flex-col",
              compact ? "h-80 overflow-hidden" : "flex-1 min-h-0"
            )}
          >
            <QueryPlanResultView
              key={`${database.name}\n${engine}\n${result.statement}`}
              rawPlan={planInResult?.source ?? ""}
              initialPlan={planInRows ? planInResult : undefined}
              engine={isVisualizerEngine(engine) ? engine : undefined}
              loadPlan={canReplay ? loadPlan : undefined}
              disallowCopyingData={disallowCopyingData}
            />
          </div>
          <ResultStatusBar
            database={database}
            statement={result.statement ?? ""}
            queryTime={queryTime}
          />
        </>
      )}

      {viewMode === "RESULT" && !showInlinePlan && (
        <>
          {result.error && (
            <Alert variant="error" className="w-full mb-2">
              <ErrorView error={result.error} />
            </Alert>
          )}

          {/* Toolbar */}
          <div className="result-toolbar relative w-full shrink-0 flex flex-row gap-x-4 justify-between items-center mb-2 hide-scrollbar">
            <div className="flex flex-row justify-start items-center gap-x-2 mr-2 flex-1">
              {isJSONView ? (
                <TextSearchControl
                  query={documentSearchQuery}
                  activeMatchIndex={documentSearchActiveIndex}
                  matchCount={documentSearchMatchCount}
                  onQueryChange={setDocumentSearchQuery}
                  onMove={moveDocumentSearchMatch}
                />
              ) : (
                <AdvancedSearch
                  params={searchParams}
                  scopeOptions={searchScopeOptions}
                  placeholder=""
                  onParamsChange={setSearchParams}
                  onEnter={scrollToNextCandidate}
                />
              )}
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
              {supportsDocumentJSONView ? (
                <SegmentedControl<DocumentViewMode>
                  value={documentViewMode}
                  options={[
                    {
                      value: "TABLE",
                      label: (
                        <SegmentIcon
                          icon={Table2Icon}
                          label={t("sql-editor.table-view")}
                        />
                      ),
                      tooltip: t("sql-editor.table-view"),
                    },
                    {
                      value: "JSON",
                      label: (
                        <SegmentIcon
                          icon={BracesIcon}
                          label={t("sql-editor.json-view")}
                        />
                      ),
                      tooltip: t("sql-editor.json-view"),
                    },
                  ]}
                  onValueChange={setDocumentViewMode}
                  ariaLabel={t("sql-editor.result-view-mode")}
                  appearance="soft"
                  className="flex-nowrap"
                  size="xs"
                />
              ) : supportsTableViewToggle ? (
                <div className="flex items-center gap-x-1">
                  <Switch
                    checked={noSQLTableView}
                    onCheckedChange={setNoSQLTableView}
                  />
                  <span className="whitespace-nowrap text-sm text-control-light">
                    {t("sql-editor.table-view")}
                  </span>
                </div>
              ) : null}
              {sqlLayoutSwitch ? (
                <SegmentedControl<ResultLayout>
                  value={layout}
                  options={layoutOptions}
                  onValueChange={setLayout}
                  ariaLabel={t("sql-editor.result-view-mode")}
                  appearance="soft"
                  className="flex-nowrap"
                  size="xs"
                />
              ) : (
                !isJSONView && (
                  <div className="flex items-center gap-x-1">
                    <Switch
                      checked={vertical}
                      onCheckedChange={(checked) =>
                        setLayout(checked ? "VERTICAL" : "TABLE")
                      }
                    />
                    <span className="whitespace-nowrap text-sm text-control-light">
                      {t("sql-editor.vertical-display")}
                    </span>
                  </div>
                )
              )}
              {!disallowCopyingData && rows.length > 0 && isJSONView && (
                <Button
                  size="sm"
                  appearance="outline"
                  className="px-2 text-control border-control-border hover:bg-control-bg-hover"
                  onClick={handleCopyJSON}
                >
                  <CopyIcon className="size-4" />
                  {t("common.copy-all")}
                </Button>
              )}
              {!disallowCopyingData && rows.length > 0 && !isJSONView && (
                <CopyAllButton />
              )}
              {showExport ? (
                <DataExportButton
                  size="sm"
                  disabled={!result || isEmpty(result)}
                  supportFormats={supportFormats}
                  viewMode="DRAWER"
                  supportPassword
                  tooltip={exportTooltip}
                  maximumExportCount={maximumExportCount}
                  formContent={<DatabaseInfo database={database} />}
                  onExport={handleExport}
                />
              ) : (
                requestExportSlot
              )}
            </div>
            {!isJSONView && <SelectionCopyTooltips />}
          </div>

          {/* Body */}
          {isJSONView ? (
            <DocumentJSONView
              result={result}
              disallowCopyingData={disallowCopyingData}
              compact={compact}
              searchQuery={documentSearchQuery}
              activeMatchIndex={documentSearchActiveIndex}
              onMatchCountChange={setDocumentSearchMatchCount}
            />
          ) : layout === "PLAN" ? (
            <div
              className={cn(
                "w-full flex flex-col",
                compact ? "h-80 overflow-hidden" : "flex-1 min-h-0"
              )}
            >
              <QueryPlanResultView
                key={planViewKey(result)}
                rawPlan=""
                initialPlan={resultPlan}
                engine={isVisualizerEngine(engine) ? engine : undefined}
                loadPlan={loadResultPlan}
                estimated
                onCancel={() => setLayout("TABLE")}
                disallowCopyingData={disallowCopyingData}
              />
            </div>
          ) : (
            <div
              className={cn(
                "w-full flex flex-col relative",
                compact
                  ? "h-80 overflow-hidden"
                  : "flex-1 min-h-0 overflow-y-auto"
              )}
            >
              {vertical ? (
                <VirtualDataBlock
                  ref={dataTableRef as React.RefObject<VirtualDataBlockHandle>}
                  rows={rows}
                  columns={columns}
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
                {searchCandidateRowIndexes.length > 0 && (
                  <div className="flex flex-row gap-x-2 border shadow rounded-sm bg-background py-1 px-2">
                    <Button
                      size="sm"
                      appearance="secondary"
                      disabled={searchCandidateActiveIndex <= 0}
                      onClick={scrollToPreviousCandidate}
                    >
                      <ArrowUpIcon className="size-4" />
                      {t("sql-editor.previous-row")}
                    </Button>
                    <Button
                      size="sm"
                      appearance="secondary"
                      disabled={
                        searchCandidateActiveIndex >=
                        searchCandidateRowIndexes.length - 1
                      }
                      onClick={scrollToNextCandidate}
                    >
                      <ArrowDownIcon className="size-4" />
                      {t("sql-editor.next-row")}
                    </Button>
                    <Button
                      size="sm"
                      appearance="secondary"
                      className="w-7 p-0"
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
                        size="md"
                        appearance="secondary"
                        className="w-9 p-0 rounded-full"
                        onClick={() => scrollToRow(0)}
                      >
                        <ArrowUpIcon className="size-4" />
                      </Button>
                    </div>
                  </Tooltip>
                  <Tooltip content={t("sql-editor.scroll-to-bottom")}>
                    <div className="rounded-full shadow bg-background">
                      <Button
                        size="md"
                        appearance="secondary"
                        className="w-9 p-0 rounded-full"
                        onClick={() => scrollToRow(rows.length - 1)}
                      >
                        <ArrowDownIcon className="size-4" />
                      </Button>
                    </div>
                  </Tooltip>
                </div>
              </div>
            </div>
          )}

          {/* Status bar */}
          <ResultStatusBar
            database={database}
            statement={result.statement ?? ""}
            queryTime={queryTime}
          />
        </>
      )}

      {!isJSONView && !showInlinePlan && (
        <DetailPanel
          rows={rows}
          columns={columns}
          database={database}
          result={result}
          statement={result.statement}
          getMaskingReason={getMaskingReason}
        />
      )}
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

function SegmentIcon({
  icon: Icon,
  label,
}: {
  icon: LucideIcon;
  label: string;
}) {
  return (
    <span className="flex w-7 items-center justify-center py-1">
      <Icon className="size-4" aria-hidden />
      <span className="sr-only">{label}</span>
    </span>
  );
}

// A plan view is keyed by the result it belongs to, so a re-run of the same
// statement gets a fresh view and a fresh plan.
const planViewKeys = new WeakMap<QueryResult, string>();
function planViewKey(result: QueryResult): string {
  let key = planViewKeys.get(result);
  if (!key) {
    key = uuidv4();
    planViewKeys.set(result, key);
  }
  return key;
}

function DatabaseInfo({ database }: { database: Database }) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-1 mb-3">
      <span className="text-xs text-control-light">{t("common.database")}</span>
      <DatabaseTargetDisplay database={database} showEnvironment />
    </div>
  );
}

function getInlineQueryPlan(result: QueryResult): InlineQueryPlan | undefined {
  const lines = result.rows.map((row) =>
    row.values.map((value) => String(extractSQLRowValuePlain(value)))
  );
  // Multi-column plans keep every column, tab-separated under a header.
  const source =
    result.columnNames.length > 1
      ? [result.columnNames, ...lines].map((line) => line.join("\t")).join("\n")
      : lines.map((line) => line[0]).join("\n");
  if (!source) return undefined;
  return { source };
}

// explainStatement asks the engine for the plan of a statement as it ran, in
// the format the visualizer draws where there is one and the engine's default
// otherwise. The signal cancels the request.
async function explainStatement(
  database: Database,
  params: SQLEditorQueryParams,
  statement: string,
  runQuery: ReturnType<typeof useExecuteSQL>["runQuery"],
  engine: Engine,
  signal?: AbortSignal
): Promise<InlineQueryPlan | undefined> {
  if (!statement) return undefined;
  const explainFormat = isVisualizerEngine(engine)
    ? QueryOption_ExplainFormat[VISUALIZER_EXPLAIN_FORMATS[engine]]
    : QueryOption_ExplainFormat.EXPLAIN_FORMAT_UNSPECIFIED;
  const context: SQLEditorDatabaseQueryContext = {
    id: uuidv4(),
    params: {
      ...params,
      statement,
      explain: true,
      queryOption: create(QueryOptionSchema, { explainFormat }),
    },
    status: "PENDING",
  };
  const run = runQuery(database, context);
  signal?.addEventListener("abort", () => context.abortController?.abort(), {
    once: true,
  });
  await run;
  const resultSet = context.resultSet;
  if (resultSet?.error) throw new Error(resultSet.error);
  const result = resultSet?.results[0];
  if (!result) return undefined;
  if (result.error) throw new Error(result.error);
  return getInlineQueryPlan(result);
}
