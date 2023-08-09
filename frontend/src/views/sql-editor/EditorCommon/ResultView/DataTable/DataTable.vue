<template>
  <div
    class="bb-data-table relative w-full flex-1 overflow-hidden flex flex-col"
  >
    <div class="w-full flex-1 flex flex-col overflow-hidden">
      <div
        class="header-track absolute z-0 left-0 top-0 right-0 h-[34px] border border-block-border bg-gray-50 dark:bg-gray-700"
      />

      <div
        ref="scrollerRef"
        class="inner-wrapper max-h-full w-full overflow-auto border-y border-r border-block-border fix-scrollbar-z-index"
        :class="data.length === 0 && 'border-b-0 border-r-0'"
      >
        <table
          ref="tableRef"
          class="relative border-collapse table-fixed z-[1]"
          v-bind="tableResize.getTableProps()"
        >
          <thead class="sticky top-0 z-[1] shadow">
            <tr>
              <th
                v-for="header of table.getFlatHeaders()"
                :key="header.index"
                class="relative px-2 py-2 min-w-[2rem] text-left bg-gray-50 dark:bg-gray-700 text-xs font-medium text-gray-500 dark:text-gray-300 tracking-wider border border-t-0 border-block-border border-b-0"
                v-bind="tableResize.getColumnProps(header.index)"
              >
                <div class="flex items-center overflow-hidden">
                  <span>
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
                  <FeatureBadgeForInstanceLicense
                    v-else-if="isColumnMissingSensitive(header.index)"
                    :show="true"
                    custom-class="ml-0.5 shrink-0"
                    feature="bb.feature.sensitive-data"
                  />
                </div>

                <!-- The drag-to-resize handler -->
                <div
                  class="absolute w-[8px] right-0 top-0 bottom-0 cursor-col-resize"
                  @dblclick="tableResize.autoAdjustColumnWidth([header.index])"
                  @pointerdown="tableResize.startResizing(header.index)"
                />
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(row, rowIndex) of table.getRowModel().rows"
              :key="rowIndex"
              class="group"
              :data-row-index="offset + rowIndex"
            >
              <td
                v-for="(cell, cellIndex) of row.getVisibleCells()"
                :key="cellIndex"
                class="p-0 text-sm dark:text-gray-100 leading-5 whitespace-nowrap break-all border border-block-border group-last:border-b-0 group-even:bg-gray-50/50 dark:group-even:bg-gray-700/50"
                :data-col-index="cellIndex"
              >
                <TableCell
                  :value="cell.getValue()"
                  :keyword="keyword"
                  :set-index="setIndex"
                  :row-index="offset + rowIndex"
                  :col-index="cellIndex"
                />
              </td>
            </tr>
          </tbody>
        </table>
        <div
          v-if="data.length === 0"
          class="text-center w-full my-12 textinfolabel"
        >
          {{ $t("sql-editor.no-rows-found") }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ColumnDef, Table } from "@tanstack/vue-table";
import { computed, nextTick, PropType, ref, watch } from "vue";
import SensitiveDataIcon from "./SensitiveDataIcon.vue";
import TableCell from "./TableCell.vue";
import useTableColumnWidthLogic from "./useTableResize";

export type DataTableColumn = {
  key: string;
  title: string;
};

const props = defineProps({
  data: {
    type: Array as PropType<string[][]>,
    default: () => [],
  },
  columns: {
    type: Array as PropType<ColumnDef<string[]>[]>,
    default: () => [],
  },
  sensitive: {
    type: Array as PropType<boolean[]>,
    default: () => [],
  },
  masked: {
    type: Array as PropType<boolean[]>,
    default: () => [],
  },
  table: {
    type: Object as PropType<Table<string[]>>,
    required: true,
  },
  keyword: {
    type: String,
    default: "",
  },
  setIndex: {
    type: Number,
    default: 0,
  },
  offset: {
    type: Number,
    default: 0,
  },
});

const scrollerRef = ref<HTMLDivElement>();
const tableRef = ref<HTMLTableElement>();

const tableResize = useTableColumnWidthLogic({
  tableRef,
  scrollerRef,
  minWidth: 64, // 4rem
  maxWidth: 640, // 40rem
});

const data = computed(() => props.data);

const isSensitiveColumn = (index: number): boolean => {
  return props.masked[index] ?? false;
};

const isColumnMissingSensitive = (index: number): boolean => {
  return (props.sensitive[index] ?? false) && !isSensitiveColumn(index);
};

const scrollTo = (x: number, y: number) => {
  scrollerRef.value?.scroll(x, y);
};

watch(
  () => props.columns.map((col) => col.header).join("|"),
  () => {
    nextTick(() => {
      // Re-calculate the column widths once the column definition changed.
      scrollTo(0, 0);
      tableResize.reset();
    });
  },
  { immediate: true }
);

defineExpose({ scrollTo });
</script>
