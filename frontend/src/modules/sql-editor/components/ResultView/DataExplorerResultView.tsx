import { ArrowDownIcon, ArrowUpIcon, XIcon } from "lucide-react";
import { useCallback, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import {
  Panel,
  Group as PanelGroup,
  Separator as PanelResizeHandle,
} from "react-resizable-panels";
import { AdvancedSearch } from "@/components/AdvancedSearch";
import { Button } from "@/components/ui/button";
import { resizeHandleClass } from "@/modules/schema-editor/resize";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/modules/sql-editor/store/tab";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";
import { CopyAllButton } from "./CopyAllButton";
import { useSQLResultViewContext } from "./context";
import { DetailPanel } from "./DetailPanel";
import { formatQueryTime, ResultStatusBar } from "./ResultStatusBar";
import type { ResultTableColumn, ResultTableRow, SortState } from "./types";
import { useResultTableSearch } from "./useResultTableSearch";
import {
  VirtualDataTable,
  type VirtualDataTableHandle,
} from "./VirtualDataTable";

interface DataExplorerResultViewProps {
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  database: Database;
  result: QueryResult;
  sortState?: SortState;
  onToggleSort: (columnIndex: number) => void;
}

export function DataExplorerResultView({
  rows,
  columns,
  database,
  result,
  sortState,
  onToggleSort,
}: DataExplorerResultViewProps) {
  const { t } = useTranslation();
  const { detail, disallowCopyingData, setDetail } = useSQLResultViewContext();
  const dataTableRef = useRef<VirtualDataTableHandle>(null);
  const search = useResultTableSearch(rows, columns, result);
  const tabId = useSQLEditorTabState((state) => state.currentTabId);
  const selectedRowKey = useSQLEditorTabState(
    (state) =>
      state.tabsById.get(state.currentTabId)?.dataExplorer?.selectedRowKey
  );

  useEffect(() => {
    if (rows.length === 0) {
      setDetail(undefined);
      return;
    }
    const selectedIndex = rows.findIndex((row) => row.key === selectedRowKey);
    const rowIndex = selectedIndex >= 0 ? selectedIndex : 0;
    setDetail({ row: rowIndex, col: 0, view: "row" });
  }, [rows, selectedRowKey, setDetail]);

  useEffect(() => {
    if (!detail || detail.view !== "row") return;
    const rowKey = rows[detail.row]?.key;
    if (rowKey === undefined || rowKey === selectedRowKey) return;
    const tabsState = getSQLEditorTabsState();
    const tab = tabsState.tabsById.get(tabId);
    if (!tab?.dataExplorer) return;
    tabsState.updateTab(tabId, {
      dataExplorer: { ...tab.dataExplorer, selectedRowKey: rowKey },
    });
  }, [detail, rows, selectedRowKey, tabId]);

  const handleRowClick = useCallback(
    (rowIndex: number) => {
      const rowKey = rows[rowIndex]?.key;
      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabId);
      if (rowKey !== undefined && tab?.dataExplorer) {
        tabsState.updateTab(tabId, {
          dataExplorer: { ...tab.dataExplorer, selectedRowKey: rowKey },
        });
      }
      setDetail({ row: rowIndex, col: 0, view: "row" });
    },
    [rows, setDetail, tabId]
  );

  useEffect(() => {
    if (search.activeRowIndex < 0) return;
    handleRowClick(search.activeRowIndex);
    requestAnimationFrame(() => {
      dataTableRef.current?.scrollTo(search.activeRowIndex);
    });
  }, [handleRowClick, search.activeRowIndex]);

  return (
    <div className="h-full min-h-0 w-full flex flex-col">
      <div className="result-toolbar relative mx-2 mt-2 mb-1 shrink-0 flex items-center justify-between gap-x-4">
        <div className="flex min-w-0 flex-1 items-center gap-x-2">
          <AdvancedSearch
            params={search.params}
            scopeOptions={search.scopeOptions}
            placeholder={t("common.search-results")}
            onParamsChange={search.setParams}
            onEnter={search.next}
          />
          <span className="whitespace-nowrap text-sm text-control-light">
            {rows.length} {t("sql-editor.rows.self")}
          </span>
        </div>
        {!disallowCopyingData && rows.length > 0 && (
          <div className="shrink-0">
            <CopyAllButton />
          </div>
        )}
      </div>
      <div className="flex-1 min-h-0">
        <PanelGroup orientation="horizontal" className="h-full min-h-0 w-full">
          <Panel defaultSize="45%" minSize="30%">
            <div className="relative h-full min-h-0 p-2 pr-1 flex flex-col">
              <VirtualDataTable
                ref={dataTableRef}
                rows={rows}
                columns={columns}
                database={database}
                statement={result.statement}
                activeRowIndex={
                  search.activeRowIndex >= 0
                    ? search.activeRowIndex
                    : detail?.view === "row"
                      ? detail.row
                      : -1
                }
                search={search.params}
                sortState={sortState}
                onToggleSort={onToggleSort}
                onRowClick={handleRowClick}
                showRowDetailAction={false}
                showCellDetailAction={false}
                allowSelection={false}
                activeRowHighlight="strong"
              />
              {search.candidateRowIndexes.length > 0 && (
                <div className="absolute bottom-4 right-3 z-1 flex gap-x-2 border shadow rounded bg-background py-1 px-2">
                  <Button
                    size="sm"
                    appearance="secondary"
                    disabled={search.candidateActiveIndex <= 0}
                    onClick={search.previous}
                  >
                    <ArrowUpIcon className="size-4" />
                    {t("sql-editor.previous-row")}
                  </Button>
                  <Button
                    size="sm"
                    appearance="secondary"
                    disabled={
                      search.candidateActiveIndex >=
                      search.candidateRowIndexes.length - 1
                    }
                    onClick={search.next}
                  >
                    <ArrowDownIcon className="size-4" />
                    {t("sql-editor.next-row")}
                  </Button>
                  <Button
                    size="sm"
                    appearance="secondary"
                    className="size-7 p-0"
                    aria-label={t("common.close")}
                    onClick={search.clear}
                  >
                    <XIcon className="size-4" />
                  </Button>
                </div>
              )}
            </div>
          </Panel>
          <PanelResizeHandle
            className={resizeHandleClass("vertical", "w-0.5")}
          />
          <Panel defaultSize="55%" minSize="30%">
            <DetailPanel
              presentation="embedded"
              rows={rows}
              columns={columns}
              database={database}
              result={result}
              statement={result.statement}
            />
          </Panel>
        </PanelGroup>
      </div>
      <div className="shrink-0 px-2 pb-1">
        <ResultStatusBar
          database={database}
          statement={result.statement ?? ""}
          queryTime={formatQueryTime(result.latency)}
        />
      </div>
    </div>
  );
}
