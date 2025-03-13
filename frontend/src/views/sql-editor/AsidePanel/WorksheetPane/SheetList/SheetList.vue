<template>
  <div class="flex flex-col items-stretch gap-y-1">
    <div v-if="isLoading" class="p-2 pl-7">
      <BBSpin :size="16" />
    </div>

    <SheetListItem
      v-for="worksheet in sortedWorksheetList"
      :key="worksheet.name"
      :view="view"
      :worksheet="worksheet"
      :keyword="keyword"
    />

    <div
      v-if="!isLoading && sortedWorksheetList.length === 0"
      class="p-2 pl-7 text-control-placeholder"
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
import { isValidDatabaseName } from "@/types";
import {
  useSheetContextByView,
  type SheetViewMode,
} from "@/views/sql-editor/Sheet";
import SheetListItem from "./SheetListItem.vue";

const props = defineProps<{
  view: SheetViewMode;
  keyword?: string;
}>();

const emit = defineEmits<{
  (event: "ready"): void;
}>();

const { isInitialized, isLoading, sheetList, fetchSheetList } =
  useSheetContextByView(props.view);
const databaseStore = useDatabaseV1Store();

const filteredWorksheetList = computed(() => {
  let sheets = sheetList.value;

  // Filter by keyword.
  const keyword = (props.keyword ?? "").trim().toLowerCase();
  if (keyword) {
    sheets = sheets.filter((worksheet) =>
      worksheet.title.toLowerCase().includes(keyword)
    );
  }

  return sheets;
});

const sortedWorksheetList = computed(() => {
  return orderBy(
    filteredWorksheetList.value,
    [
      // Unconnected sheets go behind
      (worksheet) => {
        if (!worksheet.database) {
          return Number.MAX_VALUE;
        }
        const db = databaseStore.getDatabaseByName(worksheet.database);
        if (!isValidDatabaseName(db.name)) {
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
      emit("ready");
    }
  },
  { immediate: true }
);
</script>
