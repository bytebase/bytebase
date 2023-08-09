<template>
  <div class="w-full h-full relative overflow-hidden flex">
    <Navigator />

    <Canvas class="flex-1">
      <template #desktop>
        <template v-for="(schema, i) in schemaList" :key="`schema-${i}`">
          <TableNode
            v-for="table in schema.tables"
            :key="idOfTable(table)"
            :schema="schema"
            :table="table"
            :class="initialized ? '' : 'invisible'"
          />
        </template>

        <template v-if="initialized">
          <ForeignKeyLine v-for="(fk, i) in foreignKeys" :key="i" :fk="fk" />
        </template>
      </template>

      <div
        v-if="busy || !initialized"
        class="absolute inset-0 bg-white/40 flex items-center justify-center"
      >
        <BBSpin />
      </div>
    </Canvas>
  </div>
</template>

<script lang="ts" setup>
import Emittery from "emittery";
import { uniqueId } from "lodash-es";
import { computed, nextTick, ref, toRef, watch } from "vue";
import type { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";
import Canvas from "./Canvas";
import { TableNode, autoLayout, GraphNodeItem, GraphEdgeItem } from "./ER";
import Navigator from "./Navigator";
import { provideSchemaDiagramContext } from "./common";
import {
  Point,
  Rect,
  Size,
  SchemaDiagramContext,
  ForeignKey,
  EditStatus,
  Geometry,
} from "./types";

const props = withDefaults(
  defineProps<{
    database: ComposedDatabase;
    databaseMetadata: DatabaseMetadata;
    editable?: boolean;
    schemaStatus?: (schema: SchemaMetadata) => EditStatus;
    tableStatus?: (table: TableMetadata) => EditStatus;
    columnStatus?: (column: ColumnMetadata) => EditStatus;
  }>(),
  {
    editable: false,
    schemaStatus: () => "normal" as EditStatus,
    tableStatus: () => "normal" as EditStatus,
    columnStatus: () => "normal" as EditStatus,
  }
);

const emit = defineEmits<{
  (event: "edit-table", schema: SchemaMetadata, table: TableMetadata): void;
  (
    event: "edit-column",
    schema: SchemaMetadata,
    table: TableMetadata,
    column: ColumnMetadata,
    target: "name" | "type"
  ): void;
}>();

const schemaList = computed(() => {
  return props.databaseMetadata.schemas;
});
const initialized = ref(false);
const dummy = ref(false);
const busy = ref(false);
const zoom = ref(1);
const position = ref<Point>({ x: 0, y: 0 });
const panning = ref(false);
const geometries = ref(new Set<Geometry>());
const focusedTables = ref(new Set<TableMetadata>());

const render = () => {
  nextTick(() => {
    events.emit("render");
  });
};
const events: SchemaDiagramContext["events"] = new Emittery();

const tableIds = ref(new WeakMap<TableMetadata, string>());
const rectsByTableId = ref(new Map<string, Rect>());
const foreignKeys = computed((): ForeignKey[] => {
  const find = (s: string, t: string, c: string) => {
    const schema = props.databaseMetadata.schemas.find(
      (schema) => schema.name === s
    )!;
    const table = schema.tables.find((table) => table.name === t)!;
    const column = c;
    return { schema, table, column };
  };
  const fks: ForeignKey[] = [];
  schemaList.value.forEach((schema) => {
    schema.tables.forEach((table) => {
      table.foreignKeys.forEach((fkMetadata) => {
        const {
          columns,
          referencedSchema,
          referencedTable,
          referencedColumns,
        } = fkMetadata;
        for (let i = 0; i < columns.length; i++) {
          fks.push({
            from: { schema, table, column: columns[i] },
            to: find(referencedSchema, referencedTable, referencedColumns[i]),
            metadata: fkMetadata,
          });
        }
      });
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
    const nodeList = schemaList.value
      .flatMap((schema) => {
        return schema.tables.map((table) => {
          const id = idOfTable(table);
          const elem = document.querySelector(`[bb-node-id="${id}"]`)!;
          return {
            group: `schema-${schema.name}`,
            table,
            id,
            elem,
          };
        });
      })
      .filter((item) => !!item.elem)
      .map<GraphNodeItem>((item) => {
        const size: Size = {
          width: item.elem.clientWidth,
          height: item.elem.clientHeight,
        };
        return {
          group: item.group,
          id: item.id,
          size,
          children: [],
        };
      });

    const edgeList = foreignKeys.value.map<GraphEdgeItem>((fk) => {
      const { from, to } = fk;
      const fromTableId = idOfTable(from.table);
      const toTableId = idOfTable(to.table);
      return {
        id: `${from.schema.name}.${fromTableId}.${from.column}->${to.schema.name}.${toTableId}.${to.column}`,
        from: fromTableId,
        to: toTableId,
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
events.on("edit-table", ({ schema, table }) =>
  emit("edit-table", schema, table)
);
events.on("edit-column", ({ schema, table, column, target }) =>
  emit("edit-column", schema, table, column, target)
);

provideSchemaDiagramContext({
  database: toRef(props, "database"),
  databaseMetadata: toRef(props, "databaseMetadata"),
  editable: computed(() => props.editable),
  foreignKeys,
  dummy,
  busy,
  zoom,
  position,
  panning,
  geometries,
  focusedTables,
  idOfTable,
  rectOfTable,
  render,
  layout,
  schemaStatus: props.schemaStatus,
  tableStatus: props.tableStatus,
  columnStatus: props.columnStatus,
  events,
});

// autoLayout and fit view at the first time the diagram is mounted.
watch(
  () => props.databaseMetadata,
  async () => {
    focusedTables.value = new Set();
    await layout();
    initialized.value = true;
    nextTick(() => {
      events.emit("fit-view");
    });
  },
  { immediate: true }
);
</script>
