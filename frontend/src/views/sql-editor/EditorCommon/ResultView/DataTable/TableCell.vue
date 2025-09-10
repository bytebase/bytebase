<!-- eslint-disable vue/no-v-html -->
<template>
  <div
    class="px-2 py-1 flex items-center"
    :class="classes"
    @click="handleClick"
    @dblclick="showDetail"
    ref="cellRef"
  >
    <div
      ref="wrapperRef"
      class="font-mono text-start break-words line-clamp-3"
      v-html="html"
    ></div>
    <div
      class="absolute right-1 top-1/2 translate-y-[-45%] flex items-center gap-1"
    >
      <!-- Format button for binary data -->
      <BinaryFormatButton
        v-if="hasByteData"
        :format="binaryFormat"
        @update:format="
          (format: BinaryFormat) =>
            setBinaryFormat({
              colIndex,
              rowIndex,
              setIndex,
              format,
            })
        "
        @click.stop
      />

      <!-- Expand button for long content -->
      <NButton
        v-if="clickable"
        size="tiny"
        circle
        class="dark:!bg-dark-bg"
        @click.stop="showDetail"
      >
        <template #icon>
          <heroicons:arrows-pointing-out class="w-3 h-3" />
        </template>
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { type Table } from "@tanstack/vue-table";
import { useResizeObserver } from "@vueuse/core";
import { escape } from "lodash-es";
import { NButton } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed, ref, watchEffect } from "vue";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { QueryRow, RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { extractSQLRowValuePlain, getHighlightHTMLByRegExp } from "@/utils";
import { useSQLResultViewContext } from "../context";
import {
  useBinaryFormatContext,
  formatBinaryValue,
  detectBinaryFormat,
  type BinaryFormat,
} from "./binary-format-store";
import BinaryFormatButton from "./common/BinaryFormatButton.vue";
import { useSelectionContext } from "./common/selection-logic";

const props = defineProps<{
  table: Table<QueryRow>;
  value: RowValue;
  setIndex: number;
  rowIndex: number;
  colIndex: number;
  allowSelect?: boolean;
  columnType: string; // Column type from QueryResult
}>();

const { detail, keyword } = useSQLResultViewContext();
const { database } = useConnectionOfCurrentSQLEditorTab();
const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();

const {
  state: selectionState,
  disabled: selectionDisabled,
  selectCell,
  selectRow,
} = useSelectionContext();
const wrapperRef = ref<HTMLDivElement>();
const cellRef = ref<HTMLDivElement>();
const truncated = ref(false);

const allowSelect = computed(() => {
  return props.allowSelect && !selectionDisabled.value;
});

// Check if the value is binary data (proto-es oneof pattern)
const hasByteData = computed(() => {
  return props.value.kind?.case === "bytesValue";
});

const binaryFormat = computed(() => {
  return getBinaryFormat({
    rowIndex: props.rowIndex,
    colIndex: props.colIndex,
    setIndex: props.setIndex,
  });
});

watchEffect(() => {
  if (!hasByteData.value) {
    return;
  }
  if (!binaryFormat.value) {
    const bytesValue =
      props.value.kind?.case === "bytesValue"
        ? props.value.kind.value
        : undefined;
    if (bytesValue) {
      const binaryFormat = detectBinaryFormat({
        bytesValue,
        columnType: props.columnType,
      });
      setBinaryFormat({
        rowIndex: props.rowIndex,
        colIndex: props.colIndex,
        setIndex: props.setIndex,
        format: binaryFormat,
      });
    }
  }
});

useResizeObserver(wrapperRef, (entries) => {
  const div = entries[0].target as HTMLDivElement;
  const contentHeight = div.scrollHeight;
  const visibleHeight = div.offsetHeight;
  // Check if content is truncated vertically (due to line-clamp)
  if (contentHeight > visibleHeight + 2) {
    // +2 for minor pixel differences
    truncated.value = true;
  } else {
    truncated.value = false;
  }
});

const clickable = computed(() => {
  if (truncated.value) return true;
  if (database.value.instanceResource.engine === Engine.MONGODB) {
    // A cheap way to check JSON string without paying the parsing cost.
    const maybeJSON = String(props.value).trim();
    return (
      (maybeJSON.startsWith("{") && maybeJSON.endsWith("}")) ||
      (maybeJSON.startsWith("[") && maybeJSON.endsWith("]"))
    );
  }
  return false;
});

const classes = computed(() => {
  const classes: string[] = [];
  if (allowSelect.value) {
    classes.push("cursor-pointer");
    classes.push("hover:bg-white/20 dark:hover:bg-black/5");
    if (props.colIndex === 0) {
      classes.push("pl-3");
    }
    if (
      selectionState.value.columns.length === 1 &&
      selectionState.value.rows.length === 1
    ) {
      if (
        selectionState.value.columns[0] === props.colIndex &&
        selectionState.value.rows[0] === props.rowIndex
      ) {
        classes.push("!bg-accent/10 dark:!bg-accent/40");
      }
    } else if (
      selectionState.value.columns.includes(props.colIndex) ||
      selectionState.value.rows.includes(props.rowIndex)
    ) {
      classes.push("!bg-accent/10 dark:!bg-accent/40");
    } else {
    }
  } else {
    classes.push("select-none");
  }
  return twMerge(classes);
});

// Format the binary value based on selected format (proto-es oneof pattern)
const formattedValue = computed(() => {
  const bytesValue =
    props.value.kind?.case === "bytesValue"
      ? props.value.kind.value
      : undefined;
  if (!bytesValue) {
    return props.value;
  }

  // Determine the format to use - column override, cell override, or auto-detected format
  let actualFormat = binaryFormat.value ?? "DEFAULT";

  // If format is DEFAULT, use the auto-detected format
  if (actualFormat === "DEFAULT") {
    actualFormat = detectBinaryFormat({
      bytesValue,
      columnType: props.columnType,
    });
  }

  const stringValue = formatBinaryValue({
    bytesValue,
    format: actualFormat,
  });

  // Return proto-es oneof structure with stringValue
  return {
    ...props.value,
    kind: {
      case: "stringValue" as const,
      value: stringValue,
    },
  };
});

const plainValue = computed(() => {
  const value = extractSQLRowValuePlain(formattedValue.value);
  if (value === undefined || value === null) {
    return value;
  }
  return String(value);
});

const html = computed(() => {
  // Extract the display value
  const value = plainValue.value;

  if (value === undefined) {
    return `<span class="text-gray-400 italic">UNSET</span>`;
  }
  if (value === null) {
    return `<span class="text-gray-400 italic">NULL</span>`;
  }
  if (value.length === 0) {
    return `<br style="min-width: 1rem; display: inline-flex;" />`;
  }

  const kw = keyword.value.trim();
  if (!kw) {
    return escape(value);
  }

  return getHighlightHTMLByRegExp(
    escape(value),
    escape(kw),
    false /* !caseSensitive */
  );
});

const handleClick = (e: MouseEvent) => {
  if (!allowSelect.value) {
    return;
  }
  // If the user is holding the ctrl/cmd key, select the row.
  if (e.ctrlKey || e.metaKey) {
    selectRow(props.rowIndex);
  } else {
    selectCell(props.rowIndex, props.colIndex);
  }
  e.stopPropagation();
};

const showDetail = () => {
  detail.value = {
    set: props.setIndex,
    row: props.rowIndex,
    col: props.colIndex,
    table: props.table,
  };
};

defineExpose({
  plainValue,
});
</script>
