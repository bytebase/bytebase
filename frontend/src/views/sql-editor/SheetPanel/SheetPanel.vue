<template>
  <div class="w-[75vw] max-w-[calc(100vw-2rem)]">
    <NTabs v-model:value="view">
      <NTabPane name="my" :tab="$t('sheet.mine')">
        <SheetTable
          view="my"
          :keyword="keyword"
          @select-sheet="handleSelectSheet"
        />
      </NTabPane>
      <NTabPane name="starred" :tab="$t('sheet.starred')">
        <SheetTable
          view="starred"
          :keyword="keyword"
          @select-sheet="handleSelectSheet"
        />
      </NTabPane>
      <NTabPane
        v-if="!isStandaloneMode"
        name="shared"
        :tab="$t('sheet.shared-with-me')"
      >
        <SheetTable
          view="shared"
          :keyword="keyword"
          @select-sheet="handleSelectSheet"
        />
      </NTabPane>

      <template #suffix>
        <div class="flex items-center gap-x-2">
          <SearchBox
            v-model:value="keyword"
            :placeholder="$t('sheet.search-sheets')"
          />
          <NButton type="primary" @click="handleAddSheet">
            {{ $t("common.create") }}
          </NButton>
        </div>
      </template>
    </NTabs>
  </div>
</template>

<script setup lang="ts">
import { NButton, NTabs, NTabPane } from "naive-ui";
import { computed, ref } from "vue";
import { SearchBox } from "@/components/v2";
import { usePageMode } from "@/store";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { useSheetContext, openSheet, addNewSheet } from "../Sheet";
import SheetTable from "./SheetTable";

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { view, events } = useSheetContext();
const pageMode = usePageMode();
const keyword = ref("");

const isStandaloneMode = computed(() => pageMode.value === "STANDALONE");

const handleSelectSheet = async (sheet: Sheet) => {
  if (await openSheet(sheet)) {
    emit("close");
  }
};

const handleAddSheet = () => {
  addNewSheet();
  emit("close");
  events.emit("add-sheet");
};
</script>
