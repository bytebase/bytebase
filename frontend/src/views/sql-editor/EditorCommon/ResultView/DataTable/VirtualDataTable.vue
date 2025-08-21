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
      class="relative border-collapse w-full -mx-px table-fixed"
      v-bind="tableResize.getTableProps()"
    >
      <thead
        class="bg-gray-50 dark:bg-gray-700 sticky top-0 z-[1] drop-shadow-sm"
      >
        <tr>
          <th
            v-for="header of columns"
            :key="`${setIndex}-${header.id}`"
            class="group relative px-2 py-2 min-w-[2rem] text-left text-xs font-medium text-gray-500 dark:text-gray-300 tracking-wider border-x border-block-border dark:border-zinc-500"
            :class="{
              'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800':
                !selectionDisabled,
              '!bg-accent/10 dark:!bg-accent/40':
                selectionState.rows.length === 0 &&
                selectionState.columns.includes(header.index),
              'pl-6': header.index === 0 && !selectionDisabled,
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

              <MaskingReasonPopover
                v-if="getMaskingReason && getMaskingReason(header.index)"
                :reason="getMaskingReason(header.index)"
                class="ml-0.5 shrink-0"
              />
              <SensitiveDataIcon
                v-else-if="isSensitiveColumn(header.index)"
                class="ml-0.5 shrink-0"
              />

              <ColumnSortedIcon
                :is-sorted="header.column.getIsSorted()"
                @click.stop.prevent="
                  header.column.getToggleSortingHandler()?.($event)
                "
              />

              <!-- Add binary format button if this column has binary data -->
              <BinaryFormatButton
                v-if="existBinaryValue(header.index)"
                :format="
                  getBinaryFormat({
                    colIndex: header.index,
                    setIndex,
                  })
                "
                @update:format="
                  (format: BinaryFormat) =>
                    setBinaryFormat({
                      colIndex: header.index,
                      setIndex,
                      format,
                    })
                "
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
        <!-- Virtual scrolling container -->
        <tr>
          <td :colspan="columns.length" class="!p-0 !border-0">
            <div
              ref="scrollerRef"
              class="relative w-full"
              :style="{
                height: `${virtualizer.getTotalSize()}px`,
              }"
            >
              <div
                class="absolute top-0 left-0 w-full"
                :style="{
                  transform: `translateY(${virtualItems[0]?.start ?? 0}px)`,
                }"
              >
                <table
                  class="w-full border-collapse table-fixed"
                  :style="{ tableLayout: 'fixed' }"
                >
                  <colgroup>
                    <col
                      v-for="(_, index) in columns"
                      :key="`col-${index}`"
                      v-bind="tableResize.getColumnProps(index)"
                    />
                  </colgroup>
                  <tbody>
                    <tr
                      v-for="virtualRow in virtualItems"
                      :key="`${setIndex}-${virtualRow.index}`"
                      class="group"
                      :data-row-index="offset + virtualRow.index"
                      :style="{
                        height: `${virtualRow.size}px`,
                      }"
                    >
                      <td
                        v-for="(cell, columnIndex) of rows[
                          virtualRow.index
                        ].getVisibleCells()"
                        :key="`${setIndex}-${virtualRow.index}-${columnIndex}`"
                        class="relative max-w-[50vw] !p-0 text-sm dark:text-gray-100 leading-5 whitespace-nowrap break-all border-x border-b border-block-border dark:border-zinc-500 group-even:bg-gray-100/50 dark:group-even:bg-gray-700/50"
                        :data-col-index="columnIndex"
                        :style="{
                          height: `${virtualRow.size}px`,
                        }"
                      >
                        <div
                          :class="[
                            'h-full flex items-center',
                            columnIndex === 0 && !selectionDisabled
                              ? 'ml-3'
                              : '',
                          ]"
                        >
                          <TableCell
                            :table="table"
                            :value="cell.getValue<RowValue>()"
                            :keyword="keyword"
                            :set-index="setIndex"
                            :row-index="offset + virtualRow.index"
                            :col-index="columnIndex"
                            :allow-select="true"
                            :column-type="getColumnTypeByIndex(columnIndex)"
                            :class="{
                              // 'ml-3': columnIndex === 0 && !selectionDisabled,
                              'h-full': true,
                              'w-full': true,
                            }"
                          />
                        </div>
                        <div
                          v-if="columnIndex === 0 && !selectionDisabled"
                          class="absolute inset-y-0 left-0 w-3 cursor-pointer bg-accent/5 dark:bg-white/10 hover:bg-accent/10 dark:hover:bg-accent/40"
                          :class="{
                            '!bg-accent/10 dark:!bg-accent/40':
                              selectionState.columns.length === 0 &&
                              selectionState.rows.includes(
                                offset + virtualRow.index
                              ),
                          }"
                          @click.prevent.stop="
                            selectRow(offset + virtualRow.index)
                          "
                        ></div>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
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
import { useVirtualizer } from "@tanstack/vue-virtual";
import { NEmpty } from "naive-ui";
import { computed, nextTick, onMounted, ref, watch } from "vue";
import {
  type QueryRow,
  type RowValue,
} from "@/types/proto-es/v1/sql_service_pb";
import { useSQLResultViewContext } from "../context";
import TableCell from "./TableCell.vue";
import {
  useBinaryFormatContext,
  getBinaryFormatByColumnType,
  type BinaryFormat,
} from "./binary-format-store";
import BinaryFormatButton from "./common/BinaryFormatButton.vue";
import ColumnSortedIcon from "./common/ColumnSortedIcon.vue";
import MaskingReasonPopover from "./common/MaskingReasonPopover.vue";
import SensitiveDataIcon from "./common/SensitiveDataIcon.vue";
import { useSelectionContext } from "./common/selection-logic";
import { getColumnType } from "./common/utils";
import useTableColumnWidthLogic from "./useTableResize";

const props = defineProps<{
  table: Table<QueryRow>;
  setIndex: number;
  offset: number;
  isSensitiveColumn: (index: number) => boolean;
  getMaskingReason?: (index: number) => any;
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
const scrollerRef = ref<HTMLDivElement>();

const tableResize = useTableColumnWidthLogic({
  tableRef,
  containerRef,
  minWidth: 64, // 4rem
  maxWidth: 640, // 40rem
});

const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();

const { keyword } = useSQLResultViewContext();

const rows = computed(() => props.table.getRowModel().rows);
const columns = computed(() => props.table.getFlatHeaders());

const getColumnTypeByIndex = (columnIndex: number) => {
  return getColumnType(columns.value[columnIndex]);
};

const existBinaryValue = (columnIndex: number) => {
  if (!getBinaryFormatByColumnType(getColumnTypeByIndex(columnIndex))) {
    return false;
  }

  // Check each row in the column for binary data (proto-es oneof pattern)
  for (const row of rows.value) {
    const cell = row.getVisibleCells()[columnIndex];
    if (!cell) continue;

    const value = cell.getValue<RowValue>();
    if (value?.kind?.case === "bytesValue") {
      return true;
    }
  }

  return false;
};

// Virtual scrolling setup
const virtualizer = useVirtualizer(
  computed(() => ({
    count: rows.value.length,
    getScrollElement: () => containerRef.value ?? null,
    estimateSize: () => 35, // Estimated row height
    overscan: 10, // Number of items to render outside visible area
  }))
);

const virtualItems = computed(() => virtualizer.value.getVirtualItems());

onMounted(() => {
  nextTick(() => {
    tableResize.reset();
  });
});

// Re-initialize column widths when data changes
watch(
  () => [rows.value.length, columns.value.length],
  () => {
    nextTick(() => {
      tableResize.reset();
    });
  }
);

const scrollTo = (x: number, y: number) => {
  containerRef.value?.scroll(x, y);
};

watch(
  () => props.offset,
  () => {
    // When the offset changed, we need to reset the scroll position.
    scrollTo(0, 0);
    virtualizer.value.scrollToOffset(0);
  }
);
</script>
