import { useCallback, useRef, useState } from "react";

export interface ColumnDef {
  /** Unique column identifier, also used as sessionStorage key suffix. */
  key: string;
  /** Default width in pixels. */
  defaultWidth: number;
  /** Minimum width in pixels. Defaults to 40. */
  minWidth?: number;
  /** Whether this column can be resized. Defaults to true. */
  resizable?: boolean;
}

/**
 * Manages column widths with drag-to-resize support.
 * Use with table-fixed layout: set the table width to totalWidth
 * and each col to its pixel width. Wrap in overflow-x-auto for scrolling.
 *
 * @param columns - Column definitions with default/min widths.
 * @param storageKey - Optional key for persisting widths to sessionStorage.
 */
export function useColumnWidths(columns: ColumnDef[], storageKey?: string) {
  const [widths, setWidths] = useState<number[]>(() => {
    if (storageKey) {
      try {
        const saved = sessionStorage.getItem(storageKey);
        if (saved) {
          const parsed = JSON.parse(saved) as Record<string, number>;
          return columns.map((c) => parsed[c.key] ?? c.defaultWidth);
        }
      } catch {
        // ignore
      }
    }
    return columns.map((c) => c.defaultWidth);
  });

  const dragRef = useRef<{
    colIndex: number;
    startX: number;
    startWidth: number;
  } | null>(null);

  const onResizeStart = useCallback(
    (colIndex: number, e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      dragRef.current = {
        colIndex,
        startX: e.clientX,
        startWidth: widths[colIndex],
      };

      const onMouseMove = (ev: MouseEvent) => {
        if (!dragRef.current) return;
        const delta = ev.clientX - dragRef.current.startX;
        const min = columns[dragRef.current.colIndex].minWidth ?? 40;
        const newWidth = Math.max(min, dragRef.current.startWidth + delta);
        setWidths((prev) => {
          const next = [...prev];
          next[dragRef.current!.colIndex] = newWidth;
          return next;
        });
      };
      const onMouseUp = () => {
        dragRef.current = null;
        document.removeEventListener("mousemove", onMouseMove);
        document.removeEventListener("mouseup", onMouseUp);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        if (storageKey) {
          setWidths((current) => {
            const record: Record<string, number> = {};
            columns.forEach((c, i) => {
              record[c.key] = current[i];
            });
            try {
              sessionStorage.setItem(storageKey, JSON.stringify(record));
            } catch {
              // ignore
            }
            return current;
          });
        }
      };
      document.addEventListener("mousemove", onMouseMove);
      document.addEventListener("mouseup", onMouseUp);
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
    },
    [widths, columns, storageKey]
  );

  const totalWidth = widths.reduce((sum, w) => sum + w, 0);

  return { widths, totalWidth, onResizeStart };
}
