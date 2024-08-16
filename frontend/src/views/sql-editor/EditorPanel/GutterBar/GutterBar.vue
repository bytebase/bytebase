<template>
  <div
    v-if="show"
    class="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1 shrink-0"
  >
    <div class="flex flex-col gap-y-1">
      <TabItem v-for="view in availableViews" :key="view" :view="view" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import { instanceV1SupportsExternalTable } from "@/utils";
import { useEditorPanelContext } from "../context";
import type { EditorPanelView } from "../types";
import TabItem from "./TabItem.vue";

const { instance } = useConnectionOfCurrentSQLEditorTab();
const { viewState } = useEditorPanelContext();

const show = computed(() => {
  return viewState.value?.view !== undefined;
});

const availableViews = computed(() => {
  const views: EditorPanelView[] = [
    "CODE",
    "INFO",
    "TABLES",
    "VIEWS",
    "FUNCTIONS",
    "PROCEDURES",
  ];
  if (instanceV1SupportsExternalTable(instance.value)) {
    views.push("EXTERNAL_TABLES");
  }
  views.push("DIAGRAM");
  return views;
});
</script>
