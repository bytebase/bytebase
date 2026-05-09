import { useVirtualizer } from "@tanstack/react-virtual";
import { head, last } from "lodash-es";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import { MaskingReasonPopover } from "@/react/components/sql-editor/MaskingReasonPopover";
import { cn } from "@/react/lib/utils";
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
import { useBinaryFormatContext, useSelectionContext } from "./context";
import { TableCell } from "./TableCell";
import type {
  ResultTableColumn,
  ResultTableRow,
  SortDirection,
  SortState,
} from "./types";
import { useTableResize } from "./useTableResize";

const ROW_HEIGHT = 35;
const MIN_COL_WIDTH = 64;
const MAX_COL_WIDTH = 640;

export interface VirtualDataTableHandle {
  scrollTo(index: number): void;
}

export interface VirtualDataTableProps {
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  activeRowIndex: number;
  isSensitiveColumn: (index: number) => boolean;
  getMaskingReason?: (index: number) => MaskingReason | undefined;
  database: Database;
  statement?: string;
  sortState?: SortState;
  search: SearchParams;
  onToggleSort: (columnIndex: number) => void;
}

/**
 * Virtualized result-table grid. Replaces the Naive UI `NVirtualList` host
 * with `@tanstack/react-virtual`'s `useVirtualizer`. The header is a
 * separate sticky row outside the virtual scroller so it can stay fixed
 * during vertical scroll without fighting the virtualizer's internal
 * positioning.
 */
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
  },
  ref
) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);

  // Track container width for last-column extra-space distribution.
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const update = () => setContainerWidth(el.clientWidth);
    update();
    const observer = new ResizeObserver(update);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const {
    state: selectionState,
    disabled: selectionDisabled,
    toggleSelectColumn,
    toggleSelectRow,
    deselect,
  } = useSelectionContext();

  const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();

  const getColumnTypeByIndex = useCallback(
    (columnIndex: number) => columns[columnIndex]?.columnType ?? "",
    [columns]
  );

  const getRowCellContent = useCallback(
    (columnIndex: number): string | undefined => {
      // Sample up to 3 rows for column-width estimation.
      for (let i = 0; i < Math.min(3, rows.length); i++) {
        const sampleRow = rows[i];
        if (!sampleRow) continue;
        const cell = sampleRow.item.values[columnIndex];
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

  const tableResize = useTableResize({
    columns,
    rows,
    containerWidth,
    minWidth: MIN_COL_WIDTH,
    maxWidth: MAX_COL_WIDTH,
    getRowCellContent,
  });

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => containerRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 8,
  });

  useImperativeHandle(
    ref,
    () => ({
      scrollTo(index: number) {
        virtualizer.scrollToIndex(Math.max(0, index - 1), {
          align: "start",
          behavior: "smooth",
        });
      },
    }),
    [virtualizer]
  );

  const getColumnSortDirection = (columnIndex: number): SortDirection => {
    if (!sortState || sortState.columnIndex !== columnIndex) return false;
    return sortState.direction;
  };

  const hasRowSelect =
    selectionState.columns.length === 0 && selectionState.rows.length > 0;
  const hasColumnSelect =
    selectionState.columns.length > 0 && selectionState.rows.length === 0;

  const handleSelectRow = (e: React.MouseEvent, index: number) => {
    e.stopPropagation();
    if ((e.metaKey || e.ctrlKey) && hasRowSelect) {
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

  const handleSelectColumn = (e: React.MouseEvent, index: number) => {
    e.stopPropagation();
    if ((e.metaKey || e.ctrlKey) && hasColumnSelect) {
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
    toggleSelectColumn(index);
  };

  // Precompute the set of column indices that hold any binary cell value.
  // Done once per `rows` / `columns` change instead of per-header-render —
  // the per-header form was O(rows × columns) on every VirtualDataTable
  // render and stalled the main thread on large result sets.
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
        const cell = row.item.values[colIdx];
        if (cell?.kind?.case === "bytesValue") {
          out.add(colIdx);
          remaining.delete(colIdx);
        }
      }
    }
    return out;
  }, [columns, rows]);

  // Shift+wheel for mouse + trackpad horizontal swipes.
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const handleWheel = (event: WheelEvent) => {
      const hasHorizontalOverflow =
        container.scrollWidth > container.clientWidth;
      if (!hasHorizontalOverflow) return;
      if (event.shiftKey) {
        event.preventDefault();
        event.stopPropagation();
        container.scrollLeft += event.deltaY || event.deltaX;
        return;
      }
      if (Math.abs(event.deltaX) > Math.abs(event.deltaY)) {
        event.preventDefault();
        event.stopPropagation();
        container.scrollLeft += event.deltaX;
      }
    };
    container.addEventListener("wheel", handleWheel, {
      passive: false,
      capture: true,
    });
    return () => {
      container.removeEventListener("wheel", handleWheel, { capture: true });
    };
  }, []);

  const totalRows = rows.length;
  const indexColWidth = tableResize.getColumnWidth(0);

  // The header row sits outside the virtualizer so it stays sticky during
  // vertical scroll. Same min-width as body rows so horizontal scroll
  // syncs visually.
  const minRowWidth = useMemo(
    () => `${tableResize.effectiveWidth}px`,
    [tableResize.effectiveWidth]
  );

  return (
    <div
      ref={containerRef}
      className="relative w-full flex-1 min-h-0 overflow-auto rounded-sm border"
    >
      {/* Header */}
      <div
        className="sticky top-0 z-1 bg-control-bg flex shadow-sm"
        style={{ minWidth: minRowWidth }}
      >
        {/* Index header */}
        <HeaderCell
          width={indexColWidth}
          onResizeStart={(e) => tableResize.onResizeStart(0, e)}
        >
          <div
            className={cn(
              "textinfolabel pr-1 opacity-0",
              selectionDisabled ? "pl-1" : "pl-4"
            )}
          >
            {totalRows}
          </div>
        </HeaderCell>

        {/* Data column headers */}
        {columns.map((header, columnIndex) => {
          const slotWidth = tableResize.getColumnWidth(columnIndex + 1);
          const isLast = columnIndex === columns.length - 1;
          const isSelected =
            selectionState.rows.length === 0 &&
            selectionState.columns.includes(columnIndex);
          const reason = getMaskingReason?.(columnIndex);
          return (
            <HeaderCell
              key={`${columnIndex}-${header.id}`}
              width={slotWidth}
              borderRight={!isLast}
              onResizeStart={(e) =>
                tableResize.onResizeStart(columnIndex + 1, e)
              }
              onClick={(e) => handleSelectColumn(e, columnIndex)}
              className={cn(
                "px-3 py-1.5 min-w-8 text-left text-xs font-medium text-control-light tracking-wider",
                !selectionDisabled &&
                  "cursor-pointer hover:bg-control-bg-hover",
                isSelected && "bg-accent/10!"
              )}
            >
              <div className="flex items-center min-w-0 gap-x-1">
                <span className="min-w-0 truncate select-none">
                  {String(header.name).length > 0 ? (
                    header.name
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
                  onClick={(e) => {
                    e.stopPropagation();
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
                    onClick={(e) => e.stopPropagation()}
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
              </div>
            </HeaderCell>
          );
        })}
      </div>

      {/* Virtualized body */}
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          minWidth: minRowWidth,
          position: "relative",
        }}
      >
        {virtualizer.getVirtualItems().map((virtualRow) => {
          const rowIndex = virtualRow.index;
          const row = rows[rowIndex];
          if (!row) return null;
          const isActive = activeRowIndex === rowIndex;
          const rowSelected =
            selectionState.columns.length === 0 &&
            selectionState.rows.includes(rowIndex);
          return (
            <div
              key={virtualRow.key}
              data-row-index={rowIndex}
              className="flex absolute inset-x-0 group"
              style={{
                top: 0,
                transform: `translateY(${virtualRow.start}px)`,
                height: `${ROW_HEIGHT}px`,
                minWidth: minRowWidth,
              }}
            >
              {/* Index cell */}
              <div
                className={cn(
                  "relative flex items-center shrink-0 text-sm leading-5 whitespace-nowrap break-all border-block-border border-r border-b group-even:bg-control-bg/40",
                  isActive && "bg-accent/10!",
                  rowSelected && "bg-accent/20!"
                )}
                data-col-index={0}
                style={{
                  height: `${ROW_HEIGHT}px`,
                  width: `${indexColWidth}px`,
                }}
              >
                <span
                  className={cn(
                    "textinfolabel pr-1 truncate",
                    selectionDisabled ? "pl-1" : "pl-4"
                  )}
                >
                  {rowIndex + 1}
                </span>
                {!selectionDisabled && (
                  <button
                    type="button"
                    aria-label={`Select row ${rowIndex + 1}`}
                    onClick={(e) => handleSelectRow(e, rowIndex)}
                    className={cn(
                      "absolute inset-y-0 left-0 w-3 cursor-pointer bg-accent/5 hover:bg-accent/10",
                      rowSelected && "bg-accent/20!"
                    )}
                  />
                )}
              </div>

              {/* Data cells */}
              {row.item.values.map((cell, columnIndex) => {
                const cellWidth = tableResize.getColumnWidth(columnIndex + 1);
                const isLastCol = columnIndex === row.item.values.length - 1;
                return (
                  <div
                    key={`${rowIndex}-${columnIndex + 1}`}
                    className={cn(
                      "relative shrink-0 text-sm leading-5 whitespace-nowrap break-all border-block-border border-b group-even:bg-control-bg/40",
                      !isLastCol && "border-r",
                      // Match the Vue version: an active (search-matched) row
                      // highlights every cell in that row, not just the index
                      // column.
                      isActive && "bg-accent/10!"
                    )}
                    data-col-index={columnIndex + 1}
                    style={{
                      height: `${ROW_HEIGHT}px`,
                      width: `${cellWidth}px`,
                    }}
                  >
                    <div className="h-full flex items-center overflow-hidden">
                      <TableCell
                        value={cell}
                        keyword={search.query}
                        scope={search.scopes.find(
                          (s) => s.id === columns[columnIndex]?.id
                        )}
                        rowIndex={rowIndex}
                        colIndex={columnIndex}
                        allowSelect
                        columnType={getColumnTypeByIndex(columnIndex)}
                        database={database}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
});

interface HeaderCellProps {
  width: number;
  borderRight?: boolean;
  onResizeStart: (e: React.PointerEvent) => void;
  onClick?: (e: React.MouseEvent) => void;
  className?: string;
  children?: React.ReactNode;
}

function HeaderCell({
  width,
  borderRight,
  onResizeStart,
  onClick,
  className,
  children,
}: HeaderCellProps) {
  return (
    <div
      onClick={onClick}
      className={cn(
        "group relative shrink-0 border-block-border",
        borderRight ? "border-x" : "border-l",
        className
      )}
      style={{ width: `${width}px` }}
    >
      {children}
      <span
        role="separator"
        aria-orientation="vertical"
        onPointerDown={onResizeStart}
        onClick={(e) => {
          e.stopPropagation();
          e.preventDefault();
        }}
        className="absolute w-2 right-0 top-0 bottom-0 cursor-col-resize"
      />
    </div>
  );
}
