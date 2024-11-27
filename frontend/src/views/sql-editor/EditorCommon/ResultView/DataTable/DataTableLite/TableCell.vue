<!-- eslint-disable vue/no-v-html -->
<template>
  <div
    class="relative px-2 py-1 text-sm dark:text-gray-100 leading-5 whitespace-nowrap break-all"
    :class="classes"
    @click="handleClick"
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
import { escape } from "lodash-es";
import { NButton } from "naive-ui";
import stringWidth from "string-width";
import { computed, ref } from "vue";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import type { QueryRow, RowValue } from "@/types/proto/v1/sql_service";
import { extractSQLRowValue, getHighlightHTMLByRegExp } from "@/utils";
import { useSQLResultViewContext } from "../../context";

const props = defineProps<{
  table: Table<QueryRow>;
  value: RowValue;
  width: number;
  setIndex: number;
  rowIndex: number;
  colIndex: number;
}>();

const { dark, disallowCopyingData, detail, keyword } =
  useSQLResultViewContext();
const wrapperRef = ref<HTMLDivElement>();
const plainValue = computed(() => {
  return extractSQLRowValue(props.value).plain;
});
const truncated = computed(() => {
  // not that accurate
  const content = String(plainValue.value);
  const em = 8;
  const padding = 8;
  const guessedContentWidth = stringWidth(content) * em + padding * 2;

  return guessedContentWidth > props.width;
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
  if (disallowCopyingData.value) {
    classes.push("select-none");
  }
  if (clickable.value) {
    classes.push("cursor-pointer");
    classes.push(dark.value ? "hover:!bg-white/20" : "hover:!bg-black/5");
  }
  return classes;
});

const html = computed(() => {
  const value = plainValue.value;
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

const handleClick = () => {
  if (!clickable.value) return;
  showDetail();
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
