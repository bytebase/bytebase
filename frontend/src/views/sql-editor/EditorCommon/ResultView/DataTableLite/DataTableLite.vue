<template>
  <div
    ref="containerElRef"
    class="relative w-full flex-1 overflow-hidden flex flex-col"
    :data-width="containerWidth"
    :data-height="containerHeight"
  >
    <div
      tag="div"
      class="header-track absolute z-0 left-0 top-0 right-0 h-[34px] border border-block-border bg-gray-50 dark:bg-gray-700"
    />

    <NDataTable
      ref="dataTableRef"
      :columns="columns"
      :data="layoutReady ? rows : []"
      :max-height="tableBodyHeight"
      :row-props="rowProps"
      :virtual-scroll="true"
      table-layout="fixed"
      size="small"
      class="relative z-[1]"
      style="
        --n-th-padding: 0;
        --n-td-padding: 0;
        --n-border-radius: 0;
        --n-border-color: rgb(var(--color-block-border));
      "
      v-bind="tableResize.getTableProps()"
    />
  </div>
</template>

<script lang="ts" setup>
import type { Header, Row, Table } from "@tanstack/vue-table";
import { useElementSize } from "@vueuse/core";
import { type DataTableColumn, NDataTable } from "naive-ui";
import { computed, h, nextTick, watch } from "vue";
import { QueryRow, type RowValue } from "@/types/proto/v1/sql_service";
import {
  nextAnimationFrame,
  useAutoHeightDataTable,
  usePreventBackAndForward,
} from "@/utils";
import { useSQLResultViewContext } from "../context";
import ColumnHeader from "./ColumnHeader.vue";
import TableCell from "./TableCell.vue";
import useTableColumnWidthLogic from "./useTableResize";

const DEFAULT_COLUMN_WIDTH = 128; // 8rem

const props = defineProps<{
  table: Table<QueryRow>;
  setIndex: number;
  offset: number;
  isSensitiveColumn: (index: number) => boolean;
  isColumnMissingSensitive: (index: number) => boolean;
}>();

const { keyword } = useSQLResultViewContext();

const headers = computed(() => {
  return props.table.getFlatHeaders() as Header<QueryRow, RowValue>[];
});
const rows = computed(() => {
  return props.table.getRowModel().rows;
});
const {
  dataTableRef,
  containerElRef,
  tableBodyHeight,
  scrollerRef,
  layoutReady,
} = useAutoHeightDataTable(rows);
const { height: containerHeight, width: containerWidth } =
  useElementSize(containerElRef);
usePreventBackAndForward(scrollerRef);

const queryTableHeaderElement = () => {
  return containerElRef.value?.querySelector(
    ".n-data-table-base-table-header table.n-data-table-table"
  ) as HTMLElement | undefined;
};
const queryTableBodyElement = () => {
  return containerElRef.value?.querySelector(
    ".n-data-table-base-table-body table.n-data-table-table"
  ) as HTMLElement | undefined;
};

const tableResize = useTableColumnWidthLogic({
  scrollerRef,
  queryTableHeaderElement,
  queryTableBodyElement,
  columnCount: computed(() => headers.value.length),
  defaultWidth: DEFAULT_COLUMN_WIDTH,
  minWidth: 64, // 4rem
  maxWidth: 640, // 40rem
});

const columns = computed(() => {
  return headers.value.map<DataTableColumn<Row<QueryRow>>>(
    (header, colIndex) => {
      return {
        key: header.id,
        title: () => {
          return h(ColumnHeader, {
            header,
            isSensitiveColumn: props.isSensitiveColumn,
            isColumnMissingSensitive: props.isColumnMissingSensitive,
            onStartResizing: () => {
              tableResize.startResizing(colIndex);
            },
          });
        },
        render: (row, rowIndex) => {
          const cell = row.getVisibleCells()[colIndex];
          const value = cell.getValue() as RowValue;
          return h(TableCell, {
            table: props.table,
            value,
            width:
              tableResize.state.columns[colIndex]?.width ??
              DEFAULT_COLUMN_WIDTH,
            keyword: keyword.value,
            setIndex: props.setIndex,
            rowIndex: props.offset + rowIndex,
            colIndex,
          });
        },
        width: tableResize.state.isAutoAdjusting
          ? undefined
          : (tableResize.state.columns[colIndex]?.width ??
            DEFAULT_COLUMN_WIDTH),
      };
    }
  );
});

const rowProps = (row: Row<QueryRow>) => {
  return {
    class: "group",
    "data-row-index": row.index,
  };
};

const scrollTo = async (x: number, y: number) => {
  await nextAnimationFrame();
  const table = dataTableRef.value;
  table?.scrollTo(x, y);
};

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
</script>

<style lang="postcss" scoped>
:deep(.n-data-table-th .n-data-table-resize-button::after) {
  @apply bg-control-bg h-2/3;
}
</style>
