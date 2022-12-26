<template>
  <Canvas>
    <template #desktop>
      <TableNode v-for="(table, i) in tableList" :key="i" :table="table" />
    </template>
  </Canvas>
</template>

<script lang="ts" setup>
import { computed, nextTick, onMounted, ref } from "vue";
import { uniqueId } from "lodash-es";
import Emittery from "emittery";

import { Database } from "@/types";
import { TableMetadata } from "@/types/proto/store/database";
import { Position, Rect, Size, SchemaDiagramContext } from "./types";
import Canvas from "./Canvas";
import { TableNode, autoLayout } from "./ER";
import { provideSchemaDiagramContext } from "./common";

const props = withDefaults(
  defineProps<{
    database: Database;
    tableList: TableMetadata[];
  }>(),
  {}
);

const zoom = ref(1);
const position = ref<Position>({ x: 0, y: 0 });

const render = () => {
  nextTick(() => {
    events.emit("render");
  });
};
const events: SchemaDiagramContext["events"] = new Emittery();

const tableIds = ref(new WeakMap<TableMetadata, string>());
const rectsByTableId = ref(new Map<string, Rect>());

const idOfTable = (table: TableMetadata): string => {
  const ids = tableIds.value;
  if (ids.has(table)) return ids.get(table)!;
  const id = `table-${table.name}-${uniqueId()}`;
  ids.set(table, id);
  return id;
};

const rectOfTable = (table: TableMetadata): Rect => {
  const id = idOfTable(table);
  return rectsByTableId.value.get(id) ?? { x: 0, y: 0, width: 0, height: 0 };
};

const layout = () => {
  nextTick(() => {
    const nodeList = props.tableList
      .map((table) => {
        const id = idOfTable(table);
        const elem = document.querySelector(`[bb-node-id="${id}"]`)!;
        return {
          table,
          id,
          elem,
        };
      })
      .filter((item) => !!item.elem)
      .map((item) => {
        const size: Size = {
          width: item.elem.clientWidth,
          height: item.elem.clientHeight,
        };
        return {
          ...item,
          size,
        };
      });
    const layout = autoLayout(nodeList, []);
    for (const [id, rect] of layout) {
      rectsByTableId.value.set(id, rect);
    }
  });

  render();
};

events.on("layout", layout);

provideSchemaDiagramContext({
  tableList: computed(() => props.tableList),
  zoom,
  position,
  idOfTable,
  rectOfTable,
  render,
  layout,
  events,
});

// autoLayout and fit view at the first time the diagram is mounted.
onMounted(() => {
  layout();
  nextTick(() => {
    events.emit("fit-view");
  });
});
</script>
