import { sum } from "lodash-es";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ResultTableColumn, ResultTableRow } from "./types";

export interface TableResizeOptions {
  columns: ResultTableColumn[];
  rows: ResultTableRow[];
  containerWidth: number;
  minWidth: number;
  maxWidth: number;
  /**
   * Plain-text view of cell at (rowIndex 0..2, columnIndex). Used to size
   * the first few sample rows so columns roughly fit their actual content.
   */
  getRowCellContent: (columnIndex: number) => string | undefined;
}

interface DragState {
  colIndex: number;
  startX: number;
  startWidth: number;
}

const HEADER_FONT = "12px ui-sans-serif, system-ui, sans-serif";
const CELL_FONT = "14px ui-monospace, monospace";
// Reserved width inside the header for everything other than the title:
// px-3 padding (24, both sides) + gap-x-1 (4) + sort icon size-3.5 (14) +
// gap-x-1 (4) + binary-format button size-5 (20) + a few px breathing
// room. We assume the worst case (binary button present) so columns
// without one have a small slack at the trailing edge — better than
// short titles being clipped behind the sort icon.
const HEADER_EXTRA = 72;
const CELL_PADDING = 16; // px-2 cell padding × 2

let measureCanvas: HTMLCanvasElement | null = null;
const measureTextWidth = (text: string, font: string): number => {
  if (!measureCanvas) measureCanvas = document.createElement("canvas");
  const ctx = measureCanvas.getContext("2d");
  if (!ctx) return 0;
  ctx.font = font;
  return ctx.measureText(text).width;
};

/**
 * Measures and tracks per-column widths for the virtualized result-table
 * grid. Replaces Vue's `useTableResize` composable. Returns widths inclusive
 * of the leading index column at slot 0.
 */
export function useTableResize({
  columns,
  rows,
  containerWidth,
  minWidth,
  maxWidth,
  getRowCellContent,
}: TableResizeOptions) {
  const computeInitialWidths = useCallback((): number[] => {
    // Index column: enough to fit the largest row number.
    const indexWidth =
      Math.ceil(measureTextWidth(String(rows.length || 1), CELL_FONT)) + 32; // padding + select strip

    // Data columns: max(header text, sample content rows) clamped to [min, max].
    const dataWidths = columns.map((col, idx) => {
      const headerText = String(col.name ?? "");
      const headerWidth =
        Math.ceil(measureTextWidth(headerText, HEADER_FONT)) + HEADER_EXTRA;

      let bestSample = 0;
      const sample = getRowCellContent(idx);
      if (sample !== undefined && sample.length > 0) {
        bestSample =
          Math.ceil(measureTextWidth(sample, CELL_FONT)) + CELL_PADDING;
      }
      const candidate = Math.max(headerWidth, bestSample);
      if (columns.length === 1) {
        return Math.max(minWidth, Math.min(containerWidth - 1, candidate));
      }
      return Math.max(minWidth, Math.min(maxWidth, candidate));
    });

    return [indexWidth, ...dataWidths];
  }, [
    columns,
    rows.length,
    containerWidth,
    minWidth,
    maxWidth,
    getRowCellContent,
  ]);

  const [widths, setWidths] = useState<number[]>(() => computeInitialWidths());
  const dragRef = useRef<DragState | null>(null);

  const reset = useCallback(() => {
    setWidths(computeInitialWidths());
  }, [computeInitialWidths]);

  // Recompute when columns change identity OR window/container width changes.
  // We deliberately key on columns.length+col ids, not deep-watch, since the
  // Vue version did the same via `watchDebounced` of column references.
  const columnsKey = useMemo(
    () => `${columns.length}:${columns.map((c) => c.id).join(",")}`,
    [columns]
  );
  useEffect(() => {
    setWidths(computeInitialWidths());
  }, [columnsKey, computeInitialWidths]);

  const onResizeStart = useCallback(
    (colIndex: number, e: React.PointerEvent) => {
      e.preventDefault();
      e.stopPropagation();
      const startWidth = widths[colIndex];
      if (startWidth === undefined) return;
      dragRef.current = {
        colIndex,
        startX: e.clientX,
        startWidth,
      };

      const isLast = colIndex === widths.length - 1;
      const handleMove = (ev: PointerEvent) => {
        const drag = dragRef.current;
        if (!drag) return;
        const offset = ev.clientX - drag.startX;
        let next = drag.startWidth + offset;
        if (widths.length === 1) {
          next = Math.max(minWidth, Math.min(containerWidth - 1, next));
        } else {
          next = Math.max(minWidth, Math.min(maxWidth, next));
        }
        setWidths((prev) => {
          const out = [...prev];
          out[drag.colIndex] = next;
          return out;
        });
      };
      const handleUp = () => {
        dragRef.current = null;
        document.removeEventListener("pointermove", handleMove);
        document.removeEventListener("pointerup", handleUp);
        document.body.classList.remove("cursor-col-resize");
      };
      document.addEventListener("pointermove", handleMove);
      document.addEventListener("pointerup", handleUp);
      document.body.classList.add("cursor-col-resize");
      // satisfy linter for unused var on cleanup path
      void isLast;
    },
    [widths, minWidth, maxWidth, containerWidth]
  );

  const totalWidth = sum(widths);
  const effectiveWidth = Math.max(totalWidth, containerWidth);

  /**
   * Pixel width for the column at this index. The last column absorbs any
   * extra space between totalWidth and containerWidth so the row never has
   * a trailing gap.
   */
  const getColumnWidth = useCallback(
    (index: number): number => {
      const w = widths[index];
      if (w === undefined) return 0;
      const isLast = index === widths.length - 1;
      const extra = Math.max(0, containerWidth - totalWidth);
      return isLast ? w + extra : w;
    },
    [widths, containerWidth, totalWidth]
  );

  return {
    widths,
    totalWidth,
    effectiveWidth,
    onResizeStart,
    reset,
    getColumnWidth,
  };
}
