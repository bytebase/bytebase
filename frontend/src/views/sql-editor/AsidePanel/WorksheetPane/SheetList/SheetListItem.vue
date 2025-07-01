<template>
  <AbstractListItem
    :title="worksheet.title"
    :selected="selected"
    :keyword="keyword"
    :data-item-key="keyForWorksheet(worksheet)"
    @click="handleClick"
  >
    <template #suffix>
      <Dropdown
        :sheet="worksheet"
        :view="props.view"
        :secondary="true"
        :unsaved="unsaved"
      />
    </template>
  </AbstractListItem>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useSQLEditorTabStore } from "@/store";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import type { SheetViewMode } from "@/views/sql-editor/Sheet";
import {
  Dropdown,
  openWorksheetByName,
  useSheetContext,
} from "@/views/sql-editor/Sheet";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import { keyForWorksheet } from "../common";
import AbstractListItem from "./AbstractListItem.vue";

const props = defineProps<{
  view: SheetViewMode;
  worksheet: Worksheet;
  keyword?: string;
}>();

const tabStore = useSQLEditorTabStore();
const editorContext = useSQLEditorContext();
const worksheetContext = useSheetContext();

const unsaved = computed(() => {
  const tab = tabStore.tabList.find(
    (tab) => tab.worksheet === props.worksheet.name
  );

  return tab && (tab.status === "DIRTY" || tab.status === "NEW");
});

const selected = computed(() => {
  const tab = tabStore.currentTab;

  return tab?.worksheet === props.worksheet.name;
});

const handleClick = (e: MouseEvent) => {
  openWorksheetByName(
    props.worksheet.name,
    editorContext,
    worksheetContext,
    e.metaKey || e.ctrlKey
  );
};
</script>
