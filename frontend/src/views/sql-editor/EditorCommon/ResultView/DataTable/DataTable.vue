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
      class="relative border-collapse w-full table-auto -mx-px"
      v-bind="tableResize.getTableProps()"
    >
      <thead
        class="bg-gray-50 dark:bg-gray-700 sticky top-0 z-[1] drop-shadow-sm"
      >
        <tr>
          <th
            v-for="header of table.getFlatHeaders()"
            :key="header.index"
            class="group relative px-2 py-2 min-w-[2rem] text-left text-xs font-medium text-gray-500 dark:text-gray-300 tracking-wider border-x border-block-border dark:border-zinc-500"
            :class="{
              'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800':
                !selectionDisabled,
              '!bg-accent/10 dark:!bg-accent/40':
                selectionState.rows.length === 0 &&
                selectionState.columns.includes(header.index),
              'pl-3': header.index === 0,
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

              <ColumnSortedIcon
                :is-sorted="header.column.getIsSorted()"
                @click.stop.prevent="
                  header.column.getToggleSortingHandler()?.($event)
                "
              />
              
              <!-- Add binary format button if this column has binary data -->
              <BinaryFormatButton
                v-if="isColumnWithBinaryData(header.index)"
                :column-index="header.index"
                :column-format="getColumnFormatOverride(header.index)"
                :server-format="getColumnServerFormat(header.index)"
                :has-single-bit-values="hasColumnSingleBitValues(header.index)"
                @update:format="setColumnFormat(header.index, $event)"
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
        <tr
          v-for="(row, rowIndex) of rows"
          :key="rowIndex"
          class="group"
          :data-row-index="offset + rowIndex"
        >
          <td
            v-for="(cell, cellIndex) of row.getVisibleCells()"
            :key="cellIndex"
            class="relative max-w-[50vw] p-0 text-sm dark:text-gray-100 leading-5 whitespace-nowrap break-all border-x border-b border-block-border dark:border-zinc-500 group-even:bg-gray-100/50 dark:group-even:bg-gray-700/50"
            :data-col-index="cellIndex"
          >
            <TableCell
              :table="table"
              :value="cell.getValue<RowValue>()"
              :keyword="keyword"
              :set-index="setIndex"
              :row-index="offset + rowIndex"
              :col-index="cellIndex"
              :allow-select="true"
              :column-format-override="getColumnFormatOverride(cellIndex)"
              :column-type="props.columnTypeNames?.[cellIndex]"
            />
            <div
              v-if="cellIndex === 0 && !selectionDisabled"
              class="absolute inset-y-0 left-0 w-3 cursor-pointer hover:bg-accent/10 dark:hover:bg-accent/40"
              :class="{
                'bg-accent/10 dark:bg-accent/40':
                  selectionState.columns.length === 0 &&
                  selectionState.rows.includes(offset + rowIndex),
              }"
              @click.prevent.stop="selectRow(offset + rowIndex)"
            ></div>
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
import { NEmpty } from "naive-ui";
import { computed, nextTick, onMounted, ref, watch } from "vue";
import {
  FeatureBadge,
  FeatureBadgeForInstanceLicense,
} from "@/components/FeatureGuard";
import { useSubscriptionV1Store } from "@/store";
import { type QueryRow, type RowValue } from "@/types/proto/v1/sql_service";
import { useSQLResultViewContext } from "../context";
import TableCell from "./TableCell.vue";
import ColumnSortedIcon from "./common/ColumnSortedIcon.vue";
import SensitiveDataIcon from "./common/SensitiveDataIcon.vue";
import BinaryFormatButton from "./common/BinaryFormatButton.vue";
import { useSelectionContext } from "./common/selection-logic";
import useTableColumnWidthLogic from "./useTableResize";
import { setColumnFormatOverride, setBinaryFormat, getBinaryFormat } from "./binary-format-store";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";

const props = defineProps<{
  table: Table<QueryRow>;
  setIndex: number;
  offset: number;
  isSensitiveColumn: (index: number) => boolean;
  isColumnMissingSensitive: (index: number) => boolean;
  maxHeight?: number;
  columnTypeNames?: string[]; // Column type names from QueryResult
}>();

const {
  state: selectionState,
  disabled: selectionDisabled,
  selectColumn,
  selectRow,
} = useSelectionContext();
const containerRef = ref<HTMLDivElement>();
const tableRef = ref<HTMLTableElement>();
const subscriptionStore = useSubscriptionV1Store();

const tableResize = useTableColumnWidthLogic({
  tableRef,
  containerRef,
  minWidth: 64, // 4rem
  maxWidth: 640, // 40rem
});

const { keyword } = useSQLResultViewContext();

const hasSensitiveFeature = computed(() => {
  return subscriptionStore.hasFeature("bb.feature.sensitive-data");
});

const rows = computed(() => props.table.getRowModel().rows);

// Column format overrides - map of column index to format
const columnFormatOverrides = ref<Map<number, string | null>>(new Map());

// Check if a column contains any binary data
const isColumnWithBinaryData = (columnIndex: number): boolean => {
  const columnRows = props.table.getPrePaginationRowModel().rows;
  
  // Check each row in the column for binary data
  for (const row of columnRows) {
    const cell = row.getVisibleCells()[columnIndex];
    if (!cell) continue;
    
    const value = cell.getValue<RowValue>();
    if (value?.bytesValue) {
      return true;
    }
  }
  
  return false;
};

// Determine the suitable format for a column based on column type and content
const getColumnServerFormat = (columnIndex: number): string | null => {
  // Get column type name from direct columnTypeNames prop
  let columnType = '';
  
  // Use column type names from props
  if (props.columnTypeNames && columnIndex < props.columnTypeNames.length) {
    columnType = props.columnTypeNames[columnIndex].toLowerCase();
  }
  
  // Default format based on column type (default to HEX)
  let defaultFormat = "HEX";
  
  // Detect BIT column types (bit, varbit, bit varying) - for binary format display
  const isBitColumn = (
    // Generic bit types
    columnType === 'bit' ||
    columnType.startsWith('bit(') ||
    (columnType.includes('bit') && !columnType.includes('binary')) ||
    
    // PostgreSQL bit types
    columnType === 'varbit' ||
    columnType === 'bit varying'
  );
    
  // Detect BINARY column types (binary, varbinary, bytea, blob, etc) - for hex format display
  const isBinaryColumn = (
    // Generic binary types
    columnType === 'binary' ||
    columnType.includes('binary') || 
    
    // MySQL/MariaDB binary types
    columnType.startsWith('binary(') ||
    columnType.startsWith('varbinary') ||
    columnType.includes('blob') ||
    columnType === 'longblob' ||
    columnType === 'mediumblob' ||
    columnType === 'tinyblob' ||
    
    // PostgreSQL binary type
    columnType === 'bytea' ||
    
    // SQL Server binary types
    columnType === 'image' ||
    columnType === 'varbinary(max)' ||
    
    // Oracle binary types
    columnType === 'raw' ||
    columnType === 'long raw'
  )
    
  // BIT columns default to binary format
  if (isBitColumn) {
    defaultFormat = "BINARY";
  }
  
  // BINARY/VARBINARY/BLOB columns default to HEX format
  if (isBinaryColumn) {
    defaultFormat = "HEX";
  }
  
  const columnRows = props.table.getPrePaginationRowModel().rows;
  
  // Look through rows to find byte values and determine format
  for (const row of columnRows) {
    const cell = row.getVisibleCells()[columnIndex];
    if (!cell) continue;
    
    const value = cell.getValue<RowValue>();
    if (value?.bytesValue) {
      // Get default format based on content
      // Ensure bytesValue exists before converting to array
      const byteArray = value.bytesValue ? Array.from(value.bytesValue) : [];
      
      // For single byte values (could be boolean)
      if (byteArray.length === 1 && (byteArray[0] === 0 || byteArray[0] === 1)) {
        return "BOOLEAN";
      }
      
      // Check if it's readable text
      const isReadableText = byteArray.every(byte => byte >= 32 && byte <= 126);
      if (isReadableText) {
        return "TEXT";
      }
      
      // Return default format based on column type
      return defaultFormat;
    }
  }
  
  return null;
};

// Check if a column has any single-bit values
const hasColumnSingleBitValues = (columnIndex: number): boolean => {
  const columnRows = props.table.getPrePaginationRowModel().rows;
  
  // Check if any row in this column has a single-bit value
  for (const row of columnRows) {
    const cell = row.getVisibleCells()[columnIndex];
    if (!cell) continue;
    
    const value = cell.getValue<RowValue>();
    if (value?.bytesValue?.length === 1) {
      return true;
    }
  }
  
  return false;
};

// Get the current format override for a column
const getColumnFormatOverride = (columnIndex: number): string | null => {
  return columnFormatOverrides.value.get(columnIndex) || null;
};

// Set the format for a column
const setColumnFormat = (columnIndex: number, format: string | null) => {
  if (format === null) {
    columnFormatOverrides.value.delete(columnIndex);
  } else {
    columnFormatOverrides.value.set(columnIndex, format);
  }
  
  // Store the format in the binary format store for use during copy
  const { database } = useConnectionOfCurrentSQLEditorTab();
  const databaseName = database.value?.name || '';
  setColumnFormatOverride({
    colIndex: columnIndex,
    format,
    setIndex: props.setIndex,
    databaseName
  });
  
  // Force a re-render
  columnFormatOverrides.value = new Map(columnFormatOverrides.value);
};

onMounted(() => {
  nextTick(() => {
    tableResize.reset();
    
    // Store auto-detected formats for all binary columns
    // This ensures the format is available for copy operations
    const { database } = useConnectionOfCurrentSQLEditorTab();
    const databaseName = database.value?.name || '';
    
    // For each column with binary data, store its server-detected format
    for (let colIndex = 0; colIndex < props.table.getAllColumns().length; colIndex++) {
      if (isColumnWithBinaryData(colIndex)) {
        const serverFormat = getColumnServerFormat(colIndex);
        if (serverFormat) {
          // Important: Don't set column format override during initialization
          // This would override cell-specific formats for all cells in the column
          // We only want to set it when the user explicitly chooses a format
          // setColumnFormatOverride(colIndex, serverFormat, props.setIndex, databaseName);
          
          // Only set formats for cells that don't already have a format
          const rows = props.table.getPrePaginationRowModel().rows;
          rows.forEach((row, rowIndex) => {
            const cell = row.getVisibleCells()[colIndex];
            if (cell && cell.getValue<RowValue>()?.bytesValue) {
              // Check if this cell already has a format before overriding
              const existingFormat = getBinaryFormat({
                rowIndex,
                colIndex,
                setIndex: props.setIndex,
                databaseName
              });
              if (!existingFormat) {
                // Only set if no format exists yet
                setBinaryFormat({
                  rowIndex,
                  colIndex,
                  format: serverFormat,
                  setIndex: props.setIndex,
                  databaseName
                });
              }
            }
          });
        }
      }
    }
  });
});

const scrollTo = (x: number, y: number) => {
  containerRef.value?.scroll(x, y);
};

watch(
  () => props.offset,
  () => {
    // When the offset changed, we need to reset the scroll position.
    scrollTo(0, 0);
  }
);
</script>
