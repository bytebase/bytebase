import { useCallback, useEffect, useRef, useState } from "react";

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
  // Holds the teardown function for an active drag so an unmount-mid-drag
  // (route change, modal close, etc.) tears down document-level listeners
  // and resets the body cursor/userSelect overrides.
  const dragCleanupRef = useRef<() => void>(() => {});

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
      const teardown = () => {
        dragRef.current = null;
        document.removeEventListener("mousemove", onMouseMove);
        document.removeEventListener("mouseup", teardown);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        // Reset so the unmount-cleanup is a no-op if the drag ended cleanly.
        dragCleanupRef.current = () => {};
      };
      document.addEventListener("mousemove", onMouseMove);
      document.addEventListener("mouseup", teardown);
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
      dragCleanupRef.current = teardown;
    },
    [widths, columns]
  );

  useEffect(() => {
    return () => {
      // If unmount happens mid-drag, tear down before any stale listener
      // can fire against the unmounted tree.
      dragCleanupRef.current();
    };
  }, []);

  const totalWidth = widths.reduce((sum, w) => sum + w, 0);

  return { widths, totalWidth, onResizeStart };
}
