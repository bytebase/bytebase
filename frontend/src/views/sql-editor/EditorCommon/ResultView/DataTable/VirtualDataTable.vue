<template>
  <div
    ref="containerRef"
    class="relative w-full flex-1 overflow-auto flex flex-col rounded-sm border dark:border-zinc-500"
  >
    <div class="inline-block">
      <table
        ref="tableRef"
        class="relative border-collapse -mx-px"
        v-bind="tableResize.getTableProps()"
      >
      <thead
        class="bg-gray-50 dark:bg-gray-700 sticky top-0 z-1 drop-shadow-xs"
      >
        <tr>
          <!-- header for the index -->
          <th
            :key="`${setIndex}-0-index`"
            class="group relative py-2 shrink-0 tracking-wider border-x border-block-border dark:border-zinc-500 w-px whitespace-nowrap"
            v-bind="tableResize.getColumnProps(0)"
          >
            <!-- Use the max index to calculate the cell width -->
            <div
              :class="[
                'textinfolabel pr-1 opacity-0',
                selectionDisabled
                  ? 'pl-1'
                  : 'pl-4',
              ]"
            >{{ rows.length }}</div>
            <div
              class="absolute w-2 right-0 top-0 bottom-0 cursor-col-resize"
              @pointerdown="tableResize.startResizing(0)"
              @click.stop.prevent
            />
          </th>
          <!-- header for columns -->
          <th
            v-for="(header, columnIndex) of columns"
            :key="`${setIndex}-${columnIndex + 1}-${header.id}`"
            class="group relative px-3 py-2 min-w-8 text-left text-xs font-medium text-gray-500 dark:text-gray-300 tracking-wider border-block-border dark:border-zinc-500"
            :class="{
              'border-r': columnIndex < (columns.length - 1),
              'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800':
                !selectionDisabled,
              'bg-accent/10! dark:bg-accent/40!':
                selectionState.rows.length === 0 &&
                selectionState.columns.includes(columnIndex),
            }"
            v-bind="tableResize.getColumnProps(columnIndex + 1)"
            @click.stop="selectColumn(columnIndex)"
          >
            <div class="flex items-center overflow-hidden">
              <span class="flex flex-row items-center select-none">
                <template
                  v-if="String(header.name).length > 0"
                >
                  {{ header.name }}
                </template>
                <br v-else class="min-h-4 inline-flex" />
              </span>

              <MaskingReasonPopover
                v-if="getMaskingReason && getMaskingReason(columnIndex)"
                :reason="getMaskingReason(columnIndex)!"
                class="ml-0.5 shrink-0"
              />
              <SensitiveDataIcon
                v-else-if="isSensitiveColumn(columnIndex)"
                class="ml-0.5 shrink-0"
              />

              <ColumnSortedIcon
                :is-sorted="getColumnSortDirection(columnIndex)"
                @click.stop.prevent="handleHeaderClick(columnIndex)"
              />

              <!-- Add binary format button if this column has binary data -->
              <BinaryFormatButton
                v-if="existBinaryValue(columnIndex)"
                :format="
                  getBinaryFormat({
                    colIndex: columnIndex,
                    setIndex,
                  })
                "
                @update:format="
                  (format: BinaryFormat) =>
                    setBinaryFormat({
                      colIndex: columnIndex,
                      setIndex,
                      format,
                    })
                "
                @click.stop
              />
            </div>

            <!-- The drag-to-resize handler -->
            <div
              class="absolute w-2 right-0 top-0 bottom-0 cursor-col-resize"
              @pointerdown="tableResize.startResizing(columnIndex + 1)"
              @click.stop.prevent
            />
          </th>
        </tr>
      </thead>
      </table>
    </div>
    <NVirtualList
      ref="virtualListRef"
      :items="rows"
      :item-size="ROW_HEIGHT"
      :style="{
        minWidth: `${tableResize.effectiveWidth.value}px`,
      }"
    >
      <template #default="{ item: row, index: rowIndex }: { item: { item: QueryRow; }; index: number; }">
        <div
          :key="`${setIndex}-${rowIndex}`"
          class="flex group"
          :data-row-index="rowIndex"
          :style="{
            height: `${ROW_HEIGHT}px`,
            minWidth: `${tableResize.effectiveWidth.value}px`,
          }"
        >
          <!-- the index cell  -->
          <div
            :key="`${setIndex}-${rowIndex}-0`"
            class="relative flex items-center shrink-0 text-sm dark:text-gray-100 leading-5 whitespace-nowrap break-all border-block-border dark:border-zinc-500 group-even:bg-gray-100/50 dark:group-even:bg-gray-700/50"
            :class="{
              'border-r': true,
              'border-b': true,
              'bg-accent/10! dark:bg-accent/40!': activeRowIndex === rowIndex,
              'bg-accent/20! dark:bg-accent/40!': selectionState.rows.includes(rowIndex)
            }"
            :data-col-index="0"
            :style="{
              height: `${ROW_HEIGHT}px`,
              width: `${tableResize.getColumnWidth(0)}px`,
            }"
          >
            <NPerformantEllipsis
              :class="[
                'textinfolabel pr-1',
                selectionDisabled
                  ? 'pl-1'
                  : 'pl-4',
              ]"
            >
              {{ rowIndex + 1 }}
            </NPerformantEllipsis>

            <div
              v-if="!selectionDisabled"
              class="absolute inset-y-0 left-0 w-3 cursor-pointer bg-accent/5 dark:bg-white/10 hover:bg-accent/10 dark:hover:bg-accent/40"
              :class="{
                'bg-accent/20! dark:bg-accent/40!':
                  selectionState.columns.length === 0 &&
                  selectionState.rows.includes(
                    rowIndex
                  ),
              }"
              @click.prevent.stop="
                selectRow(rowIndex)
              "
            ></div>
          </div>

          <!-- other cells -->
          <div
            v-for="(cell, columnIndex) of row.item.values"
            :key="`${setIndex}-${rowIndex}-${columnIndex + 1}`"
            class="relative shrink-0 text-sm dark:text-gray-100 leading-5 whitespace-nowrap break-all border-block-border dark:border-zinc-500 group-even:bg-gray-100/50 dark:group-even:bg-gray-700/50"
            :class="{
              'border-r': columnIndex < (row.item.values.length - 1),
              'border-b': true,
            }"
            :data-col-index="columnIndex + 1"
            :style="{
              height: `${ROW_HEIGHT}px`,
              width: `${tableResize.getColumnWidth(columnIndex + 1)}px`,
            }"
          >
            <div
              class="h-full flex items-center overflow-hidden"
            >
              <TableCell
                :value="cell"
                :keyword="search.query"
                :scope="search.scopes.find(scope => scope.id === columns[columnIndex]?.id)"
                :set-index="setIndex"
                :row-index="rowIndex"
                :col-index="columnIndex"
                :allow-select="true"
                :column-type="getColumnTypeByIndex(columnIndex)"
                class="h-full w-full truncate"
                :database="database"
                :class="{
                  'bg-accent/10! dark:bg-accent/40!': activeRowIndex === rowIndex
                }"
              />
            </div>
          </div>
        </div>
      </template>
    </NVirtualList>
  </div>
</template>

<script lang="ts" setup>
import { useWindowSize, watchDebounced } from "@vueuse/core";
import { NPerformantEllipsis, NVirtualList } from "naive-ui";
import { nextTick, onMounted, onUnmounted, ref } from "vue";
import { type ComposedDatabase, DEBOUNCE_SEARCH_DELAY } from "@/types";
import type { MaskingReason } from "@/types/proto-es/v1/sql_service_pb";
import { type QueryRow } from "@/types/proto-es/v1/sql_service_pb";
import { type SearchParams } from "@/utils";
import BinaryFormatButton from "./common/BinaryFormatButton.vue";
import {
  type BinaryFormat,
  getBinaryFormatByColumnType,
  useBinaryFormatContext,
} from "./common/binary-format-store";
import ColumnSortedIcon from "./common/ColumnSortedIcon.vue";
import MaskingReasonPopover from "./common/MaskingReasonPopover.vue";
import SensitiveDataIcon from "./common/SensitiveDataIcon.vue";
import { useSelectionContext } from "./common/selection-logic";
import type {
  ResultTableColumn,
  ResultTableRow,
  SortDirection,
  SortState,
} from "./common/types";
import { getPlainValue } from "./common/utils";
import TableCell from "./TableCell.vue";
import useTableColumnWidthLogic from "./useTableResize";

const props = defineProps<{
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  setIndex: number;
  activeRowIndex: number;
  isSensitiveColumn: (index: number) => boolean;
  getMaskingReason?: (index: number) => MaskingReason | undefined;
  database: ComposedDatabase;
  sortState?: SortState;
  search: SearchParams;
}>();

const emit = defineEmits<{
  (event: "toggle-sort", columnIndex: number): void;
}>();

const ROW_HEIGHT = 35; // 35px height for every row
const { width: windowWidth } = useWindowSize();

const getColumnSortDirection = (columnIndex: number): SortDirection => {
  if (!props.sortState || props.sortState.columnIndex !== columnIndex) {
    return false;
  }
  return props.sortState.direction;
};

const handleHeaderClick = (columnIndex: number) => {
  emit("toggle-sort", columnIndex);
};

const {
  state: selectionState,
  disabled: selectionDisabled,
  selectColumn,
  selectRow,
} = useSelectionContext();
const containerRef = ref<HTMLDivElement>();
const tableRef = ref<HTMLTableElement>();
const virtualListRef = ref<InstanceType<typeof NVirtualList>>();

// Get row cell content for column width calculation
const getRowCellContent = (columnIndex: number): string | undefined => {
  // check at most 3 rows
  for (let i = 0; i < Math.min(3, props.rows.length); i++) {
    const firstRow = props.rows[i];
    if (!firstRow) {
      continue;
    }
    const cell = firstRow.item.values[columnIndex];
    if (!cell) {
      continue;
    }
    const columnType = props.columns[columnIndex]?.columnType ?? "";
    const plainValue = getPlainValue(cell, columnType, "DEFAULT");
    if (plainValue) {
      return plainValue;
    }
  }
  return undefined;
};

const tableResize = useTableColumnWidthLogic({
  tableRef,
  containerRef,
  minWidth: 64, // 4rem
  maxWidth: 640, // 40rem
  getRowCellContent,
});

const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();

const getColumnTypeByIndex = (columnIndex: number) => {
  return props.columns[columnIndex].columnType;
};

const existBinaryValue = (columnIndex: number) => {
  if (!getBinaryFormatByColumnType(getColumnTypeByIndex(columnIndex))) {
    return false;
  }

  // Check each row in the column for binary data (proto-es oneof pattern)
  for (const row of props.rows) {
    const cell = row.item.values[columnIndex];
    if (!cell) continue;

    if (cell.kind?.case === "bytesValue") {
      return true;
    }
  }

  return false;
};

// Re-initialize column widths when data changes
watchDebounced(
  () => props.columns,
  () => {
    nextTick(() => {
      tableResize.reset();
    });
  },
  { immediate: true, deep: true, debounce: DEBOUNCE_SEARCH_DELAY }
);

watchDebounced(
  () => windowWidth.value,
  () => {
    nextTick(() => {
      tableResize.reset();
    });
  },
  { debounce: DEBOUNCE_SEARCH_DELAY }
);

// Handle shift+wheel for horizontal scrolling
// Use capture phase to intercept before NVirtualList handles it
const handleWheel = (event: WheelEvent) => {
  const container = containerRef.value;
  if (!container) return;

  // Only handle shift+wheel when there's horizontal overflow
  const hasHorizontalOverflow = container.scrollWidth > container.clientWidth;
  if (!event.shiftKey || !hasHorizontalOverflow) return;

  event.preventDefault();
  event.stopPropagation();
  container.scrollLeft += event.deltaY || event.deltaX;
};

onMounted(() => {
  // Use capture phase to intercept before NVirtualList
  containerRef.value?.addEventListener("wheel", handleWheel, {
    passive: false,
    capture: true,
  });
});

onUnmounted(() => {
  containerRef.value?.removeEventListener("wheel", handleWheel, {
    capture: true,
  });
});

defineExpose({
  scrollTo: (index: number) =>
    virtualListRef.value?.scrollTo({
      top: Math.max(0, (index - 1) * ROW_HEIGHT),
      debounce: true,
      behavior: "smooth",
    }),
});
</script>
