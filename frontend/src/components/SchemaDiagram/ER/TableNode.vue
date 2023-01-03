<template>
  <div
    class="absolute overflow-hidden rounded-md shadow-lg border-b border-gray-200 bg-white w-[16rem] divide-y z-[10]"
    bb-node-type="table"
    :bb-node-id="idOfTable(table)"
    :bb-status="tableStatus(table)"
    :style="{
      left: `${position.x}px`,
      top: `${position.y}px`,
    }"
  >
    <h3
      class="font-medium leading-6 text-white truncate px-2 py-2 rounded-t-md flex items-center justify-center gap-x-1"
      :style="{
        'background-color': tableColor,
      }"
    >
      <span :class="[isTableDropped && 'line-through']">{{ table.name }}</span>
      <span v-if="isTableCreated" class="text-xs">(Created)</span>
      <span v-if="isTableDropped" class="text-xs">(Dropped)</span>
      <span v-if="isTableChanged" class="text-xs">(Changed)</span>
    </h3>
    <table class="w-full text-sm table-fixed">
      <tr
        v-for="(column, i) in table.columns"
        :key="i"
        :bb-column-name="column.name"
        :bb-status="columnStatus(column)"
        :class="[
          isDroppedColumn(column) && 'text-red-700 bg-red-50 line-through',
          isCreatedColumn(column) && 'text-green-700 bg-green-50',
        ]"
      >
        <td class="w-5 py-1.5">
          <heroicons-outline:key
            v-if="isPrimaryKey(table, column)"
            class="w-3 h-3 mx-auto text-amber-500"
          />
          <tabler:diamonds
            v-else-if="isIndex(table, column)"
            class="w-3 h-3 mx-auto text-gray-500"
          />
        </td>
        <td class="w-auto text-xs py-1.5">
          <div class="whitespace-pre-wrap break-words pr-1.5">
            {{ column.name }}
          </div>
        </td>
        <td
          class="w-[6rem] text-xs text-gray-400 py-1.5 pr-1.5 text-right truncate"
        >
          {{ column.type }}
        </td>
      </tr>
    </table>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { hashCode } from "@/bbkit/BBUtil";
import { ColumnMetadata, TableMetadata } from "@/types/proto/store/database";
import { useSchemaDiagramContext, isPrimaryKey, isIndex } from "../common";

const props = withDefaults(
  defineProps<{
    table: TableMetadata;
  }>(),
  {}
);
const { idOfTable, rectOfTable, tableStatus, columnStatus } =
  useSchemaDiagramContext();

const COLOR_LIST = [
  "#64748B",
  "#EF4444",
  "#F97316",
  "#EAB308",
  "#84CC16",
  "#22C55E",
  "#10B981",
  "#06B6D4",
  "#0EA5E9",
  "#3B82F6",
  "#6366F1",
  "#8B5CF6",
  "#A855F7",
  "#D946EF",
  "#EC4899",
  "#F43F5E",
];

const tableColor = computed(() => {
  const index = (hashCode(props.table.name) & 0xfffffff) % COLOR_LIST.length;
  return COLOR_LIST[index];
});

const isTableDropped = computed(() => {
  return tableStatus(props.table) === "dropped";
});

const isTableCreated = computed(() => {
  return tableStatus(props.table) === "created";
});

const isTableChanged = computed(() => {
  return tableStatus(props.table) === "changed";
});

const isDroppedColumn = (column: ColumnMetadata) => {
  return columnStatus(column) === "dropped";
};

const isCreatedColumn = (column: ColumnMetadata) => {
  return columnStatus(column) === "created";
};

const position = computed(() => rectOfTable(props.table));
</script>
