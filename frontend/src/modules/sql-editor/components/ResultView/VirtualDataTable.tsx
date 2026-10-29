import { head, last } from "lodash-es";
import { ExpandIcon } from "lucide-react";
import { forwardRef, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { MaskingReasonPopover } from "@/modules/sql-editor/components/MaskingReasonPopover";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { MaskingReason } from "@/types/proto-es/v1/sql_service_pb";
import type { SearchParams } from "@/utils/v1/advanced-search/common";
import { BinaryFormatButton } from "./BinaryFormatButton";
import {
  type BinaryFormat,
  getBinaryFormatByColumnType,
} from "./binary-format";
import { ColumnSortedIcon } from "./ColumnSortedIcon";
import { getPlainValue } from "./cell-value";
import {
  useBinaryFormatContext,
  useSelectionContext,
  useSQLResultViewContext,
} from "./context";
import {
  QueryResultTableGrid,
  type QueryResultTableGridHandle,
} from "./QueryResultTableGrid";
import { ResultCopyContextMenu } from "./ResultCopyContextMenu";
import { TableCell } from "./TableCell";
import type {
  ResultTableColumn,
  ResultTableRow,
  SortDirection,
  SortState,
} from "./types";

export type VirtualDataTableHandle = QueryResultTableGridHandle;

export interface VirtualDataTableProps {
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  activeRowIndex: number;
  getMaskingReason?: (index: number) => MaskingReason | undefined;
  database: Database;
  statement?: string;
  sortState?: SortState;
  search: SearchParams;
  onToggleSort: (columnIndex: number) => void;
  onRowClick?: (rowIndex: number) => void;
  showRowDetailAction?: boolean;
  showCellDetailAction?: boolean;
  allowSelection?: boolean;
  activeRowHighlight?: "subtle" | "strong";
}

export const VirtualDataTable = forwardRef<
  VirtualDataTableHandle,
  VirtualDataTableProps
>(function VirtualDataTable(
  {
    rows,
    columns,
    activeRowIndex,
    getMaskingReason,
    database,
    statement,
    sortState,
    search,
    onToggleSort,
    onRowClick,
    showRowDetailAction = true,
    showCellDetailAction = true,
    allowSelection = true,
    activeRowHighlight = "subtle",
  },
  ref
) {
  const { t } = useTranslation();
  const {
    state: selectionState,
    disabled: selectionDisabled,
    toggleSelectColumn,
    toggleSelectRow,
    deselect,
  } = useSelectionContext();
  const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();
  const { setDetail } = useSQLResultViewContext();
  const selectionEnabled = allowSelection && !selectionDisabled;

  const getColumnTypeByIndex = useCallback(
    (columnIndex: number) => columns[columnIndex]?.columnType ?? "",
    [columns]
  );
  const getRowCellContent = useCallback(
    (columnIndex: number): string | undefined => {
      for (let i = 0; i < Math.min(3, rows.length); i++) {
        const cell = rows[i]?.item.values[columnIndex];
        if (!cell) continue;
        const plain = getPlainValue(
          cell,
          getColumnTypeByIndex(columnIndex),
          "DEFAULT"
        );
        if (plain) return plain;
      }
      return undefined;
    },
    [rows, getColumnTypeByIndex]
  );

  const getColumnSortDirection = (columnIndex: number): SortDirection => {
    if (!sortState || sortState.columnIndex !== columnIndex) return false;
    return sortState.direction;
  };

  const hasRowSelect =
    selectionState.columns.length === 0 && selectionState.rows.length > 0;
  const hasColumnSelect =
    selectionState.columns.length > 0 && selectionState.rows.length === 0;

  const handleSelectRow = (event: React.MouseEvent, index: number) => {
    event.stopPropagation();
    if ((event.metaKey || event.ctrlKey) && hasRowSelect) {
      const firstRowIndex = head(selectionState.rows);
      const lastRowIndex = last(selectionState.rows);
      if (lastRowIndex !== undefined && index > lastRowIndex) {
        for (let i = lastRowIndex + 1; i <= index; i++) toggleSelectRow(i);
      } else if (firstRowIndex !== undefined && index < firstRowIndex) {
        deselect();
        for (let i = index; i <= firstRowIndex; i++) toggleSelectRow(i);
      }
      return;
    }
    toggleSelectRow(index);
  };

  const handleSelectColumn = (event: React.MouseEvent, index: number) => {
    event.stopPropagation();
    if ((event.metaKey || event.ctrlKey) && hasColumnSelect) {
      const firstCol = head(selectionState.columns);
      const lastCol = last(selectionState.columns);
      if (lastCol !== undefined && index > lastCol) {
        for (let i = lastCol + 1; i <= index; i++) toggleSelectColumn(i);
      } else if (firstCol !== undefined && index < firstCol) {
        deselect();
        for (let i = index; i <= firstCol; i++) toggleSelectColumn(i);
      }
      return;
    }
    return toggleSelectColumn(index);
  };

  const binaryColumnSet = useMemo(() => {
    const candidates: number[] = [];
    for (let i = 0; i < columns.length; i++) {
      if (getBinaryFormatByColumnType(columns[i]?.columnType)) {
        candidates.push(i);
      }
    }
    if (candidates.length === 0) return new Set<number>();
    const out = new Set<number>();
    const remaining = new Set(candidates);
    for (const row of rows) {
      if (remaining.size === 0) break;
      for (const colIdx of [...remaining]) {
        if (row.item.values[colIdx]?.kind?.case === "bytesValue") {
          out.add(colIdx);
          remaining.delete(colIdx);
        }
      }
    }
    return out;
  }, [columns, rows]);

  const activeClassName = (rowIndex: number, forIndex: boolean) => {
    if (activeRowIndex !== rowIndex) return undefined;
    if (activeRowHighlight === "strong") {
      return forIndex
        ? "bg-accent/20! text-accent font-medium"
        : "bg-accent/20!";
    }
    return "bg-accent/10!";
  };

  return (
    <ResultCopyContextMenu>
      <QueryResultTableGrid
        ref={ref}
        rows={rows}
        columns={columns}
        getRowCellContent={getRowCellContent}
        indexHeaderClassName={selectionEnabled ? "pl-4" : "pl-1"}
        headerClassName={(columnIndex) =>
          cn(
            selectionEnabled && "cursor-pointer hover:bg-control-bg-hover",
            selectionState.rows.length === 0 &&
              selectionState.columns.includes(columnIndex) &&
              "bg-accent/10!"
          )
        }
        onHeaderClick={selectionEnabled ? handleSelectColumn : undefined}
        renderHeader={(column, columnIndex) => {
          const reason = getMaskingReason?.(columnIndex);
          return (
            <>
              <span className="min-w-0 truncate select-none">
                {String(column.name).length > 0 ? (
                  column.name
                ) : (
                  <br className="min-h-4 inline-flex" />
                )}
              </span>
              {reason && (
                <span className="shrink-0">
                  <MaskingReasonPopover
                    reason={reason}
                    statement={statement}
                    database={database.name}
                  />
                </span>
              )}
              <span
                onClick={(event) => {
                  event.stopPropagation();
                  onToggleSort(columnIndex);
                }}
                role="button"
                tabIndex={-1}
                className="shrink-0 inline-flex"
              >
                <ColumnSortedIcon
                  isSorted={getColumnSortDirection(columnIndex)}
                />
              </span>
              {binaryColumnSet.has(columnIndex) && (
                <span
                  onClick={(event) => event.stopPropagation()}
                  className="shrink-0"
                >
                  <BinaryFormatButton
                    format={getBinaryFormat({ colIndex: columnIndex })}
                    onFormatChange={(format: BinaryFormat) =>
                      setBinaryFormat({ colIndex: columnIndex, format })
                    }
                  />
                </span>
              )}
            </>
          );
        }}
        rowIndexClassName={(rowIndex) =>
          cn(
            activeClassName(rowIndex, true),
            selectionState.columns.length === 0 &&
              selectionState.rows.includes(rowIndex) &&
              "bg-accent/20!"
          )
        }
        renderRowIndex={(rowIndex) => (
          <>
            {showRowDetailAction && (
              <div className="absolute left-3 top-1/2 size-7 -translate-y-1/2 opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto group-focus-within:opacity-100 group-focus-within:pointer-events-auto">
                <Tooltip content={t("sql-editor.view-detail")} side="right">
                  <Button
                    size="xs"
                    appearance="outline"
                    className="rounded-full shadow"
                    aria-label={t("sql-editor.view-detail")}
                    onClick={(event) => {
                      event.stopPropagation();
                      if (event.detail > 0) event.currentTarget.blur();
                      setDetail({ row: rowIndex, col: 0, view: "row" });
                    }}
                  >
                    <ExpandIcon className="size-3" />
                  </Button>
                </Tooltip>
              </div>
            )}
            <span
              className={cn(
                "textinfolabel pr-1 truncate",
                selectionEnabled ? "pl-4" : "pl-1",
                showRowDetailAction &&
                  "group-hover:opacity-0 group-hover:pointer-events-none group-focus-within:opacity-0 group-focus-within:pointer-events-none"
              )}
            >
              {rowIndex + 1}
            </span>
            {selectionEnabled && (
              <Button
                appearance="secondary"
                size="xs"
                type="button"
                aria-label={`Select row ${rowIndex + 1}`}
                onClick={(event) => handleSelectRow(event, rowIndex)}
                className={cn(
                  "absolute inset-y-0 left-0 w-3 cursor-pointer bg-accent/5 hover:bg-accent/10",
                  selectionState.columns.length === 0 &&
                    selectionState.rows.includes(rowIndex) &&
                    "bg-accent/20!"
                )}
              />
            )}
          </>
        )}
        cellClassName={(rowIndex) => activeClassName(rowIndex, false)}
        renderCell={(rowIndex, columnIndex) => {
          const cell = rows[rowIndex]?.item.values[columnIndex];
          if (!cell) return null;
          return (
            <TableCell
              value={cell}
              keyword={search.query}
              scope={search.scopes.find(
                (scope) => scope.id === columns[columnIndex]?.id
              )}
              rowIndex={rowIndex}
              colIndex={columnIndex}
              allowSelect={selectionEnabled}
              columnType={getColumnTypeByIndex(columnIndex)}
              showDetailAction={showCellDetailAction}
            />
          );
        }}
        onRowClick={onRowClick}
        onRowDoubleClick={(rowIndex) =>
          setDetail({ row: rowIndex, col: 0, view: "row" })
        }
      />
    </ResultCopyContextMenu>
  );
});
