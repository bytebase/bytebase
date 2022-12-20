<template>
  <div
    class="absolute rounded-md shadow-lg border-b border-gray-200 bg-white w-[14rem] divide-y z-[10]"
    bb-node-type="table"
    :bb-node-id="idOfTable(table)"
    :style="{
      left: `${position.x}px`,
      top: `${position.y}px`,
    }"
  >
    <h3
      class="font-medium leading-6 text-white text-center truncate px-2 py-2 rounded-t-md"
      :style="{
        'background-color': tableColor,
      }"
    >
      {{ table.newName }}
    </h3>
    <table class="w-full text-sm">
      <tr
        v-for="(column, i) in table.columnList"
        :key="i"
        :bb-column-name="column.newName"
      >
        <td class="w-5 py-1.5">
          <heroicons-outline:key
            v-if="column.newName === 'emp_no' || column.newName === 'dept_no'"
            class="w-3 h-3 mx-auto text-amber-500"
          />
        </td>
        <td class="w-auto text-xs py-1.5">{{ column.newName }}</td>
        <td class="w-16 text-xs text-gray-400 text-right px-2 py-1.5">
          {{ column.type }}
        </td>
      </tr>
    </table>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { hashCode } from "@/bbkit/BBUtil";
import { Table } from "@/types/schemaEditor/atomType";
import { useSchemaDiagramContext } from "../common";

const props = withDefaults(
  defineProps<{
    table: Table;
  }>(),
  {}
);
const { idOfTable, rectOfTable } = useSchemaDiagramContext();

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
  const index = (hashCode(props.table.newName) & 0xfffffff) % COLOR_LIST.length;
  return COLOR_LIST[index];
});

const position = computed(() => rectOfTable(props.table));
</script>
