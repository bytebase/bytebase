<template>
  <NVirtualList
    ref="virtualListRef"
    item-resizable
    :items="rows"
    :item-size="estimateRowHeight"
  >
    <template #default="{ item: row, index: rowIndex }: { item: { item: QueryRow; }; index: number; }">
      <div
        :key="`row-${rowIndex}`"
        class="font-mono mb-2 mx-2"
        :style="{
          height: `${estimateRowHeight}px`,
        }"
      >
        <p
          class="font-bold text-gray-500 dark:text-gray-300 overflow-hidden whitespace-nowrap mb-1"
        >
          ********************************
          {{ rowIndex + 1 }}. row
          ********************************
        </p>
        <div
          class="py-2 px-3 bg-gray-50 dark:bg-gray-700 rounded relative"
          :class="{
            'border-2 border-accent/20 bg-accent/10!': activeRowIndex === rowIndex
          }"
        >
          <div
            class="absolute right-2 top-2 z-50 opacity-70 hover:opacity-100"
          >
            <CopyButton size="small" :content="() => getContent(rowIndex)" />
          </div>
          <div
            v-for="(column, columnIndex) in columns"
            :key="column.id"
            class="flex items-start text-gray-500 dark:text-gray-300 text-sm"
          >
            <div class="min-w-28 text-left flex items-start font-medium pt-1">
              {{ column.name }}
              <MaskingReasonPopover
                v-if="getMaskingReason && getMaskingReason(columnIndex)"
                :reason="getMaskingReason(columnIndex)!"
                class="ml-0.5 shrink-0"
              />
              <SensitiveDataIcon
                v-else-if="isSensitiveColumn(columnIndex)"
                class="ml-0.5 shrink-0"
              />
              :
            </div>
            <div class="flex-1">
              <TableCell
                :value="row.item.values[columnIndex]"
                :keyword="search.query"
                :scope="search.scopes.find(scope => scope.id === columns[columnIndex]?.id)"
                :set-index="setIndex"
                :row-index="rowIndex"
                :col-index="columnIndex"
                :column-type="column.columnType"
                :allow-select="true"
                :database="database"
              />
            </div>
          </div>
        </div>
      </div>
    </template>
  </NVirtualList>
</template>

<script setup lang="ts">
import { NVirtualList } from "naive-ui";
import { computed, ref } from "vue";
import { CopyButton } from "@/components/v2";
import { type ComposedDatabase } from "@/types";
import type {
  MaskingReason,
  QueryRow,
} from "@/types/proto-es/v1/sql_service_pb";
import { type SearchParams } from "@/utils";
import { useBinaryFormatContext } from "./DataTable/common/binary-format-store";
import MaskingReasonPopover from "./DataTable/common/MaskingReasonPopover.vue";
import SensitiveDataIcon from "./DataTable/common/SensitiveDataIcon.vue";
import type {
  ResultTableColumn,
  ResultTableRow,
} from "./DataTable/common/types";
import { getPlainValue } from "./DataTable/common/utils";
import TableCell from "./DataTable/TableCell.vue";

const props = defineProps<{
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  setIndex: number;
  activeRowIndex: number;
  isSensitiveColumn: (index: number) => boolean;
  getMaskingReason?: (index: number) => MaskingReason | undefined;
  database: ComposedDatabase;
  search: SearchParams;
}>();

const { getBinaryFormat } = useBinaryFormatContext();
const virtualListRef = ref<InstanceType<typeof NVirtualList>>();

// Calculate the height of each row block
const estimateRowHeight = computed(() => {
  // Estimate based on number of columns
  // Each column is roughly 28px, plus padding and header
  const columnCount = props.columns.length;
  return 60 + columnCount * 28; // Header + columns
});

const getContent = (rowIndex: number): string => {
  const object = props.columns.reduce(
    (obj, column, columnIndex) => {
      if (!column.name) {
        return obj;
      }
      const binaryFormat = getBinaryFormat({
        rowIndex: rowIndex,
        colIndex: columnIndex,
        setIndex: props.setIndex,
      });
      obj[`${column.name}`] = getPlainValue(
        props.rows[rowIndex].item.values[columnIndex],
        column.columnType,
        binaryFormat
      );
      return obj;
    },
    {} as Record<string, unknown>
  );

  return JSON.stringify(object, null, 4);
};

defineExpose({
  scrollTo: (index: number) =>
    virtualListRef.value?.scrollTo({
      top: index * estimateRowHeight.value,
      debounce: true,
      behavior: "smooth",
    }),
});
</script>
