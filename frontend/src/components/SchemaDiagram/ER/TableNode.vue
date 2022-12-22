<template>
  <div
    class="absolute rounded-md shadow-lg border-b border-gray-200 bg-white w-[16rem] divide-y z-[10]"
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
      {{ table.name }}
    </h3>
    <table class="w-full text-sm table-fixed">
      <tr
        v-for="(column, i) in table.columns"
        :key="i"
        :bb-column-name="column.name"
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
        <td class="w-[6rem] text-xs text-gray-400 py-1.5 pr-1.5">
          <NEllipsis
            class="w-full text-right"
            :tooltip="{
              showArrow: false,
              contentStyle: tooltipStyle,
            }"
          >
            {{ column.type }}
          </NEllipsis>
        </td>
      </tr>
    </table>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NEllipsis } from "naive-ui";

import { hashCode } from "@/bbkit/BBUtil";
import { TableMetadata } from "@/types/proto/database";
import { useSchemaDiagramContext, isPrimaryKey, isIndex } from "../common";

const props = withDefaults(
  defineProps<{
    table: TableMetadata;
  }>(),
  {}
);
const { idOfTable, rectOfTable } = useSchemaDiagramContext();

const tooltipStyle = `
  max-width: 20rem;
  white-space: pre-wrap;
  word-break: break-all
`;

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

const position = computed(() => rectOfTable(props.table));
</script>
