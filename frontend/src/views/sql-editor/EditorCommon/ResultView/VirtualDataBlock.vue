<template>
  <div ref="containerRef" class="h-full overflow-auto">
    <div
      class="relative"
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
        <div
          v-for="virtualRow in virtualItems"
          :key="`row-${offset + virtualRow.index}`"
          class="font-mono mb-5 mx-2"
          :style="{
            height: `${virtualRow.size}px`,
          }"
        >
          <p
            class="font-medium text-gray-500 dark:text-gray-300 overflow-hidden whitespace-nowrap"
          >
            ********************************
            {{ offset + virtualRow.index + 1 }}. row
            ********************************
          </p>
          <div class="py-2 px-3 bg-gray-50 dark:bg-gray-700 rounded-sm">
            <div
              v-for="header in columnHeaders"
              :key="`${virtualRow.index}-${header.index}`"
              class="flex items-center text-gray-500 dark:text-gray-300 text-sm"
            >
              <div class="min-w-[7rem] text-left flex items-center font-medium">
                {{ header.column.columnDef.header }}
                <MaskingReasonPopover
                  v-if="getMaskingReason && getMaskingReason(header.index)"
                  :reason="getMaskingReason(header.index)"
                  class="ml-0.5 shrink-0"
                />
                <SensitiveDataIcon
                  v-else-if="isSensitiveColumn(header.index)"
                  class="ml-0.5 shrink-0"
                />
                :
              </div>
              <div class="flex-1">
                <TableCell
                  :table="table"
                  :value="
                    rows[virtualRow.index]
                      .getVisibleCells()
                      [header.index].getValue() as RowValue
                  "
                  :keyword="keyword"
                  :set-index="setIndex"
                  :row-index="offset + virtualRow.index"
                  :col-index="header.index"
                  :column-type="getColumnType(header)"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div
      v-if="rows.length === 0"
      class="text-center w-full my-12 textinfolabel"
    >
      {{ $t("sql-editor.no-data-available") }}
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Table } from "@tanstack/vue-table";
import { useVirtualizer } from "@tanstack/vue-virtual";
import { computed, ref, watch } from "vue";
import type { QueryRow, RowValue } from "@/types/proto-es/v1/sql_service_pb";
import TableCell from "./DataTable/TableCell.vue";
import MaskingReasonPopover from "./DataTable/common/MaskingReasonPopover.vue";
import SensitiveDataIcon from "./DataTable/common/SensitiveDataIcon.vue";
import { getColumnType } from "./DataTable/common/utils";
import { useSQLResultViewContext } from "./context";

const props = defineProps<{
  table: Table<QueryRow>;
  setIndex: number;
  offset: number;
  isSensitiveColumn: (index: number) => boolean;
  getMaskingReason?: (index: number) => any;
}>();

const { keyword } = useSQLResultViewContext();
const containerRef = ref<HTMLDivElement>();

const rows = computed(() => props.table.getRowModel().rows);

// Cache column headers to avoid recalculation
const columnHeaders = computed(() => props.table.getFlatHeaders());

// Calculate the height of each row block
const estimateRowHeight = () => {
  // Estimate based on number of columns
  // Each column is roughly 28px, plus padding and header
  const columnCount = columnHeaders.value.length;
  return 60 + columnCount * 28; // Header + columns
};

// Virtual scrolling setup
const virtualizer = useVirtualizer(
  computed(() => ({
    count: rows.value.length,
    getScrollElement: () => containerRef.value ?? null,
    estimateSize: estimateRowHeight,
    overscan: 3, // Fewer items in overscan since each item is larger
  }))
);

const virtualItems = computed(() => virtualizer.value.getVirtualItems());

// Reset scroll position when offset changes
watch(
  () => props.offset,
  () => {
    containerRef.value?.scrollTo({ top: 0 });
    virtualizer.value.scrollToOffset(0);
  }
);
</script>
