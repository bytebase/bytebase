import { useCallback, useRef, useState } from "react";

export interface ColumnWithWidth {
  defaultWidth: number;
  minWidth?: number;
}

/**
 * Tracks per-column widths for a `table-fixed` table and exposes a mouse
 * handler to start a drag-to-resize gesture on a column header. State is
 * positional: `widths[i]` corresponds to `columns[i]`.
 *
 * Widths are not persisted across remounts.
 */
export function useColumnWidths<T extends ColumnWithWidth>(columns: T[]) {
  const [widths, setWidths] = useState<number[]>(() =>
    columns.map((c) => c.defaultWidth)
  );
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
      };
      document.addEventListener("mousemove", onMouseMove);
      document.addEventListener("mouseup", onMouseUp);
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
    },
    [widths, columns]
  );

  const totalWidth = widths.reduce((sum, w) => sum + w, 0);

  return { widths, totalWidth, onResizeStart };
}
