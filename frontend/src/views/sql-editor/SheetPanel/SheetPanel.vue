<template>
  <div class="w-[75vw] max-w-[calc(100vw-2rem)]">
    <NTabs v-model:value="view">
      <NTabPane name="my" :tab="$t('sheet.my-sheets')">
        <SheetTable view="my" @select-sheet="handleSelectSheet" />
      </NTabPane>
      <NTabPane name="shared" :tab="$t('sheet.shared-with-me')">
        <SheetTable view="shared" @select-sheet="handleSelectSheet" />
      </NTabPane>
      <NTabPane name="starred" :tab="$t('common.starred')">
        <SheetTable view="starred" @select-sheet="handleSelectSheet" />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script setup lang="ts">
import { NTabs, NTabPane } from "naive-ui";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { useSheetContext, openSheet } from "../Sheet";
import SheetTable from "./SheetTable";

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { view } = useSheetContext();

const handleSelectSheet = async (sheet: Sheet) => {
  if (await openSheet(sheet)) {
    emit("close");
  }
};
</script>
