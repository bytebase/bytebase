import { useVirtualizer } from "@tanstack/react-virtual";
import {
  forwardRef,
  type ReactNode,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import { cn } from "@/lib/utils";
import type { ResultTableColumn, ResultTableRow } from "./types";
import { useTableResize } from "./useTableResize";

const ROW_HEIGHT = 35;
const MIN_COL_WIDTH = 64;
const MAX_COL_WIDTH = 640;

export interface QueryResultTableGridHandle {
  scrollTo(index: number): void;
}

interface QueryResultTableGridProps {
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  getRowCellContent: (columnIndex: number) => string | undefined;
  renderHeader: (column: ResultTableColumn, columnIndex: number) => ReactNode;
  renderRowIndex: (rowIndex: number) => ReactNode;
  renderCell: (rowIndex: number, columnIndex: number) => ReactNode;
  indexHeaderClassName?: string;
  headerClassName?: (columnIndex: number) => string | undefined;
  rowIndexClassName?: (rowIndex: number) => string | undefined;
  cellClassName?: (rowIndex: number, columnIndex: number) => string | undefined;
  onHeaderClick?: (event: React.MouseEvent, columnIndex: number) => void;
  onRowClick?: (rowIndex: number) => void;
  onRowDoubleClick?: (rowIndex: number) => void;
  className?: string;
}

export const QueryResultTableGrid = forwardRef<
  QueryResultTableGridHandle,
  QueryResultTableGridProps
>(function QueryResultTableGrid(
  {
    rows,
    columns,
    getRowCellContent,
    renderHeader,
    renderRowIndex,
    renderCell,
    indexHeaderClassName,
    headerClassName,
    rowIndexClassName,
    cellClassName,
    onHeaderClick,
    onRowClick,
    onRowDoubleClick,
    className,
  },
  ref
) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const update = () => setContainerWidth(container.clientWidth);
    update();
    const observer = new ResizeObserver(update);
    observer.observe(container);
    return () => observer.disconnect();
  }, []);

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

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const handleWheel = (event: WheelEvent) => {
      if (container.scrollWidth <= container.clientWidth) return;
      if (event.shiftKey) {
        event.preventDefault();
        event.stopPropagation();
        container.scrollLeft += event.deltaY || event.deltaX;
      } else if (Math.abs(event.deltaX) > Math.abs(event.deltaY)) {
        event.preventDefault();
        event.stopPropagation();
        container.scrollLeft += event.deltaX;
      }
    };
    container.addEventListener("wheel", handleWheel, {
      passive: false,
      capture: true,
    });
    return () =>
      container.removeEventListener("wheel", handleWheel, { capture: true });
  }, []);

  const minRowWidth = useMemo(
    () => `${tableResize.effectiveWidth}px`,
    [tableResize.effectiveWidth]
  );

  return (
    <div
      ref={containerRef}
      className={cn(
        "relative w-full flex-1 min-h-0 overflow-auto rounded-sm border border-block-border bg-background",
        className
      )}
      data-query-result-table-grid
      role="table"
    >
      <div
        className="sticky top-0 z-1 bg-control-bg flex shadow-sm"
        style={{ minWidth: minRowWidth }}
        role="row"
      >
        <HeaderCell
          width={tableResize.getColumnWidth(0)}
          borderRight
          onResizeStart={(event) => tableResize.onResizeStart(0, event)}
        >
          <div
            className={cn(
              "textinfolabel pr-1 opacity-0",
              indexHeaderClassName ?? "pl-1"
            )}
          >
            {rows.length}
          </div>
        </HeaderCell>
        {columns.map((column, columnIndex) => (
          <HeaderCell
            key={`${columnIndex}-${column.id}`}
            width={tableResize.getColumnWidth(columnIndex + 1)}
            borderRight={columnIndex < columns.length - 1}
            onResizeStart={(event) =>
              tableResize.onResizeStart(columnIndex + 1, event)
            }
            onClick={
              onHeaderClick
                ? (event) => onHeaderClick(event, columnIndex)
                : undefined
            }
            className={cn(
              "px-3 py-1.5 min-w-8 text-left text-xs font-medium text-control-light tracking-wider",
              headerClassName?.(columnIndex)
            )}
          >
            <div className="flex items-center min-w-0 gap-x-1">
              {renderHeader(column, columnIndex)}
            </div>
          </HeaderCell>
        ))}
      </div>
      <div
        role="rowgroup"
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          minWidth: minRowWidth,
          position: "relative",
        }}
      >
        {virtualizer.getVirtualItems().map((virtualRow) => {
          const rowIndex = virtualRow.index;
          if (!rows[rowIndex]) return null;
          return (
            <div
              key={virtualRow.key}
              data-row-index={rowIndex}
              role="row"
              className={cn(
                "flex absolute inset-x-0 group",
                onRowClick && "cursor-pointer"
              )}
              onClickCapture={
                onRowClick ? () => onRowClick(rowIndex) : undefined
              }
              onDoubleClick={
                onRowDoubleClick ? () => onRowDoubleClick(rowIndex) : undefined
              }
              style={{
                top: 0,
                transform: `translateY(${virtualRow.start}px)`,
                height: `${ROW_HEIGHT}px`,
                minWidth: minRowWidth,
              }}
            >
              <div
                className={cn(
                  "relative flex items-center shrink-0 text-sm text-control leading-5 whitespace-nowrap break-all border-block-border border-r border-b group-even:bg-control-bg/40",
                  rowIndexClassName?.(rowIndex)
                )}
                data-col-index={0}
                role="cell"
                style={{
                  height: `${ROW_HEIGHT}px`,
                  width: `${tableResize.getColumnWidth(0)}px`,
                }}
              >
                {renderRowIndex(rowIndex)}
              </div>
              {columns.map((_, columnIndex) => (
                <div
                  key={`${rowIndex}-${columnIndex + 1}`}
                  className={cn(
                    "relative shrink-0 text-sm text-control leading-5 whitespace-nowrap break-all border-block-border border-b group-even:bg-control-bg/40",
                    columnIndex < columns.length - 1 && "border-r",
                    cellClassName?.(rowIndex, columnIndex)
                  )}
                  data-col-index={columnIndex + 1}
                  role="cell"
                  style={{
                    height: `${ROW_HEIGHT}px`,
                    width: `${tableResize.getColumnWidth(columnIndex + 1)}px`,
                  }}
                >
                  <div className="h-full flex items-center overflow-hidden">
                    {renderCell(rowIndex, columnIndex)}
                  </div>
                </div>
              ))}
            </div>
          );
        })}
      </div>
    </div>
  );
});

function HeaderCell({
  width,
  borderRight,
  onResizeStart,
  onClick,
  className,
  children,
}: {
  width: number;
  borderRight?: boolean;
  onResizeStart: (event: React.PointerEvent) => void;
  onClick?: (event: React.MouseEvent) => void;
  className?: string;
  children?: ReactNode;
}) {
  return (
    <div
      onClick={onClick}
      role="columnheader"
      className={cn(
        "group relative shrink-0 border-block-border",
        borderRight && "border-r",
        className
      )}
      style={{ width: `${width}px` }}
    >
      {children}
      <span
        role="separator"
        aria-orientation="vertical"
        onPointerDown={onResizeStart}
        onClick={(event) => {
          event.stopPropagation();
          event.preventDefault();
        }}
        className="absolute w-2 right-0 top-0 bottom-0 cursor-col-resize"
      />
    </div>
  );
}
