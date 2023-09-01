<template>
  <NTabs
    v-model:value="sheetTab"
    size="small"
    class="h-full pt-1"
    pane-style="height: calc(100% - 29px); padding: 0;"
    :tabs-padding="4"
  >
    <NTabPane name="my" :tab="$t('sheet.mine')">
      <SheetList view="my" />
    </NTabPane>
    <NTabPane name="starred" :tab="$t('sheet.starred')">
      <SheetList view="starred" />
    </NTabPane>
    <NTabPane name="shared" :tab="$t('sheet.shared-with-me')">
      <SheetList view="shared" />
    </NTabPane>
  </NTabs>
</template>

<script setup lang="ts">
import { NTabs, NTabPane } from "naive-ui";
import { ref } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSheetContext } from "../../Sheet";
import SheetList from "./SheetList";

const { events: sheetEvents } = useSheetContext();

const sheetTab = ref<"my" | "shared" | "starred">("my");

useEmitteryEventListener(sheetEvents, "add-sheet", () => {
  sheetTab.value = "my";
});
</script>
