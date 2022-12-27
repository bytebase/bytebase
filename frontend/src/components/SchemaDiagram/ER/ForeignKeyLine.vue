<template>
  <SVGLine
    :path="path"
    :decorators="[]"
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
import { Position, Rect, Path, ForeignKey } from "../types";
import {
  segmentOverlap1D,
  SegmentOverlap1D,
  useSchemaDiagramContext,
} from "../common";

type Direction = "LEFT" | "RIGHT";
type Segment1D = [number, number];

const props = withDefaults(
  defineProps<{
    fk: ForeignKey;
  }>(),
  {}
);

const { zoom, idOfTable, rectOfTable, events } = useSchemaDiagramContext();
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

const grow = (pos: Position, dir: Direction, length: number): Position => {
  const x = dir === "LEFT" ? pos.x - length : pos.x + length;
  return { x, y: pos.y };
};
const anchor = (rect: Rect, edge: Direction): Position => {
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

const updatePath = () => {
  const { from, to } = props.fk;
  fromRect.value = findRect(from.table, from.column);
  toRect.value = findRect(to.table, to.column);
};

events.on("render", () => void updatePath());

onMounted(updatePath);
</script>
