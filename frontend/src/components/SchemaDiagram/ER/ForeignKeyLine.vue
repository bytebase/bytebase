<template>
  <SVGLine
    class="transition-opacity"
    :class="lineClasses"
    :path="path"
    :decorators="[startManyArrow]"
    :bb-edge-from="`${fk.from.table.name}.${fk.from.column}`"
    :bb-edge-to="`${fk.to.table.name}.${fk.to.column}`"
    :bb-fk-name="fk.metadata.name"
    :bb-fk-on-update="fk.metadata.onUpdate"
    :bb-fk-on-delete="fk.metadata.onDelete"
    :bb-fk-match-type="fk.metadata.matchType"
  />
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { TableMetadata } from "@/types/proto/store/database";
import type { VueClass } from "@/utils";
import {
  segmentOverlap1D,
  SegmentOverlap1D,
  useGeometry,
  useSchemaDiagramContext,
} from "../common";
import { Point, Rect, Path, ForeignKey } from "../types";

type Direction = "LEFT" | "RIGHT";
type Segment1D = [number, number];

const props = withDefaults(
  defineProps<{
    fk: ForeignKey;
  }>(),
  {}
);

const { zoom, focusedTables, idOfTable, rectOfTable, events } =
  useSchemaDiagramContext();
const fromRect = ref<Rect>({ x: 0, y: 0, width: 0, height: 0 });
const toRect = ref<Rect>({ x: 0, y: 0, width: 0, height: 0 });
const path = computed((): Path => {
  const from = fromRect.value;
  const to = toRect.value;

  if (from.width === 0 || to.width === 0) {
    return [];
  }

  const fromPorts: Segment1D = [from.x, from.x + from.width];
  const toPorts: Segment1D = [to.x, to.x + to.width];

  const rel = segmentOverlap1D(
    fromPorts[0],
    fromPorts[1],
    toPorts[0],
    toPorts[1]
  );
  const [fromPort, toPort] = pickPorts(rel, fromPorts, toPorts);

  return generateLine(from, fromPort, to, toPort);
});

const lineClasses = computed((): VueClass => {
  const classes: string[] = [];
  if (focusedTables.value.size > 0) {
    const { from, to } = props.fk;
    if (
      focusedTables.value.has(from.table) ||
      focusedTables.value.has(to.table)
    ) {
      classes.push("opacity-100");
    } else {
      classes.push("opacity-20");
    }
  }
  return classes;
});

const findRect = (table: TableMetadata, columnName: string): Rect => {
  const id = idOfTable(table);
  const tableRect = rectOfTable(table);
  const tableElement = document.querySelector(`[bb-node-id="${id}"]`);
  if (!tableElement) {
    return { x: 0, y: 0, width: 0, height: 0 };
  }
  const columnElement = tableElement.querySelector(
    `[bb-column-name="${columnName}"]`
  );
  if (columnElement) {
    const tableView = tableElement.getBoundingClientRect();
    const columnView = columnElement.getBoundingClientRect();
    const columnRect = {
      x: tableRect.x + (columnView.left - tableView.left) / zoom.value,
      y: tableRect.y + (columnView.top - tableView.top) / zoom.value,
      width: columnView.width / zoom.value,
      height: columnView.height / zoom.value,
    };
    return columnRect;
  }
  return tableRect;
};

// A field has two "ports" - left and right.
// For a field-field relationship a-b, we need to find which ports
// of a and b we should use to make the connection line looks natural.
const pickPorts = (
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

const grow = (pos: Point, dir: Direction, length: number): Point => {
  const x = dir === "LEFT" ? pos.x - length : pos.x + length;
  return { x, y: pos.y };
};
const anchor = (rect: Rect, edge: Direction): Point => {
  const x = edge === "LEFT" ? rect.x : rect.x + rect.width;
  const y = rect.y + rect.height / 2;
  return { x, y };
};
const generateLine = (src: Rect, se: Direction, dest: Rect, de: Direction) => {
  const GROWTH_LEN = 16;
  const sp = anchor(src, se); // source point
  const dp = anchor(dest, de); // destination point
  const sc = grow(sp, se, GROWTH_LEN); // source corner
  const dc = grow(dp, de, GROWTH_LEN); // destination corner
  return [sp, sc, dc, dp];
};
const startManyArrow = computed((): Path => {
  if (path.value.length < 2) return [];
  /**
   * Create a "fork"-like line to indicate "to-many" relationship
   *       m
   *     /
   * q--c--p
   *     \
   *       n
   */
  const p = path.value[0];
  const q = path.value[1];
  const dir = Math.sign(q.x - p.x);
  const h = 12;
  const v = 6;
  const m = { x: p.x, y: p.y + v };
  const c = { x: p.x + h * dir, y: p.y };
  const n = { x: p.x, y: p.y - v };
  return [m, c, n];
});

const updatePath = () => {
  const { from, to } = props.fk;
  fromRect.value = findRect(from.table, from.column);
  toRect.value = findRect(to.table, to.column);
};

events.on("render", () => void updatePath());

onMounted(updatePath);

useGeometry(path);
</script>
