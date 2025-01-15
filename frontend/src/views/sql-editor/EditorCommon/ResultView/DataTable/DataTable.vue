<template>
  <div
    ref="containerRef"
    class="relative w-full flex-1 overflow-auto flex flex-col rounded border dark:border-zinc-500"
    :style="{
      maxHeight: maxHeight ? `${maxHeight}px` : undefined,
    }"
  >
    <table
      ref="tableRef"
      class="relative border-collapse w-full table-auto -mx-px"
      v-bind="tableResize.getTableProps()"
    >
      <thead
        class="bg-gray-50 dark:bg-gray-700 sticky top-0 z-[1] drop-shadow-sm"
      >
        <tr>
          <th
            v-for="header of table.getFlatHeaders()"
            :key="header.index"
            class="group relative px-2 py-2 min-w-[2rem] text-left text-xs font-medium text-gray-500 dark:text-gray-300 tracking-wider border-x border-block-border dark:border-zinc-500"
            :class="{
              '!bg-accent/10 dark:!bg-accent/40':
                selectionState.rows.length === 0 &&
                selectionState.columns.includes(header.index),
            }"
            v-bind="tableResize.getColumnProps(header.index)"
          >
            <div class="flex items-center overflow-hidden">
              <span
                class="flex flex-row items-center select-none"
                :class="{
                  'cursor-pointer hover:text-accent dark:hover:text-gray-500':
                    !selectionDisabled,
                }"
                @click.stop="selectColumn(header.index)"
              >
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
                @click.stop.prevent="
                  header.column.getToggleSortingHandler()?.($event)
                "
              />
            </div>

            <!-- The drag-to-resize handler -->
            <div
              class="absolute w-[8px] right-0 top-0 bottom-0 cursor-col-resize"
              @pointerdown="tableResize.startResizing(header.index)"
              @click.stop.prevent
            />
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
            class="relative p-0 text-sm dark:text-gray-100 leading-5 whitespace-nowrap break-all border-x border-b border-block-border dark:border-zinc-500 group-even:bg-gray-100/50 dark:group-even:bg-gray-700/50"
            :data-col-index="cellIndex"
          >
            <TableCell
              :table="table"
              :value="cell.getValue<RowValue>()"
              :keyword="keyword"
              :set-index="setIndex"
              :row-index="offset + rowIndex"
              :col-index="cellIndex"
              :allow-select="true"
            />
            <div
              v-if="cellIndex === 0"
              class="absolute inset-y-0 left-0 w-2"
              :class="{
                'cursor-pointer hover:bg-accent/10 dark:hover:bg-accent/40':
                  !selectionDisabled,
                'bg-accent/10 dark:bg-accent/40':
                  selectionState.columns.length === 0 &&
                  selectionState.rows.includes(offset + rowIndex),
              }"
              @click.prevent.stop="selectRow(offset + rowIndex)"
            ></div>
          </td>
        </tr>
      </tbody>
    </table>
    <div
      class="w-full sticky left-0 flex justify-center items-center py-12"
      v-if="rows.length === 0"
    >
      <NEmpty />
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Table } from "@tanstack/vue-table";
import { NEmpty } from "naive-ui";
import { computed, nextTick, onMounted, ref, watch } from "vue";
import {
  FeatureBadge,
  FeatureBadgeForInstanceLicense,
} from "@/components/FeatureGuard";
import { useSubscriptionV1Store } from "@/store";
import type { QueryRow, RowValue } from "@/types/proto/v1/sql_service";
import { useSQLResultViewContext } from "../context";
import TableCell from "./TableCell.vue";
import ColumnSortedIcon from "./common/ColumnSortedIcon.vue";
import SensitiveDataIcon from "./common/SensitiveDataIcon.vue";
import { useSelectionContext } from "./common/selection-logic";
import useTableColumnWidthLogic from "./useTableResize";

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
  selectRow,
} = useSelectionContext();
const containerRef = ref<HTMLDivElement>();
const tableRef = ref<HTMLTableElement>();
const subscriptionStore = useSubscriptionV1Store();

const tableResize = useTableColumnWidthLogic({
  tableRef,
  containerRef,
  minWidth: 64, // 4rem
  maxWidth: 640, // 40rem
});

const { keyword } = useSQLResultViewContext();

const hasSensitiveFeature = computed(() => {
  return subscriptionStore.hasFeature("bb.feature.sensitive-data");
});

const rows = computed(() => props.table.getRowModel().rows);

onMounted(() => {
  nextTick(() => {
    tableResize.reset();
  });
});

const scrollTo = (x: number, y: number) => {
  containerRef.value?.scroll(x, y);
};

watch(
  () => props.offset,
  () => {
    // When the offset changed, we need to reset the scroll position.
    scrollTo(0, 0);
  }
);
</script>
