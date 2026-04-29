import { useCallback, useRef, useState } from "react";
import { cn } from "@/react/lib/utils";

/**
 * Shared visual styling for resize handles inside SchemaEditorLite.
 *
 * One source of truth for thickness + colors across:
 *   - the horizontal Aside | Editor split (driven by `react-resizable-panels`)
 *   - the vertical Editor / Preview split (driven by `usePersistedDragSize`)
 *
 * Both handles render as a 4px `bg-control-border` track that highlights to
 * `bg-accent` on hover or while being dragged. Pick the orientation that
 * matches the *split direction* you're separating, not the handle's own
 * shape:
 *
 *   - `vertical` — vertical bar between two horizontally-stacked panels
 *   - `horizontal` — horizontal bar between two vertically-stacked panels
 */
export type ResizeHandleOrientation = "vertical" | "horizontal";

const RESIZE_HANDLE_BASE_CLASS =
  "shrink-0 bg-control-border transition-colors hover:bg-accent data-[resize-handle-active]:bg-accent";

export function resizeHandleClass(
  orientation: ResizeHandleOrientation,
  className?: string
): string {
  return cn(
    RESIZE_HANDLE_BASE_CLASS,
    orientation === "vertical"
      ? "w-1 cursor-ew-resize"
      : "h-1 w-full cursor-ns-resize",
    className
  );
}

// ---- usePersistedDragSize -------------------------------------------------

/**
 * Which side of the resize handle is being sized. The handle separates two
 * panels; the consumer specifies which one this hook tracks.
 *
 * Examples:
 *   - PreviewPane sits BELOW its top-edge handle and grows upward as the
 *     handle is dragged up → `axis: "y"`, `growsToward: "before"`.
 *   - A right-side details pane sits to the RIGHT of its left-edge handle
 *     and grows rightward as the handle is dragged right → `axis: "x"`,
 *     `growsToward: "after"`.
 */
export type ResizeAxis = "x" | "y";
export type SizeGrowth = "before" | "after";

export interface UsePersistedDragSizeOptions {
  /** localStorage key for persisting the chosen size between sessions. */
  storageKey: string;
  /** Which mouse coordinate drives the resize. */
  axis: ResizeAxis;
  /**
   * Direction the tracked panel grows in *handle coordinates*:
   *   - `before` — dragging toward smaller coordinate values grows it
   *     (e.g. dragging the top-edge handle UPWARD grows a panel below).
   *   - `after`  — dragging toward larger coordinate values grows it.
   */
  growsToward: SizeGrowth;
  defaultSize: number;
  minSize: number;
  maxSize: number;
}

export interface UsePersistedDragSizeResult {
  size: number;
  handleResizeStart: (event: React.MouseEvent) => void;
}

const isClient = typeof window !== "undefined";

function readPersistedSize(
  storageKey: string,
  fallback: number,
  minSize: number,
  maxSize: number
): number {
  if (!isClient) return fallback;
  try {
    const raw = window.localStorage.getItem(storageKey);
    if (raw === null) return fallback;
    const value = Number(raw);
    if (Number.isFinite(value) && value >= minSize && value <= maxSize) {
      return value;
    }
  } catch {
    // ignore – fallback to default
  }
  return fallback;
}

function writePersistedSize(storageKey: string, size: number): void {
  if (!isClient) return;
  try {
    window.localStorage.setItem(storageKey, String(size));
  } catch {
    // ignore – non-fatal, just won't persist
  }
}

/**
 * Hook that pairs a draggable resize handle with localStorage persistence.
 *
 * Returns the current `size` (clamped to [minSize, maxSize]) and a
 * `handleResizeStart` to wire up to a handle's `onMouseDown`. While the user
 * drags, the document cursor is locked to the appropriate resize cursor and
 * text selection is suppressed; both are cleaned up on release.
 */
export function usePersistedDragSize(
  options: UsePersistedDragSizeOptions
): UsePersistedDragSizeResult {
  const { storageKey, axis, growsToward, defaultSize, minSize, maxSize } =
    options;

  const [size, setSize] = useState(() =>
    readPersistedSize(storageKey, defaultSize, minSize, maxSize)
  );
  // Track latest size in a ref so the mouseup persistence sees the final
  // value (the closure captured by addEventListener freezes initial state).
  const sizeRef = useRef(size);
  sizeRef.current = size;

  const handleResizeStart = useCallback(
    (event: React.MouseEvent) => {
      event.preventDefault();
      const startCoord = axis === "y" ? event.clientY : event.clientX;
      const startSize = sizeRef.current;
      const sign = growsToward === "before" ? -1 : 1;

      const onMove = (ev: MouseEvent) => {
        const current = axis === "y" ? ev.clientY : ev.clientX;
        const delta = (current - startCoord) * sign;
        const next = Math.max(minSize, Math.min(maxSize, startSize + delta));
        setSize(next);
      };
      const onUp = () => {
        document.removeEventListener("mousemove", onMove);
        document.removeEventListener("mouseup", onUp);
        writePersistedSize(storageKey, sizeRef.current);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      };
      document.addEventListener("mousemove", onMove);
      document.addEventListener("mouseup", onUp);
      document.body.style.cursor = axis === "y" ? "ns-resize" : "ew-resize";
      document.body.style.userSelect = "none";
    },
    [axis, growsToward, storageKey, minSize, maxSize]
  );

  return { size, handleResizeStart };
}
