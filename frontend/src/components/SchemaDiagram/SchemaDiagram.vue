<template>
  <Canvas>
    <template #desktop>
      <TableNode
        v-for="(table, i) in tableList"
        :key="i"
        :table="table"
        :class="initialized ? '' : 'invisible'"
      />

      <template v-if="initialized">
        <ForeignKeyLine v-for="(fk, i) in foreignKeys" :key="i" :fk="fk" />
      </template>
    </template>

    <div
      v-if="!initialized"
      class="absolute inset-0 bg-white/40 flex items-center justify-center"
    >
      <BBSpin />
    </div>
  </Canvas>
</template>

<script lang="ts" setup>
import { computed, nextTick, onMounted, ref } from "vue";
import { uniqueId } from "lodash-es";
import Emittery from "emittery";

import { Database } from "@/types";
import { TableMetadata } from "@/types/proto/store/database";
import {
  Position,
  Rect,
  Size,
  SchemaDiagramContext,
  ForeignKey,
} from "./types";
import Canvas from "./Canvas";
import { TableNode, autoLayout, GraphNodeItem, GraphEdgeItem } from "./ER";
import { provideSchemaDiagramContext } from "./common";

const props = withDefaults(
  defineProps<{
    database: Database;
    tableList: TableMetadata[];
  }>(),
  {}
);

const initialized = ref(false);
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
const foreignKeys = computed((): ForeignKey[] => {
  const find = (t: string, c: string) => {
    const table = props.tableList.find((table) => table.name === t)!;
    const column = c;
    return { table, column };
  };
  const fks: ForeignKey[] = [];
  props.tableList.forEach((table) => {
    table.foreignKeys.forEach((fkMetadata) => {
      const { columns, referencedTable, referencedColumns } = fkMetadata;
      for (let i = 0; i < columns.length; i++) {
        fks.push({
          from: { table, column: columns[i] },
          to: find(referencedTable, referencedColumns[i]),
          metadata: fkMetadata,
        });
      }
    });
  });

  return fks;
});

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
  return nextTick(async () => {
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
      .map<GraphNodeItem>((item) => {
        const size: Size = {
          width: item.elem.clientWidth,
          height: item.elem.clientHeight,
        };
        return {
          id: item.id,
          size,
          children: [],
        };
      });

    const edgeList = foreignKeys.value.map<GraphEdgeItem>((fk) => {
      const { from, to } = fk;
      return {
        id: `${from.table.name}.${from.column}->${to.table.name}.${to.column}`,
        from: idOfTable(from.table),
        to: idOfTable(to.table),
      };
    });
    const { rects } = await autoLayout(nodeList, edgeList);
    for (const [id, rect] of rects) {
      rectsByTableId.value.set(id, rect);
    }

    render();
  });
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
onMounted(async () => {
  await layout();
  initialized.value = true;
  nextTick(() => {
    events.emit("fit-view");
  });
});
</script>
