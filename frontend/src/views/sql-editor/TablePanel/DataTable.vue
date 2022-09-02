<template>
  <div
    class="bb-data-table relative w-full h-full overflow-hidden flex flex-col"
  >
    <div ref="wrapperRef" class="w-full flex-1 overflow-hidden">
      <div
        class="inner-wrapper max-h-full overflow-auto border-y border-block-border"
        :class="data.length === 0 && 'border-b-0'"
      >
        <table class="border-collapse min-w-full">
          <thead class="sticky top-0 z-[1] shadow">
            <tr>
              <th
                v-for="header of table.getFlatHeaders()"
                :key="header.index"
                class="relative px-2 py-2 min-w-[2rem] truncate text-left bg-gray-50 text-xs font-medium text-gray-500 tracking-wider border border-t-0 border-block-border border-b-0"
              >
                {{ header.column.columnDef.header }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(row, rowIndex) of table.getRowModel().rows"
              :key="rowIndex"
            >
              <td
                v-for="(cell, cellIndex) of row.getVisibleCells()"
                :key="cellIndex"
                class="px-2 py-2 text-sm leading-5 whitespace-pre-wrap border border-block-border"
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

const wrapperRef = ref<HTMLDivElement>();

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
  wrapperRef.value?.scrollTo(0, 0);
};

watch(data, () => {
  nextTick(() => {
    wrapperRef.value?.scrollTo(0, 0);
  });
});
</script>

<style>
.bb-data-table tbody tr:first-child td {
  @apply border-t-0;
}
.bb-data-table tbody tr:last-child td {
  @apply border-b-0;
}
</style>
