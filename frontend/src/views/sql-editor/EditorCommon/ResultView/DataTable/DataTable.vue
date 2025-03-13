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
              'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800':
                !selectionDisabled,
              '!bg-accent/10 dark:!bg-accent/40':
                selectionState.rows.length === 0 &&
                selectionState.columns.includes(header.index),
              'pl-3': header.index === 0,
            }"
            v-bind="tableResize.getColumnProps(header.index)"
            @click.stop="selectColumn(header.index)"
          >
            <div class="flex items-center overflow-hidden">
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
                @click.stop.prevent="
                  header.column.getToggleSortingHandler()?.($event)
                "
              />
              
              <!-- Add binary format button if this column has binary data -->
              <BinaryFormatButton
                v-if="isColumnWithBinaryData(header.index)"
                :column-index="header.index"
                :column-format="getColumnFormatOverride(header.index)"
                :server-format="getColumnServerFormat(header.index)"
                :has-single-bit-values="hasColumnSingleBitValues(header.index)"
                @update:format="setColumnFormat(header.index, $event)"
                @click.stop
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
            class="relative max-w-[50vw] p-0 text-sm dark:text-gray-100 leading-5 whitespace-nowrap break-all border-x border-b border-block-border dark:border-zinc-500 group-even:bg-gray-100/50 dark:group-even:bg-gray-700/50"
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
              :column-format-override="getColumnFormatOverride(cellIndex)"
            />
            <div
              v-if="cellIndex === 0 && !selectionDisabled"
              class="absolute inset-y-0 left-0 w-3 cursor-pointer hover:bg-accent/10 dark:hover:bg-accent/40"
              :class="{
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
import BinaryFormatButton from "./common/BinaryFormatButton.vue";
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

// Column format overrides - map of column index to format
const columnFormatOverrides = ref<Map<number, string | null>>(new Map());

// Check if a column contains any binary data
const isColumnWithBinaryData = (columnIndex: number): boolean => {
  const columnRows = props.table.getPrePaginationRowModel().rows;
  
  // Check each row in the column for binary data
  for (const row of columnRows) {
    const cell = row.getVisibleCells()[columnIndex];
    if (!cell) continue;
    
    const value = cell.getValue<RowValue>();
    if (value?.byteDataValue) {
      return true;
    }
  }
  
  return false;
};

// Get the server-provided format for a column
const getColumnServerFormat = (columnIndex: number): string | null => {
  const columnRows = props.table.getPrePaginationRowModel().rows;
  
  // Look through rows to find the first byte data value with a format
  for (const row of columnRows) {
    const cell = row.getVisibleCells()[columnIndex];
    if (!cell) continue;
    
    const value = cell.getValue<RowValue>();
    if (value?.byteDataValue?.displayFormat) {
      return value.byteDataValue.displayFormat;
    }
  }
  
  return null;
};

// Check if a column has any single-bit values
const hasColumnSingleBitValues = (columnIndex: number): boolean => {
  const columnRows = props.table.getPrePaginationRowModel().rows;
  
  // Check if any row in this column has a single-bit value
  for (const row of columnRows) {
    const cell = row.getVisibleCells()[columnIndex];
    if (!cell) continue;
    
    const value = cell.getValue<RowValue>();
    if (value?.byteDataValue?.value.length === 1) {
      return true;
    }
  }
  
  return false;
};

// Get the current format override for a column
const getColumnFormatOverride = (columnIndex: number): string | null => {
  return columnFormatOverrides.value.get(columnIndex) || null;
};

// Set the format for a column
const setColumnFormat = (columnIndex: number, format: string | null) => {
  if (format === null) {
    columnFormatOverrides.value.delete(columnIndex);
  } else {
    columnFormatOverrides.value.set(columnIndex, format);
  }
  
  // Force a re-render
  columnFormatOverrides.value = new Map(columnFormatOverrides.value);
};

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
