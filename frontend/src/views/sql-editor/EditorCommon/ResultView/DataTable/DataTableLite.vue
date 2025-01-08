<template>
  <div
    ref="containerRef"
    class="relative w-full flex-1 overflow-auto flex flex-col"
    :style="{
      maxHeight: maxHeight ? `${maxHeight}px` : undefined,
    }"
  >
    <table ref="tableRef" class="relative border-collapse table-auto">
      <thead
        class="sticky top-0 z-[1] drop-shadow-sm bg-gray-50 dark:bg-gray-700"
      >
        <tr>
          <th
            v-for="header of table.getFlatHeaders()"
            :key="header.index"
            class="relative min-w-[2rem] max-w-[50vw] text-left text-xs font-medium text-gray-500 dark:text-gray-300 tracking-wider border-l first:border-l-0 border-block-border dark:border-zinc-500"
            :class="{
              'cursor-pointer': !selectionDisabled,
              'bg-accent/10 dark:bg-accent/40': selectionState.columns.includes(
                header.index
              ),
            }"
            @click.stop="selectColumn(header.index)"
          >
            <div class="px-2 py-2 flex items-center overflow-hidden">
              <span class="flex flex-row items-center select-none">
                <template
                  v-if="String(header.column.columnDef.header).length > 0"
                >
                  {{ header.column.columnDef.header }}
                </template>
                <br v-else class="min-h-[1rem] inline-flex" />
              </span>

              <SensitiveDataIcon
                v-if="isSensitiveColumn(header.index)"
                class="ml-0.5 shrink-0"
              />
              <template v-else-if="isColumnMissingSensitive(header.index)">
                <FeatureBadgeForInstanceLicense
                  v-if="hasSensitiveFeature"
                  :show="true"
                  custom-class="ml-0.5 shrink-0"
                  feature="bb.feature.sensitive-data"
                />
                <FeatureBadge
                  v-else
                  feature="bb.feature.sensitive-data"
                  custom-class="ml-0.5 shrink-0"
                />
              </template>

              <ColumnSortedIcon
                :is-sorted="header.column.getIsSorted()"
                @click="header.column.getToggleSortingHandler()?.($event)"
              />
            </div>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(row, rowIndex) of rows"
          :key="rowIndex"
          class="group"
          :data-row-index="offset + rowIndex"
        >
          <td
            v-for="(cell, cellIndex) of row.getVisibleCells()"
            :key="cellIndex"
            class="relative min-w-8 max-w-[50vw] p-0 text-sm dark:text-gray-100 leading-5 whitespace-nowrap break-all border-l first:border-l-0 border-block-border dark:border-zinc-500 group-even:bg-gray-100/50 dark:group-even:bg-gray-700/50"
            :data-col-index="cellIndex"
          >
            <TableCell
              :table="table"
              :value="cell.getValue() as RowValue"
              :keyword="keyword"
              :set-index="setIndex"
              :row-index="offset + rowIndex"
              :col-index="cellIndex"
            />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script lang="ts" setup>
import type { Table } from "@tanstack/vue-table";
import { computed, nextTick, ref, watch } from "vue";
import {
  FeatureBadge,
  FeatureBadgeForInstanceLicense,
} from "@/components/FeatureGuard";
import { useSubscriptionV1Store } from "@/store";
import type { QueryRow, RowValue } from "@/types/proto/v1/sql_service";
import ColumnSortedIcon from "../DataTable/common/ColumnSortedIcon.vue";
import SensitiveDataIcon from "../DataTable/common/SensitiveDataIcon.vue";
import { useSQLResultViewContext } from "../context";
import TableCell from "./TableCell.vue";
import { useSelectionContext } from "./common/selection-logic";

export type DataTableColumn = {
  key: string;
  title: string;
};

const props = defineProps<{
  table: Table<QueryRow>;
  setIndex: number;
  offset: number;
  isSensitiveColumn: (index: number) => boolean;
  isColumnMissingSensitive: (index: number) => boolean;
  maxHeight?: number;
}>();

const {
  state: selectionState,
  disabled: selectionDisabled,
  selectColumn,
} = useSelectionContext();
const containerRef = ref<HTMLDivElement>();
const tableRef = ref<HTMLTableElement>();
const subscriptionStore = useSubscriptionV1Store();

const { keyword } = useSQLResultViewContext();

const hasSensitiveFeature = computed(() => {
  return subscriptionStore.hasFeature("bb.feature.sensitive-data");
});

const scrollTo = (x: number, y: number) => {
  containerRef.value?.scroll(x, y);
};

const rows = computed(() => props.table.getRowModel().rows);

watch(
  () =>
    props.table
      .getFlatHeaders()
      .map((header) => String(header.column.columnDef.header))
      .join("|"),
  () => {
    nextTick(() => {
      // Re-calculate the column widths once the column definition changed.
      scrollTo(0, 0);
    });
  },
  { immediate: true }
);

watch(
  () => props.offset,
  () => {
    // When the offset changed, we need to reset the scroll position.
    scrollTo(0, 0);
  }
);
</script>
