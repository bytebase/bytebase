import { useCallback, useEffect, useRef, useState } from "react";

export interface ColumnDef {
  /** Unique column identifier, also used as sessionStorage key suffix. */
  key: string;
  /** Default width in pixels — used as the proportional weight when fitting to container. */
  defaultWidth: number;
  /** Minimum width in pixels. Defaults to 40. */
  minWidth?: number;
  /** Whether this column can be resized. Defaults to true. */
  resizable?: boolean;
}

/**
 * Manages column widths with drag-to-resize support.
 *
 * On mount, measures the container (via containerRef) and scales default widths
 * proportionally to fill it. Returns a ref to attach to the scrollable wrapper.
 *
 * @param columns - Column definitions with default/min widths.
 * @param storageKey - Optional key for persisting widths to sessionStorage.
 */
export function useColumnWidths(columns: ColumnDef[], storageKey?: string) {
  const containerRef = useRef<HTMLDivElement>(null);
  const defaultTotal = columns.reduce((s, c) => s + c.defaultWidth, 0);

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

  // On mount, scale widths to fill container if no saved widths exist
  const initialized = useRef(false);
  useEffect(() => {
    if (initialized.current) return;
    initialized.current = true;
    const container = containerRef.current;
    if (!container) return;
    // Skip if widths were restored from storage
    if (storageKey) {
      try {
        if (sessionStorage.getItem(storageKey)) return;
      } catch {
        // ignore
      }
    }
    const containerWidth = container.clientWidth;
    if (containerWidth > defaultTotal) {
      // Only scale resizable columns; fixed columns keep their default width
      const fixedWidth = columns.reduce(
        (s, c) => s + (c.resizable === false ? c.defaultWidth : 0),
        0
      );
      const resizableDefault = defaultTotal - fixedWidth;
      const resizableTarget = containerWidth - fixedWidth;
      const scale =
        resizableDefault > 0 ? resizableTarget / resizableDefault : 1;
      setWidths(
        columns.map((c) =>
          c.resizable === false
            ? c.defaultWidth
            : Math.round(c.defaultWidth * scale)
        )
      );
    }
  }, [columns, defaultTotal, storageKey]);

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

  return { containerRef, widths, totalWidth, onResizeStart };
}
