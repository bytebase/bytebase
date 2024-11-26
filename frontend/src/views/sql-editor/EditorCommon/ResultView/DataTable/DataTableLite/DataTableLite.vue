<template>
  <div
    ref="containerRef"
    class="relative w-full flex-1 overflow-hidden flex flex-col"
    :data-width="containerWidth"
    :data-height="containerHeight"
  >
    <div
      class="header-track absolute z-[1] left-0 top-0 right-0 h-[34px] border border-block-border bg-gray-50 dark:bg-gray-700"
    />
    <div
      class="absolute z-[0] left-0 right-0 border-b border-r border-block-border"
      :style="padStyle"
      :data-container-width="containerWidth"
      :data-container-height="containerHeight"
      :data-table-body-height="tableBodyHeight"
      :data-rendered-table-body-height="renderedTableBodyHeight"
    />

    <NDataTable
      ref="dataTableRef"
      :columns="columns"
      :data="rows"
      :row-props="rowProps"
      :virtual-scroll="true"
      :virtual-scroll-x="true"
      :virtual-scroll-header="true"
      :header-height="HEADER_HEIGHT"
      :max-height="tableBodyHeight"
      :min-row-height="ROW_HEIGHT"
      :height-for-row="() => ROW_HEIGHT"
      :scroll-x="tableResize.getTableScrollWidth()"
      :scrollbar-props="{
        trigger: 'none',
      }"
      table-layout="fixed"
      size="small"
      class="relative z-[1] -mr-px"
      style="
        --n-th-padding: 0;
        --n-td-padding: 0;
        --n-border-radius: 0;
        --n-border-color: rgb(var(--color-block-border));
      "
      :style="{
        width: `${tableResize.getTableRenderWidth()}px`,
      }"
    />
  </div>
</template>

<script lang="ts" setup>
import type { Header, Row, Table } from "@tanstack/vue-table";
import { pausableWatch, useElementSize } from "@vueuse/core";
import { type DataTableColumn, type DataTableInst, NDataTable } from "naive-ui";
import { computed, h, nextTick, ref, toRef, watch, type StyleValue } from "vue";
import { QueryRow, type RowValue } from "@/types/proto/v1/sql_service";
import { nextAnimationFrame, usePreventBackAndForward } from "@/utils";
import { useSQLResultViewContext } from "../../context";
import ColumnHeader from "./ColumnHeader.vue";
import TableCell from "./TableCell.vue";
import useTableColumnWidthLogic from "./useTableResize";

const COLUMN_WIDTH = {
  DEFAULT: 128, // 8rem
  MIN: 64, // 4rem
  MAX: 640, // 40rem
};
const HEADER_HEIGHT = 33;
const GAP_AND_BORDER_HEIGHT = 2;
const ROW_HEIGHT = 28;

const props = defineProps<{
  table: Table<QueryRow>;
  setIndex: number;
  offset: number;
  isSensitiveColumn: (index: number) => boolean;
  isColumnMissingSensitive: (index: number) => boolean;
  maxHeight?: number;
}>();

const { keyword } = useSQLResultViewContext();

const headers = computed(() => {
  return props.table.getFlatHeaders() as Header<QueryRow, RowValue>[];
});
const rows = computed(() => {
  return props.table.getRowModel().rows;
});
const dataTableRef = ref<DataTableInst>();
const containerRef = ref<HTMLElement>();
const scrollerRef = computed(() => {
  const getter = (dataTableRef.value as any)?.$refs.mainTableInstRef.$refs
    .bodyInstRef.virtualListContainer;
  if (typeof getter === "function") {
    return getter() as HTMLElement | undefined;
  }
  return undefined;
});
const { height: containerHeight, width: containerWidth } =
  useElementSize(containerRef);
usePreventBackAndForward(scrollerRef);

const tableBodyHeight = ref(0);
const { pause: pauseWatch, resume: resumeWatch } = pausableWatch(
  [() => props.table.getRowCount(), containerHeight],
  async () => {
    if (containerHeight.value === 0) return;
    if (!props.maxHeight) {
      // The table height is always limited by the container height
      // and need not to be adjusted automatically
      const experimentalMaxBodyHeight =
        containerHeight.value - HEADER_HEIGHT - GAP_AND_BORDER_HEIGHT;

      tableBodyHeight.value = Math.max(0, experimentalMaxBodyHeight);
      return;
    }

    pauseWatch();
    const experimentalMaxBodyHeight =
      props.maxHeight - HEADER_HEIGHT - GAP_AND_BORDER_HEIGHT;
    tableBodyHeight.value = Math.max(0, experimentalMaxBodyHeight);
    await nextAnimationFrame();
    resumeWatch();
  },
  { immediate: true }
);

const queryTableHeaderElement = () => {
  return containerRef.value?.querySelector(
    ".n-data-table-base-table-header table.n-data-table-table"
  ) as HTMLElement | undefined;
};
const queryTableBodyElement = () => {
  return containerRef.value?.querySelector(
    ".n-data-table-base-table-body table.n-data-table-table"
  ) as HTMLElement | undefined;
};

const tableResize = useTableColumnWidthLogic({
  table: toRef(props, "table"),
  containerWidth,
  scrollerRef,
  queryTableHeaderElement,
  queryTableBodyElement,
  columnCount: computed(() => headers.value.length),
  defaultWidth: COLUMN_WIDTH.DEFAULT,
  minWidth: COLUMN_WIDTH.MIN,
  maxWidth: COLUMN_WIDTH.MAX,
});

const tableBodyElemRef = computed(() => {
  return queryTableBodyElement();
});
const { height: renderedTableBodyHeight } = useElementSize(tableBodyElemRef);
const padStyle = computed(() => {
  const style: StyleValue = {
    top: `${HEADER_HEIGHT}px`,
  };
  const height = Math.min(renderedTableBodyHeight.value, tableBodyHeight.value);
  style.height = `${height + GAP_AND_BORDER_HEIGHT}px`;
  return style;
});

const columns = computed(() => {
  return headers.value.map<DataTableColumn<Row<QueryRow>>>(
    (header, colIndex) => {
      return {
        key: colIndex,
        cellProps: (row) => {
          return {
            class: "truncate",
            "data-row-index": row.index,
            "data-col-index": colIndex,
          };
        },
        title: () => {
          return h(ColumnHeader, {
            header,
            isSensitiveColumn: props.isSensitiveColumn,
            isColumnMissingSensitive: props.isColumnMissingSensitive,
            onStartResizing: () => {
              tableResize.startResizing(colIndex);
            },
            onAutoResize: () => {
              tableResize.autoAdjustColumnWidth([colIndex]);
            },
          });
        },
        render: (row, rowIndex) => {
          const cell = row.getVisibleCells()[colIndex];
          const value = cell.getValue() as RowValue;
          return h(TableCell, {
            table: props.table,
            value,
            width: tableResize.getColumnWidth(colIndex),
            keyword: keyword.value,
            setIndex: props.setIndex,
            rowIndex: props.offset + rowIndex,
            colIndex,
            class: row.index % 2 === 1 && "bg-gray-100/50 dark:bg-gray-700/50",
          });
        },
        width: tableResize.state.autoAdjusting.has(colIndex)
          ? COLUMN_WIDTH.DEFAULT
          : tableResize.getColumnWidth(colIndex),
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
:deep(.n-data-table-base-table-body) {
  height: auto;
}
:deep(.n-data-table-th .n-data-table-resize-button::after) {
  @apply bg-control-bg h-2/3;
}
:deep(.n-data-table-th:not(.n-data-table-th--last)) {
  border-right: 1px solid var(--n-merged-border-color);
}
:deep(.n-data-table-td:not(.n-data-table-td--last-col)) {
  border-right: 1px solid var(--n-merged-border-color);
}
</style>
