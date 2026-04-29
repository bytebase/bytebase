import { useCallback, useEffect, useMemo, useState } from "react";
import { cn } from "@/react/lib/utils";
import type { TableMetadata } from "@/types/proto-es/v1/database_service_pb";
import { useGeometry, useSchemaDiagramContext } from "../common/context";
import { type SegmentOverlap1D, segmentOverlap1D } from "../common/geometry";
import type { ForeignKey, Path, Point, Rect } from "../types";
import { SVGLine } from "./libs/SVGLine";

type Direction = "LEFT" | "RIGHT";
type Segment1D = [number, number];

interface ForeignKeyLineProps {
  fk: ForeignKey;
}

const ZERO_RECT: Rect = { x: 0, y: 0, width: 0, height: 0 };

/**
 * React port of `ER/ForeignKeyLine.vue`. Reads each endpoint's column
 * rect by querying the live DOM (via `data-bb-node-id` /
 * `data-bb-column-name`), picks anchor sides, builds the SVG path, and
 * registers it as a geometry so fit-view's bbox includes the FK line.
 *
 * Recomputes when the context emits `render` (typically after layout
 * resolves or when a table is dragged).
 */
export function ForeignKeyLine({ fk }: ForeignKeyLineProps) {
  const ctx = useSchemaDiagramContext();
  const { zoom, focusedTables, idOfTable, rectOfTable, events } = ctx;

  // Recompute trigger — bumped when the diagram emits "render".
  const [, setTick] = useState(0);

  useEffect(() => {
    const off = events.on("render", () => setTick((n) => n + 1));
    return () => {
      off();
    };
  }, [events]);

  const findRect = useCallback(
    (table: TableMetadata, columnName: string): Rect => {
      const id = idOfTable(table);
      const tableRect = rectOfTable(table);
      const tableEl = document.querySelector(`[data-bb-node-id="${id}"]`);
      if (!tableEl) return ZERO_RECT;
      const colEl = tableEl.querySelector(
        `[data-bb-column-name="${columnName}"]`
      );
      if (colEl) {
        const tableView = tableEl.getBoundingClientRect();
        const columnView = colEl.getBoundingClientRect();
        return {
          x: tableRect.x + (columnView.left - tableView.left) / zoom,
          y: tableRect.y + (columnView.top - tableView.top) / zoom,
          width: columnView.width / zoom,
          height: columnView.height / zoom,
        };
      }
      return tableRect;
    },
    [idOfTable, rectOfTable, zoom]
  );

  // Read tableRects so a layout / drag triggers a re-render.
  // (rectOfTable is regenerated whenever the underlying tableRects state
  // changes, so listing it as a dep covers the layout-resolved case.)
  const fromRect = useMemo(
    () => findRect(fk.from.table, fk.from.column),
    [fk.from.table, fk.from.column, findRect]
  );
  const toRect = useMemo(
    () => findRect(fk.to.table, fk.to.column),
    [fk.to.table, fk.to.column, findRect]
  );

  const path = useMemo<Path>(() => {
    if (fromRect.width === 0 || toRect.width === 0) return [];
    const fromPorts: Segment1D = [fromRect.x, fromRect.x + fromRect.width];
    const toPorts: Segment1D = [toRect.x, toRect.x + toRect.width];
    const rel = segmentOverlap1D(
      fromPorts[0],
      fromPorts[1],
      toPorts[0],
      toPorts[1]
    );
    const [fromPort, toPort] = pickPorts(rel, fromPorts, toPorts);
    return generateLine(fromRect, fromPort, toRect, toPort);
  }, [fromRect, toRect]);

  const startManyArrow = useMemo<Path>(() => {
    if (path.length < 2) return [];
    const p = path[0];
    const q = path[1];
    const dir = Math.sign(q.x - p.x);
    const h = 12;
    const v = 6;
    return [
      { x: p.x, y: p.y + v },
      { x: p.x + h * dir, y: p.y },
      { x: p.x, y: p.y - v },
    ];
  }, [path]);

  const lineClasses = useMemo(() => {
    if (focusedTables.size === 0) return "";
    if (focusedTables.has(fk.from.table) || focusedTables.has(fk.to.table)) {
      return "opacity-100";
    }
    return "opacity-20";
  }, [focusedTables, fk]);

  useGeometry(path);

  return (
    <SVGLine
      className={cn("transition-opacity", lineClasses)}
      path={path}
      decorators={[startManyArrow]}
      data-bb-edge-from={`${fk.from.table.name}.${fk.from.column}`}
      data-bb-edge-to={`${fk.to.table.name}.${fk.to.column}`}
      data-bb-fk-name={fk.metadata.name}
      data-bb-fk-on-update={fk.metadata.onUpdate}
      data-bb-fk-on-delete={fk.metadata.onDelete}
      data-bb-fk-match-type={fk.metadata.matchType}
    />
  );
}

// ===== Pure helpers (extracted for testability — see ForeignKeyLine.test) =====

export const pickPorts = (
  rel: SegmentOverlap1D,
  a: Segment1D,
  b: Segment1D
): [Direction, Direction] => {
  switch (rel) {
    case "BEFORE":
      return ["RIGHT", "LEFT"];
    case "AFTER":
      return ["LEFT", "RIGHT"];
    case "CONTAINS":
    case "OVERLAPS":
      return b[0] <= (a[0] + a[1]) / 2 ? ["LEFT", "LEFT"] : ["RIGHT", "RIGHT"];
    case "OVERLAPPED":
    case "CONTAINED":
      return a[0] <= (b[0] + b[1]) / 2 ? ["LEFT", "LEFT"] : ["RIGHT", "RIGHT"];
  }
};

export const grow = (pos: Point, dir: Direction, length: number): Point => {
  const x = dir === "LEFT" ? pos.x - length : pos.x + length;
  return { x, y: pos.y };
};

export const anchor = (rect: Rect, edge: Direction): Point => {
  const x = edge === "LEFT" ? rect.x : rect.x + rect.width;
  const y = rect.y + rect.height / 2;
  return { x, y };
};

export const generateLine = (
  src: Rect,
  se: Direction,
  dest: Rect,
  de: Direction
): Path => {
  const GROWTH_LEN = 16;
  const sp = anchor(src, se);
  const dp = anchor(dest, de);
  const sc = grow(sp, se, GROWTH_LEN);
  const dc = grow(dp, de, GROWTH_LEN);
  return [sp, sc, dc, dp];
};
