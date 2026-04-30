import { ScanSearch } from "lucide-react";
import { useCallback, useMemo } from "react";
import { cn } from "@/react/lib/utils";
import type { TableMetadata } from "@/types/proto-es/v1/database_service_pb";
import { DEFAULT_PADDINGS } from "./const";
import { useSchemaDiagramContext } from "./context";
import { expectedZoomRange } from "./utils";

interface FocusButtonProps {
  table: TableMetadata;
  /** Extra classes when the table is currently focused. */
  focusedClass?: string;
  /** When true, clicking re-centers the canvas on the table. Default: true. */
  setCenter?: boolean;
  className?: string;
}

/**
 * React port of `frontend/src/components/SchemaDiagram/common/FocusButton.vue`.
 *
 * Toggles the `focusedTables` membership for the given table and (by
 * default) emits `set-center` so the canvas re-centers + zooms onto it.
 * Used inside Navigator tree node suffixes and on hover over each
 * table card in the canvas.
 */
export function FocusButton({
  table,
  focusedClass = "",
  setCenter = true,
  className,
}: FocusButtonProps) {
  const ctx = useSchemaDiagramContext();
  const { zoom, focusedTables, setFocusedTables, events } = ctx;

  const isFocused = useMemo(
    () => focusedTables.has(table),
    [focusedTables, table]
  );

  const toggleFocus = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      const next = new Set(focusedTables);
      const turningOn = !next.has(table);
      if (turningOn) {
        next.add(table);
      } else {
        next.delete(table);
      }
      setFocusedTables(next);
      if (setCenter) {
        void events.emit("set-center", {
          type: "table",
          target: table,
          padding: DEFAULT_PADDINGS,
          zooms: turningOn ? expectedZoomRange(zoom, 0.5, 1) : undefined,
        });
      }
    },
    [table, focusedTables, setFocusedTables, setCenter, events, zoom]
  );

  return (
    <button
      type="button"
      onClick={toggleFocus}
      className={cn(
        "p-0.5 rounded-sm hover:bg-control-bg",
        isFocused ? "visible" : "invisible",
        isFocused && focusedClass,
        className
      )}
    >
      <ScanSearch className="size-4" />
    </button>
  );
}
