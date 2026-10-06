import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

export interface ColumnWithWidth {
  defaultWidth: number;
  /** The narrowest a drag or `distributeColumnWidths` may make the column. */
  minWidth?: number;
  /**
   * False for a column whose content has a width of its own, such as a date,
   * so it takes none of the spare width. Tables that leave spare width to the
   * browser do not read it.
   */
  grow?: boolean;
  /** When this column gives way in a table too narrow for its defaults. */
  yieldOrder?: number;
}

const DEFAULT_MIN_WIDTH = 40;
const floorOf = (c: ColumnWithWidth) => c.minWidth ?? DEFAULT_MIN_WIDTH;

/**
 * Distributes `containerWidth` across columns so the table fills its container
 * on first render instead of overflowing at the sum of `defaultWidth`s: the
 * widths add up to exactly the container unless it cannot hold every column
 * at the narrowest the rules below allow, and then the table scrolls. A
 * column that is not resizable always keeps its `defaultWidth`.
 *
 * - Narrower than the defaults together, with any column declaring a
 *   `yieldOrder`: columns give way in ascending order, each down to its
 *   `minWidth` before the next gives any, and those without an order last.
 * - Otherwise: columns that do not grow keep their `defaultWidth`, and the
 *   rest share the remaining space in proportion to theirs, none below its
 *   `minWidth`.
 */
export function distributeColumnWidths<
  T extends ColumnWithWidth & { resizable?: boolean },
>(columns: T[], containerWidth: number): number[] {
  const preferredTotal = columns.reduce((sum, c) => sum + c.defaultWidth, 0);
  if (
    containerWidth < preferredTotal &&
    columns.some((c) => c.yieldOrder !== undefined)
  ) {
    return yieldInOrder(columns, containerWidth);
  }
  const keepsWidth = (c: T) => c.resizable === false || c.grow === false;
  const fixedTotal = columns
    .filter(keepsWidth)
    .reduce((sum, c) => sum + c.defaultWidth, 0);
  const available = Math.max(0, containerWidth - fixedTotal);

  // Raising a column to its floor has to take that width from the others, so
  // floored columns are set aside and the remainder re-shared until every
  // share clears its floor. Each pass sets aside at least one column.
  const floored = new Set<T>();
  let shares = new Map<T, number>();
  for (;;) {
    const open = columns.filter((c) => !keepsWidth(c) && !floored.has(c));
    const base = open.reduce((sum, c) => sum + c.defaultWidth, 0);
    const rest =
      available - [...floored].reduce((sum, c) => sum + floorOf(c), 0);
    shares = new Map(
      open.map((c) => [c, base > 0 ? (rest * c.defaultWidth) / base : 0])
    );
    const under = open.filter((c) => (shares.get(c) ?? 0) < floorOf(c));
    if (under.length === 0) {
      break;
    }
    for (const c of under) {
      floored.add(c);
    }
  }

  const widths = columns.map((c) => {
    if (keepsWidth(c)) return c.defaultWidth;
    if (floored.has(c)) return floorOf(c);
    return Math.floor(shares.get(c) ?? 0);
  });
  // Flooring each share leaves a few pixels unassigned; the widest column
  // still sharing takes them, so the total lands on the container.
  const sharing = columns
    .map((_, i) => i)
    .filter((i) => shares.has(columns[i]));
  if (sharing.length > 0) {
    const widest = sharing.reduce((a, b) => (widths[b] > widths[a] ? b : a));
    widths[widest] += containerWidth - widths.reduce((sum, w) => sum + w, 0);
  }
  return widths;
}

/**
 * The width a table can take inside `scroller` without scrolling it, which
 * even half a pixel too many does. Borders and padding come from the computed
 * style, exact even where the browser snaps a 1px border to 0.8px at 125%;
 * `offsetWidth` and `clientWidth` each round to a pixel, so a scrollbar is
 * known only to within one and is given that pixel.
 */
export function fillableWidth(scroller: HTMLElement): number {
  const style = getComputedStyle(scroller);
  const px = (value: string) => Number.parseFloat(value) || 0;
  const borders = px(style.borderLeftWidth) + px(style.borderRightWidth);
  const padding = px(style.paddingLeft) + px(style.paddingRight);
  const scrollbar = scroller.offsetWidth - scroller.clientWidth - borders;
  const scrollbarAllowance = scrollbar > 1 ? scrollbar + 1 : 0;
  return Math.floor(
    scroller.getBoundingClientRect().width -
      borders -
      padding -
      scrollbarAllowance
  );
}

// Starts every column at its default, or its floor if that is wider, and
// takes back what overruns the container a yield group at a time: within a
// group in proportion to how far each column can shrink.
function yieldInOrder<T extends ColumnWithWidth & { resizable?: boolean }>(
  columns: T[],
  containerWidth: number
): number[] {
  const orderOf = (c: T) => c.yieldOrder ?? Number.POSITIVE_INFINITY;
  const widths = columns.map((c) =>
    c.resizable === false
      ? c.defaultWidth
      : Math.max(c.defaultWidth, floorOf(c))
  );
  let remaining = widths.reduce((sum, w) => sum + w, 0) - containerWidth;
  const orders = [
    ...new Set(columns.filter((c) => c.resizable !== false).map(orderOf)),
  ].sort((a, b) => a - b);
  for (const order of orders) {
    if (remaining <= 0) {
      break;
    }
    const group = columns
      .map((_, i) => i)
      .filter(
        (i) => columns[i].resizable !== false && orderOf(columns[i]) === order
      );
    const room = group.map((i) => widths[i] - floorOf(columns[i]));
    const groupRoom = room.reduce((sum, r) => sum + r, 0);
    const take = Math.min(remaining, groupRoom);
    let taken = 0;
    group.forEach((i, j) => {
      const cut = groupRoom > 0 ? Math.floor((take * room[j]) / groupRoom) : 0;
      widths[i] -= cut;
      taken += cut;
    });
    // Flooring each cut leaves a few pixels; the widest in the group gives
    // them, within its floor.
    for (const i of [...group].sort((a, b) => widths[b] - widths[a])) {
      const cut = Math.min(take - taken, widths[i] - floorOf(columns[i]));
      widths[i] -= cut;
      taken += cut;
    }
    remaining -= take;
  }
  return widths;
}

/**
 * Tracks per-column widths for a `table-fixed` table and exposes a mouse
 * handler to start a drag-to-resize gesture on a column header. State is
 * positional: `widths[i]` corresponds to `columns[i]`.
 *
 * Widths are not persisted across remounts.
 *
 * Eager-capture contract: at drag start the hook snapshots colIndex,
 * startX, startWidth, and minWidth into a closure-local drag record.
 * Once captured, the in-flight drag is immune to subsequent changes in
 * `widths` or `columns` (reordering, equal-length swaps, minWidth
 * tweaks, even removal). Callers don't need to memoize aggressively —
 * just don't expect a drag to react to column changes mid-gesture.
 *
 * If a caller grows the column array after mount without remounting the
 * hook, `widths` state stays at its initial length until the next user
 * resize on the new column. The drag's `startWidth` falls back to the
 * column's `defaultWidth` so the gesture starts from a sensible value
 * rather than NaN.
 */
export function useColumnWidths<T extends ColumnWithWidth>(columns: T[]) {
  const [widths, setWidths] = useState<number[]>(() =>
    columns.map((c) => c.defaultWidth)
  );
  const dragRef = useRef<{
    colIndex: number;
    startX: number;
    startWidth: number;
    minWidth: number;
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
    // Defensive: the user clicked a header that was just rendered, so the
    // column at this index should be defined. Guard against the edge case
    // where columnsRef has been synced to a shorter array via a commit
    // that's interleaved with this click.
    const col = columnsRef.current[colIndex];
    if (!col) return;
    dragRef.current = {
      colIndex,
      startX: e.clientX,
      // Fall back to defaultWidth if widths state hasn't grown to include
      // a column added after mount (widths state is sized at first render
      // and never auto-extends). Without this, startWidth would be
      // undefined and the drag's newWidth math would produce NaN.
      startWidth: widthsRef.current[colIndex] ?? col.defaultWidth,
      minWidth: floorOf(col),
    };

    const onMouseMove = (ev: MouseEvent) => {
      if (!dragRef.current) return;
      // Read only the snapshot — see "Eager-capture contract" in JSDoc.
      const { colIndex, startX, startWidth, minWidth } = dragRef.current;
      const delta = ev.clientX - startX;
      const newWidth = Math.max(minWidth, startWidth + delta);
      setWidths((prev) => {
        // Short-circuit no-op updates. While clamped at minWidth (cursor
        // moving farther past the edge), this fires once per pixel of
        // mouse motion with the same newWidth — without this guard,
        // React would still re-render every consumer per tick.
        if (prev[colIndex] === newWidth) return prev;
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

  return { widths, totalWidth, onResizeStart, setWidths };
}
