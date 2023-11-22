<!-- eslint-disable vue/no-v-html -->
<template>
  <div class="relative px-2 py-1" :class="classes" @click="handleClick">
    <div
      ref="wrapperRef"
      class="overflow-hidden whitespace-pre font-mono"
      :class="valueContainerAdditionalClass"
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
import { useResizeObserver } from "@vueuse/core";
import { escape } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useDatabaseV1Store, useTabStore } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { getHighlightHTMLByRegExp } from "@/utils";
import { useSQLResultViewContext } from "../context";

const props = defineProps<{
  value: unknown;
  setIndex: number;
  rowIndex: number;
  colIndex: number;
}>();

const { dark, disallowCopyingData, detail, keyword } =
  useSQLResultViewContext();
const wrapperRef = ref<HTMLDivElement>();
const truncated = ref(false);

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

const database = computed(() => {
  const conn = useTabStore().currentTab.connection;
  return useDatabaseV1Store().getDatabaseByUID(conn.databaseId);
});

const valueContainerAdditionalClass = computed(() => {
  // Always only show the first line for MongoDB.
  if (database.value.instanceEntity.engine === Engine.MONGODB) {
    return "line-clamp-1";
  }
  return "";
});

const clickable = computed(() => {
  if (truncated.value) return true;
  if (database.value.instanceEntity.engine === Engine.MONGODB) {
    // A cheap way to check JSON string without paying the parsing cost.
    return (
      String(props.value).startsWith("{") && String(props.value).endsWith("}")
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
  if (props.value === null) {
    return `<span class="text-gray-400 italic">NULL</span>`;
  }
  const str = String(props.value);
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
  };
};
</script>
