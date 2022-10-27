<template>
  <div
    class="bb-data-table relative w-full h-full overflow-hidden flex flex-col"
  >
    <div class="w-full flex-1 overflow-hidden">
      <div
        class="header-track absolute z-0 left-0 top-0 right-0 h-[34px] border border-block-border bg-gray-50"
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
                class="relative px-2 py-2 min-w-[2rem] text-left bg-gray-50 text-xs font-medium text-gray-500 tracking-wider border border-t-0 border-block-border border-b-0"
                v-bind="tableResize.getColumnProps(header.index)"
              >
                {{ header.column.columnDef.header }}

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
            >
              <td
                v-for="(cell, cellIndex) of row.getVisibleCells()"
                :key="cellIndex"
                class="px-2 py-1 text-sm leading-5 whitespace-pre-wrap break-all border border-block-border group-last:border-b-0 group-even:bg-gray-50/50"
              >
                {{ cell.getValue() }}
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

    <div v-if="showPagination" class="flex justify-end pt-2">
      <NPagination
        :item-count="table.getCoreRowModel().rows.length"
        :page="table.getState().pagination.pageIndex + 1"
        :page-size="table.getState().pagination.pageSize"
        :show-quick-jumper="true"
        :show-size-picker="true"
        :page-sizes="[20, 50, 100]"
        @update-page="handleChangePage"
        @update-page-size="(ps) => table.setPageSize(ps)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, PropType, ref, watch } from "vue";
import {
  getCoreRowModel,
  useVueTable,
  getPaginationRowModel,
  ColumnDef,
} from "@tanstack/vue-table";
import { NPagination } from "naive-ui";
import useTableColumnWidthLogic from "./useTableResize";

export type DataTableColumn = {
  key: string;
  title: string;
};

const PAGE_SIZES = [20, 50, 100];
const DEFAULT_PAGE_SIZE = 50;

const props = defineProps({
  data: {
    type: Array as PropType<string[][]>,
    default: () => [],
  },
  columns: {
    type: Array as PropType<DataTableColumn[]>,
    default: () => [],
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
const columns = computed(() =>
  props.columns.map<ColumnDef<string[]>>((col, index) => ({
    // accessorKey: col.key,
    accessorFn: (item) => item[index],
    header: col.title,
  }))
);

const table = useVueTable<string[]>({
  get data() {
    return data.value;
  },
  get columns() {
    return columns.value;
  },
  getCoreRowModel: getCoreRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
});

table.setPageSize(DEFAULT_PAGE_SIZE);
const showPagination = computed(() => props.data.length > PAGE_SIZES[0]);

const handleChangePage = (page: number) => {
  table.setPageIndex(page - 1);
  scrollerRef.value?.scrollTo(0, 0);
};

watch(
  () => props.columns.map((col) => col.title).join("|"),
  () => {
    nextTick(() => {
      // Re-calculate the column widths once the column definition changed.
      scrollerRef.value?.scrollTo(0, 0);
      tableResize.reset();
    });
  },
  { immediate: true }
);
</script>
