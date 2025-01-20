<!-- eslint-disable vue/no-v-html -->
<template>
  <div
    class="relative px-2 py-1"
    :class="classes"
    @click="handleClick"
    @dblclick="showDetail"
  >
    <div
      ref="wrapperRef"
      class="whitespace-nowrap font-mono text-start line-clamp-1"
      v-html="html"
    ></div>
    <div v-if="clickable" class="absolute right-1 top-1/2 translate-y-[-45%]">
      <NButton
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
import { computed, ref } from "vue";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import type { QueryRow, RowValue } from "@/types/proto/v1/sql_service";
import { extractSQLRowValuePlain, getHighlightHTMLByRegExp } from "@/utils";
import { useSQLResultViewContext } from "../context";
import { useSelectionContext } from "./common/selection-logic";

const props = defineProps<{
  table: Table<QueryRow>;
  value: RowValue;
  setIndex: number;
  rowIndex: number;
  colIndex: number;
  allowSelect?: boolean;
}>();

const { detail, keyword } = useSQLResultViewContext();

const {
  state: selectionState,
  disabled: selectionDisabled,
  selectCell,
  selectRow,
} = useSelectionContext();
const wrapperRef = ref<HTMLDivElement>();
const truncated = ref(false);

const allowSelect = computed(() => {
  return props.allowSelect && !selectionDisabled.value;
});

useResizeObserver(wrapperRef, (entries) => {
  const div = entries[0].target as HTMLDivElement;
  const contentWidth = div.scrollWidth;
  const visibleWidth = div.offsetWidth;
  if (contentWidth > visibleWidth) {
    truncated.value = true;
  } else {
    truncated.value = false;
  }
});
const { database } = useConnectionOfCurrentSQLEditorTab();

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

const html = computed(() => {
  const value = extractSQLRowValuePlain(props.value);
  if (value === undefined) {
    return `<span class="text-gray-400 italic">UNSET</span>`;
  }
  if (value === null) {
    return `<span class="text-gray-400 italic">NULL</span>`;
  }
  const str = String(value);
  if (str.length === 0) {
    return `<br style="min-width: 1rem; display: inline-flex;" />`;
  }

  const kw = keyword.value.trim();
  if (!kw) {
    return escape(str);
  }

  return getHighlightHTMLByRegExp(
    escape(str),
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
    show: true,
    set: props.setIndex,
    row: props.rowIndex,
    col: props.colIndex,
    table: props.table,
  };
};
</script>
