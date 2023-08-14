<template>
  <div class="relative px-2 py-1" :class="classes" @click="handleClick">
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div ref="wrapperRef" class="overflow-hidden" v-html="html"></div>
    <div v-if="clickable" class="absolute right-0 top-1/2 translate-y-[-50%]">
      <NButton size="tiny" circle class="dark:!bg-dark-bg" @click="showDetail">
        <template #icon>
          <heroicons:arrows-pointing-out class="w-4 h-4" />
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
import { UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { getHighlightHTMLByRegExp } from "@/utils";
import { useSQLResultViewContext } from "../context";

const props = defineProps<{
  value: unknown;
  keyword?: string;
  setIndex: number;
  rowIndex: number;
  colIndex: number;
}>();

const { dark, disallowCopyingData, detail } = useSQLResultViewContext();
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

const clickable = computed(() => {
  if (truncated.value) return true;
  const conn = useTabStore().currentTab.connection;
  if (conn.databaseId !== String(UNKNOWN_ID)) {
    const db = useDatabaseV1Store().getDatabaseByUID(conn.databaseId);
    if (db.instanceEntity.engine === Engine.MONGODB) {
      return true;
    }
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

  const { keyword } = props;
  if (!keyword) {
    return escape(str);
  }

  return getHighlightHTMLByRegExp(
    escape(str),
    escape(keyword),
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
