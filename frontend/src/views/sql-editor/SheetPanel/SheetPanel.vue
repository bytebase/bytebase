<template>
  <div class="w-[75vw] max-w-[calc(100vw-2rem)]">
    <NTabs v-model:value="view">
      <NTabPane name="my" :tab="$t('sheet.mine')">
        <SheetTable view="my" @select-sheet="handleSelectSheet" />
      </NTabPane>
      <NTabPane name="starred" :tab="$t('sheet.starred')">
        <SheetTable view="starred" @select-sheet="handleSelectSheet" />
      </NTabPane>
      <NTabPane name="shared" :tab="$t('sheet.shared-with-me')">
        <SheetTable view="shared" @select-sheet="handleSelectSheet" />
      </NTabPane>

      <template #suffix>
        <NButton type="primary" @click="handleAddSheet">
          {{ $t("common.create") }}
        </NButton>
      </template>
    </NTabs>
  </div>
</template>

<script setup lang="ts">
import { NButton, NTabs, NTabPane } from "naive-ui";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { useSheetContext, openSheet, addNewSheet } from "../Sheet";
import SheetTable from "./SheetTable";

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { view, events } = useSheetContext();

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
