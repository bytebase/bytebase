import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

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
  // Closures should capture state eagerly via refs, not via React deps.
  // Putting `widths` in onResizeStart's dep array would rebind the callback
  // on every mousemove (since each tick calls setWidths), which would
  // re-render every header consumer mid-drag. Mirror the latest values
  // into refs via useLayoutEffect (runs synchronously after commit, before
  // paint and before any subsequent event handler can fire) so onResizeStart
  // stays referentially stable AND avoids render-phase ref writes that
  // React 19 concurrent mode could expose from a discarded render.
  const widthsRef = useRef(widths);
  const columnsRef = useRef(columns);
  useLayoutEffect(() => {
    widthsRef.current = widths;
    columnsRef.current = columns;
  });

  const onResizeStart = useCallback((colIndex: number, e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    // Defensively tear down any in-flight drag before starting a new one.
    // A second onResizeStart without an intervening mouseup (rapid
    // sequential mousedowns, missed mouseup outside the document,
    // multi-touch trackpad, programmatic dispatch) would otherwise leak
    // the prior drag's document listeners and orphan its cleanup.
    dragCleanupRef.current();
    dragRef.current = {
      colIndex,
      startX: e.clientX,
      startWidth: widthsRef.current[colIndex],
    };

    const onMouseMove = (ev: MouseEvent) => {
      if (!dragRef.current) return;
      // Capture everything we need before scheduling the state update.
      // The setWidths updater may run AFTER teardown nulls dragRef.current
      // (mouseup → React's pending-update queue → next render's
      // basicStateReducer), so the updater closure must not deref the ref.
      const { colIndex, startX, startWidth } = dragRef.current;
      const delta = ev.clientX - startX;
      const min = columnsRef.current[colIndex].minWidth ?? 40;
      const newWidth = Math.max(min, startWidth + delta);
      setWidths((prev) => {
        const next = [...prev];
        next[colIndex] = newWidth;
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
  }, []);

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
