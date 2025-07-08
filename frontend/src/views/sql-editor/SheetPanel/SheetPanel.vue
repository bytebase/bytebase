<template>
  <div class="w-[75vw] max-w-[calc(100vw-2rem)] relative">
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
      <NTabPane name="shared" :tab="$t('sheet.shared')">
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
    <MaskSpinner v-if="isFetching" />
  </div>
</template>

<script setup lang="ts">
import { NButton, NTabs, NTabPane } from "naive-ui";
import { ref } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { SearchBox } from "@/components/v2";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import { useSheetContext, openWorksheetByName, addNewSheet } from "../Sheet";
import { useSQLEditorContext } from "../context";
import SheetTable from "./SheetTable";

const emit = defineEmits<{
  (event: "close"): void;
}>();

const editorContext = useSQLEditorContext();
const worksheetContext = useSheetContext();
const { view, isFetching, events } = worksheetContext;
const keyword = ref("");

const handleSelectSheet = async (sheet: Worksheet) => {
  if (await openWorksheetByName(sheet.name, editorContext, worksheetContext)) {
    emit("close");
  }
};

const handleAddSheet = () => {
  addNewSheet();
  emit("close");
  events.emit("add-sheet");
};
</script>
