<template>
  <div class="h-full overflow-hidden pb-1">
    <NScrollbar class="h-full overflow-hidden">
      <NCollapse
        class="worksheet-pane"
        :default-expanded-names="['my', 'starred', 'shared']"
      >
        <NCollapseItem name="my" :title="$t('sheet.mine')">
          <SheetList view="my" />
        </NCollapseItem>
        <NCollapseItem name="starred" :title="$t('sheet.starred')">
          <SheetList view="starred" />
        </NCollapseItem>
        <NCollapseItem
          v-if="!isStandaloneMode"
          name="shared"
          :title="$t('sheet.shared')"
        >
          <SheetList view="shared" />
        </NCollapseItem>
      </NCollapse>
    </NScrollbar>
  </div>
</template>

<script setup lang="ts">
import { NCollapse, NCollapseItem, NScrollbar } from "naive-ui";
import { computed, watch } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { usePageMode, useSQLEditorTabStore, useWorkSheetStore } from "@/store";
import { useSheetContext } from "../../Sheet";
import SheetList from "./SheetList";

const { events: sheetEvents } = useSheetContext();
const pageMode = usePageMode();
const tabStore = useSQLEditorTabStore();
const sheetStore = useWorkSheetStore();

const isStandaloneMode = computed(() => pageMode.value === "STANDALONE");

useEmitteryEventListener(sheetEvents, "add-sheet", () => {
  // TODO: expand "my" group
});

watch(
  () => tabStore.currentTab,
  (tab) => {
    if (!tab) {
      return;
    }
    if (tab.sheet) {
      return;
    }
    const sheet = sheetStore.getSheetByName(tab.sheet);
    if (!sheet) {
      return;
    }
    // TODO: expand starred or shared group if needed
    // if (sheet.starred) {
    //   sheetTab.value = "starred";
    // } else if (sheet.creator != `users/${me.value.email}`) {
    //   sheetTab.value = "shared";
    // }
  },
  { immediate: true }
);
</script>

<style lang="postcss">
.worksheet-pane {
  --n-title-padding: 0.5rem 0 0 2px !important;
  --n-item-margin: 0.5rem 0 !important;
}
.worksheet-pane
  .n-collapse-item
  .n-collapse-item__content-wrapper
  .n-collapse-item__content-inner {
  padding-top: 0 !important;
}
</style>
