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
      class="font-mono text-start wrap-break-word line-clamp-3"
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
        class="shadow bg-white! dark:bg-dark-bg!"
        :class="{
          'opacity-90': clickable,
          'hover:opacity-100': clickable,
        }"
        @click.stop="showDetail"
      >
        <template #icon>
          <ExpandIcon class="w-3 h-3" />
        </template>
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useResizeObserver } from "@vueuse/core";
import { escape } from "lodash-es";
import { ExpandIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed, ref } from "vue";
import { type ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { getHighlightHTMLByRegExp, type SearchScope } from "@/utils";
import { useSQLResultViewContext } from "../context";
import BinaryFormatButton from "./common/BinaryFormatButton.vue";
import {
  type BinaryFormat,
  useBinaryFormatContext,
} from "./common/binary-format-store";
import { useSelectionContext } from "./common/selection-logic";
import { getPlainValue } from "./common/utils";

const props = defineProps<{
  value: RowValue;
  setIndex: number;
  rowIndex: number;
  colIndex: number;
  allowSelect?: boolean;
  columnType: string; // Column type from QueryResult
  database: ComposedDatabase;
  scope?: SearchScope;
  keyword: string;
}>();

const { detail } = useSQLResultViewContext();
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

useResizeObserver(cellRef, () => {
  const cell = cellRef.value;
  const wrapper = wrapperRef.value;
  if (!cell || !wrapper) return;

  // Check if content is truncated vertically (due to line-clamp)
  const contentHeight = wrapper.scrollHeight;
  const visibleHeight = wrapper.offsetHeight;
  const verticalTruncated = contentHeight > visibleHeight + 2; // +2 for minor pixel differences

  // Check if content is truncated horizontally
  const contentWidth = wrapper.scrollWidth;
  const visibleWidth = cell.offsetWidth;
  const horizontalTruncated = contentWidth > visibleWidth + 2;

  truncated.value = verticalTruncated || horizontalTruncated;
});

const clickable = computed(() => {
  if (truncated.value) return true;
  if (props.database.instanceResource.engine === Engine.MONGODB) {
    // A cheap way to check JSON string without paying the parsing cost.
    const maybeJSON = String(props.value).trim();
    return (
      (maybeJSON.startsWith("{") && maybeJSON.endsWith("}")) ||
      (maybeJSON.startsWith("[") && maybeJSON.endsWith("]"))
    );
  }
  return false;
});

const selected = computed(() => {
  if (!allowSelect.value) {
    return false;
  }
  if (
    selectionState.value.columns.length === 1 &&
    selectionState.value.rows.length === 1
  ) {
    if (
      selectionState.value.columns[0] === props.colIndex &&
      selectionState.value.rows[0] === props.rowIndex
    ) {
      return true;
    }
  } else if (
    selectionState.value.columns.includes(props.colIndex) ||
    selectionState.value.rows.includes(props.rowIndex)
  ) {
    return true;
  }
  return false;
});

const classes = computed(() => {
  const classes: string[] = [];
  if (allowSelect.value) {
    classes.push("cursor-pointer");
    classes.push("hover:bg-accent/10 dark:hover:bg-accent/20");
    if (selected.value) {
      classes.push("bg-accent/20! dark:bg-accent/40!");
    }
  } else {
    classes.push("select-none");
  }
  return twMerge(classes);
});

const plainValue = computed(() => {
  return getPlainValue(props.value, props.columnType, binaryFormat.value);
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

  let kw = props.scope?.value.trim();
  if (!kw) {
    kw = props.keyword.trim();
  }
  if (!kw) {
    return escape(value);
  }

  return getHighlightHTMLByRegExp(
    value,
    escape(kw),
    false /* !caseSensitive */
  );
});

const handleClick = (e: MouseEvent) => {
  if (!allowSelect.value) {
    return;
  }
  const selectedString = window.getSelection()?.toString();
  if (selectedString) {
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
  };
};
</script>
