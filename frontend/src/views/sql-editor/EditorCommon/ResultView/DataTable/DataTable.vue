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
        :class="rows.length === 0 && 'border-b-0 border-r-0'"
      >
        <table
          ref="tableRef"
          class="relative border-collapse table-fixed z-[1]"
          v-bind="tableResize.getTableProps()"
        >
          <thead class="sticky top-0 z-[1] drop-shadow-sm">
            <tr>
              <th
                v-for="header of table.getFlatHeaders()"
                :key="header.index"
                class="relative px-2 py-2 min-w-[2rem] text-left bg-gray-50 dark:bg-gray-700 text-xs font-medium text-gray-500 dark:text-gray-300 tracking-wider border border-t-0 border-block-border border-b-0"
                v-bind="tableResize.getColumnProps(header.index)"
              >
                <div
                  class="flex items-center overflow-hidden cursor-pointer"
                  @click="header.column.getToggleSortingHandler()?.($event)"
                >
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

                  <ColumnSortedIcon :is-sorted="header.column.getIsSorted()" />
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
              v-for="(row, rowIndex) of rows"
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
                  :table="table"
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
          v-if="rows.length === 0"
          class="text-center w-full my-12 textinfolabel"
        >
          {{ $t("sql-editor.no-rows-found") }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Table } from "@tanstack/vue-table";
import { computed, nextTick, ref, watch } from "vue";
import { useSubscriptionV1Store } from "@/store";
import { useSQLResultViewContext } from "../context";
import ColumnSortedIcon from "./ColumnSortedIcon.vue";
import SensitiveDataIcon from "./SensitiveDataIcon.vue";
import TableCell from "./TableCell.vue";
import useTableColumnWidthLogic from "./useTableResize";

export type DataTableColumn = {
  key: string;
  title: string;
};

const props = defineProps<{
  table: Table<string[]>;
  setIndex: number;
  offset: number;
  isSensitiveColumn: (index: number) => boolean;
  isColumnMissingSensitive: (index: number) => boolean;
}>();

const scrollerRef = ref<HTMLDivElement>();
const tableRef = ref<HTMLTableElement>();
const subscriptionStore = useSubscriptionV1Store();

const tableResize = useTableColumnWidthLogic({
  tableRef,
  scrollerRef,
  minWidth: 64, // 4rem
  maxWidth: 640, // 40rem
});

const { keyword } = useSQLResultViewContext();

const hasSensitiveFeature = computed(() => {
  return subscriptionStore.hasFeature("bb.feature.sensitive-data");
});

const scrollTo = (x: number, y: number) => {
  scrollerRef.value?.scroll(x, y);
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
      tableResize.reset();
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

watch(
  () => props.table.getState().sorting,
  () => {
    // When the sorting changed, we need to reset table size.
    tableResize.reset();
  }
);
</script>
