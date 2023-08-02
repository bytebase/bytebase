<template>
  <div class="flex flex-col h-full overflow-hidden">
    <div class="px-2 py-2 gap-x-1 flex items-center">
      <NInput
        v-model:value="keyword"
        :disabled="isLoading"
        :placeholder="$t('sheet.search-sheets')"
        :clearable="true"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
      <NButton quaternary style="--n-padding: 0 8px">
        <template #icon>
          <heroicons:plus />
        </template>
      </NButton>
    </div>
    <div class="flex-1 flex flex-col gap-y-2 h-full overflow-y-auto">
      <div v-if="isLoading" class="flex flex-col items-center py-8">
        <BBSpin />
      </div>
      <template v-else>
        <div
          v-if="filteredSheetList.length === 0"
          class="flex flex-col items-center justify-center text-control-placeholder"
        >
          <p class="py-8">{{ $t("common.no-data") }}</p>
        </div>
        <div
          v-for="sheet in filteredSheetList"
          :key="sheet.name"
          class="flex items-start justify-between hover:bg-gray-200 px-2 py-1 gap-x-1"
        >
          <div
            class="flex-1 text-sm cursor-pointer"
            @click="handleSheetClick(sheet)"
          >
            <!-- eslint-disable-next-line vue/no-v-html -->
            <span v-if="sheet.title" v-html="titleHTML(sheet)" />
            <span v-else class="text-control-placeholder">
              {{ $t("sql-editor.untitled-sheet") }}
            </span>
          </div>
          <div>
            <Dropdown :sheet="sheet" :view="view" />
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { NButton, NInput } from "naive-ui";
import { escape, orderBy } from "lodash-es";

import { Sheet } from "@/types/proto/v1/sheet_service";
import {
  SheetViewMode,
  openSheet,
  useSheetContextByView,
} from "@/views/sql-editor/Sheet";
import { Dropdown } from "@/views/sql-editor/Sheet";
import { extractSheetUID, getHighlightHTMLByRegExp } from "@/utils";

const props = defineProps<{
  view: SheetViewMode;
}>();

const { isInitialized, isLoading, sheetList, fetchSheetList } =
  useSheetContextByView(props.view);
const keyword = ref("");

const sortedSheetList = computed(() => {
  return orderBy<Sheet>(
    sheetList.value,
    [
      (sheet) => (extractSheetUID(sheet.name).startsWith("-") ? 0 : 1), // Unsaved sheets go ahead.
      (sheet) => (sheet.title ? 0 : 1), // Untitled sheets go behind.
      (sheet) => sheet.title,
    ],
    ["asc", "asc"]
  );
});

const filteredSheetList = computed(() => {
  const kw = keyword.value.toLowerCase().trim();
  if (!kw) return sortedSheetList.value;
  return sortedSheetList.value.filter((sheet) =>
    sheet.title.toLowerCase().includes(kw)
  );
});

const titleHTML = (sheet: Sheet) => {
  const kw = keyword.value.toLowerCase().trim();
  const { title } = sheet;

  if (!kw) {
    return escape(title);
  }

  return getHighlightHTMLByRegExp(
    escape(title),
    escape(kw),
    false /* !caseSensitive */
  );
};

const handleSheetClick = (sheet: Sheet) => {
  openSheet(sheet);
};

watch(
  isInitialized,
  () => {
    if (!isInitialized.value) {
      fetchSheetList();
    }
  },
  { immediate: true }
);
</script>
