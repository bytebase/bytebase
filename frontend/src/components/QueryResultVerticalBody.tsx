import { useVirtualizer } from "@tanstack/react-virtual";
import {
  forwardRef,
  type ReactNode,
  useCallback,
  useImperativeHandle,
  useRef,
} from "react";
import { cn } from "@/lib/utils";

export interface QueryResultVerticalBodyHandle {
  scrollTo(index: number): void;
}

export interface QueryResultVerticalBodyProps {
  rowCount: number;
  columnCount: number;
  renderRow: (rowIndex: number) => ReactNode;
  className?: string;
}

export const QueryResultVerticalBody = forwardRef<
  QueryResultVerticalBodyHandle,
  QueryResultVerticalBodyProps
>(function QueryResultVerticalBody(
  { rowCount, columnCount, renderRow, className },
  ref
) {
  const containerRef = useRef<HTMLDivElement>(null);
  const estimateRowHeight = useCallback(
    () => 60 + columnCount * 28,
    [columnCount]
  );
  const virtualizer = useVirtualizer({
    count: rowCount,
    getScrollElement: () => containerRef.current,
    estimateSize: estimateRowHeight,
    measureElement: (el) => el.getBoundingClientRect().height,
    overscan: 4,
    // Absolute rows need virtualizer spacing so visual gaps match scroll offsets.
    gap: 16,
    paddingEnd: 96,
  });

  useImperativeHandle(
    ref,
    () => ({
      scrollTo(index: number) {
        virtualizer.scrollToIndex(index, {
          align: "start",
          behavior: "smooth",
        });
      },
    }),
    [virtualizer]
  );

  return (
    <div
      ref={containerRef}
      data-query-result-vertical-body
      className={cn(
        "relative w-full min-h-0 flex-1 overflow-auto rounded-sm border border-block-border bg-background",
        className
      )}
    >
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          position: "relative",
        }}
      >
        {virtualizer.getVirtualItems().map((virtualRow) => (
          <div
            key={virtualRow.key}
            ref={virtualizer.measureElement}
            data-index={virtualRow.index}
            data-query-result-vertical-row
            className="absolute inset-x-0 mx-2 flex flex-col gap-y-1 font-mono"
            style={{ top: 0, transform: `translateY(${virtualRow.start}px)` }}
          >
            <p className="flex min-w-0 items-center gap-x-2 font-bold text-control-light">
              <span
                aria-hidden
                className="min-w-0 overflow-hidden whitespace-nowrap"
              >
                ********************************
              </span>
              <span className="shrink-0">{virtualRow.index + 1}. row</span>
              <span
                aria-hidden
                className="min-w-0 overflow-hidden whitespace-nowrap"
              >
                ********************************
              </span>
            </p>
            {renderRow(virtualRow.index)}
          </div>
        ))}
      </div>
    </div>
  );
});
