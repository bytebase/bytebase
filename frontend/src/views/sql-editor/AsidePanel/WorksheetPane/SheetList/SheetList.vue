<template>
  <div class="flex flex-col items-stretch gap-y-1">
    <div
      class="text-xs font-mono max-h-[33vh] max-w-[40vw] overflow-auto fixed bottom-0 right-0 p-2 bg-white/50 border border-gray-400 z-[999999]"
    >
      <div>view: {{ view }}</div>
      <div>isInitialized: {{ isInitialized }}</div>
      <div>isLoading: {{ isLoading }}</div>
      <div>sheetList.length: {{ sheetList.length }}</div>
      <div v-for="sheet in sheetList" :key="sheet.name">
        {{ sheet.name }}
      </div>
    </div>

    <div v-if="isLoading" class="py-2">
      <BBSpin :size="16" />
    </div>

    <SheetListItem
      v-for="worksheet in sortedWorksheetList"
      :key="worksheet.name"
      :view="view"
      :worksheet="worksheet"
    />

    <div
      v-if="!isLoading && sortedWorksheetList.length === 0"
      class="py-2 text-control-placeholder"
    >
      {{ $t("common.no-data") }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { orderBy } from "lodash-es";
import { computed, nextTick, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { useDatabaseV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import {
  useSheetContextByView,
  type SheetViewMode,
} from "@/views/sql-editor/Sheet";
import SheetListItem from "./SheetListItem.vue";

const props = defineProps<{
  view: SheetViewMode;
  keyword?: string;
}>();

const { isInitialized, isLoading, sheetList, fetchSheetList } =
  useSheetContextByView(props.view);

const sortedWorksheetList = computed(() => {
  return orderBy(
    sheetList.value,
    [
      // Unconnected sheets go behind
      (worksheet) => {
        if (!worksheet.database) {
          return Number.MAX_VALUE;
        }
        const db = useDatabaseV1Store().getDatabaseByName(worksheet.database);
        if (db.uid === String(UNKNOWN_ID)) {
          return Number.MAX_VALUE;
        }
        return 1;
      },
      // Alphabetically by default
      (item) => item.title,
    ],
    ["asc", "asc"]
  );
});

watch(
  isInitialized,
  async () => {
    if (!isInitialized.value) {
      await fetchSheetList();
      await nextTick();
      // scrollToCurrentTabOrSheet();
    }
  },
  { immediate: true }
);
</script>
