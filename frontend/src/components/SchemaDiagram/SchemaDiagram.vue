<template>
  <Canvas>
    <template #desktop>
      <TableNode v-for="(table, i) in tableList" :key="i" :table="table" />
    </template>
  </Canvas>
</template>

<script lang="ts" setup>
import { computed, nextTick, ref, watch } from "vue";
import { uniqueId } from "lodash-es";
import Emittery from "emittery";

import { Database } from "@/types";
import { Table } from "@/types/schemaEditor/atomType";
import { Position, Rect, Size, SchemaDiagramContext } from "./types";
import Canvas from "./Canvas";
import { TableNode, autoLayout } from "./ER";
import { provideSchemaDiagramContext } from "./common";

const props = withDefaults(
  defineProps<{
    database: Database;
    tableList: Table[];
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

const tableIds = ref(new WeakMap<Table, string>());
const rectsByTableId = ref(new Map<string, Rect>());

const idOfTable = (table: Table): string => {
  const ids = tableIds.value;
  if (ids.has(table)) return ids.get(table)!;
  const id = `table-${table.newName}-${uniqueId()}`;
  ids.set(table, id);
  return id;
};

const rectOfTable = (table: Table): Rect => {
  const id = idOfTable(table);
  return rectsByTableId.value.get(id) ?? { x: 0, y: 0, width: 0, height: 0 };
};

const moveTable = (table: Table, dx: number, dy: number) => {
  const rect = rectOfTable(table);
  rect.x += dx;
  rect.y += dy;
  render();
};

provideSchemaDiagramContext({
  tableList: computed(() => props.tableList),
  zoom,
  position,
  idOfTable,
  rectOfTable,
  moveTable,
  render,
  events,
});

watch(
  () => props.tableList,
  (tableList) => {
    nextTick(() => {
      const nodeList = tableList
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

      render();
      events.emit("fit-view");
    });
  },
  { immediate: true }
);
</script>
