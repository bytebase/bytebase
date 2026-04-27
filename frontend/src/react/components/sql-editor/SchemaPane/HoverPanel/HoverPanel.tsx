import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { cn } from "@/react/lib/utils";
import type { Position } from "@/types";
import { minmax } from "@/utils";
import { useHoverState } from "../hover-state";
import { ColumnInfo } from "./ColumnInfo";
import { ExternalTableInfo } from "./ExternalTableInfo";
import { TableInfo } from "./TableInfo";
import { TablePartitionInfo } from "./TablePartitionInfo";
import { ViewInfo } from "./ViewInfo";

type Props = {
  readonly offsetX: number;
  readonly offsetY: number;
  readonly margin: number;
  readonly onClickOutside?: (e: MouseEvent) => void;
};

/**
 * Replaces `HoverPanel/HoverPanel.vue`. Floating, cursor-anchored panel
 * that previews schema metadata for the hovered tree row. Mirrors Vue's
 * `v-if` chain — most-specific shape wins:
 *   column + table → ColumnInfo
 *   table  + partition → TablePartitionInfo
 *   table              → TableInfo
 *   externalTable      → ExternalTableInfo
 *   view               → ViewInfo
 *
 * Position is driven by `useHoverState()`'s sparse `state` + cursor
 * `position`. The y coordinate is clamped so the panel never escapes the
 * viewport. Mounted into the shared overlay layer via a portal so the
 * absolute coords are document-relative regardless of where SchemaPane
 * lives in the DOM.
 *
 * Hover-keep: mouseenter on the panel itself reschedules the open delay
 * to 50ms so the panel doesn't flash off when the cursor crosses from a
 * row onto the panel. mouseleave triggers the standard 350ms close.
 */
export function HoverPanel({
  offsetX,
  offsetY,
  margin,
  onClickOutside,
}: Props) {
  const { state, position, update } = useHoverState();
  const popoverRef = useRef<HTMLDivElement | null>(null);
  const [popoverHeight, setPopoverHeight] = useState(0);

  useLayoutEffect(() => {
    const el = popoverRef.current;
    if (!el) return;
    setPopoverHeight(el.getBoundingClientRect().height);
    const ro = new ResizeObserver(() => {
      setPopoverHeight(el.getBoundingClientRect().height);
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, [state]);

  const show = state !== undefined && position.x !== 0 && position.y !== 0;

  const displayPosition = useMemo<Position>(() => {
    const p: Position = {
      x: position.x + offsetX,
      y: position.y + offsetY,
    };
    if (typeof window !== "undefined") {
      const yMin = margin;
      const yMax = window.innerHeight - popoverHeight - margin;
      p.y = minmax(p.y, yMin, yMax);
    }
    return p;
  }, [position, offsetX, offsetY, margin, popoverHeight]);

  useEffect(() => {
    if (!show || !onClickOutside) return;
    const handler = (e: MouseEvent) => {
      const el = popoverRef.current;
      if (el && !el.contains(e.target as Node)) {
        onClickOutside(e);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [show, onClickOutside]);

  // Render nothing at all when there's no state to show — avoids a stray
  // 0×0 element sitting at top-left until the user hovers something.
  if (!state) return null;

  let body: React.ReactNode = null;
  if (state.column && state.table) {
    body = (
      <ColumnInfo
        database={state.database}
        schema={state.schema}
        table={state.table}
        column={state.column}
      />
    );
  } else if (state.table && state.partition) {
    body = (
      <TablePartitionInfo
        database={state.database}
        schema={state.schema}
        table={state.table}
        partition={state.partition}
      />
    );
  } else if (state.table) {
    body = (
      <TableInfo
        database={state.database}
        schema={state.schema}
        table={state.table}
      />
    );
  } else if (state.externalTable) {
    body = (
      <ExternalTableInfo
        database={state.database}
        schema={state.schema}
        externalTable={state.externalTable}
      />
    );
  } else if (state.view) {
    body = (
      <ViewInfo
        database={state.database}
        schema={state.schema}
        view={state.view}
      />
    );
  }

  const panel = (
    <div
      ref={popoverRef}
      className={cn(
        "fixed border border-gray-100 rounded-sm bg-white p-2 shadow-sm transition-[top] text-sm",
        LAYER_SURFACE_CLASS,
        !show && "invisible pointer-events-none"
      )}
      style={{
        left: `${displayPosition.x}px`,
        top: `${displayPosition.y}px`,
      }}
      onMouseEnter={() => update(state, "before", 50)}
      onMouseLeave={() => update(undefined, "after")}
    >
      {body}
    </div>
  );

  if (typeof document === "undefined") return null;
  return createPortal(panel, getLayerRoot("overlay"));
}
