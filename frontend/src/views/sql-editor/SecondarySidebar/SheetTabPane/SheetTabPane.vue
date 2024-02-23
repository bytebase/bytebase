<template>
  <NTabs
    v-model:value="sheetTab"
    size="small"
    class="bb-sql-editor--sheet-tab-pane--tabs h-full pt-1.5 px-1"
    pane-style="height: calc(100% - 29px); padding: 0;"
    justify-content="start"
  >
    <NTabPane name="my" :tab="$t('sheet.mine')">
      <SheetList view="my" />
    </NTabPane>
    <NTabPane name="starred" :tab="$t('sheet.starred')">
      <SheetList view="starred" />
    </NTabPane>
    <NTabPane
      v-if="!isStandaloneMode"
      name="shared"
      :tab="$t('sheet.shared-with-me')"
    >
      <SheetList view="shared" />
    </NTabPane>
  </NTabs>
</template>

<script setup lang="ts">
import { NTabs, NTabPane } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  usePageMode,
  useTabStore,
  useWorkSheetStore,
  useCurrentUserV1,
} from "@/store";
import { useSheetContext } from "../../Sheet";
import SheetList from "./SheetList";

const { events: sheetEvents } = useSheetContext();
const pageMode = usePageMode();
const tabStore = useTabStore();
const sheetStore = useWorkSheetStore();
const me = useCurrentUserV1();

const isStandaloneMode = computed(() => pageMode.value === "STANDALONE");

const sheetTab = ref<"my" | "shared" | "starred">("my");

useEmitteryEventListener(sheetEvents, "add-sheet", () => {
  sheetTab.value = "my";
});

watch(
  () => tabStore.currentTab,
  (tab) => {
    if (!tab.isSaved || !tab.sheetName) {
      return;
    }
    const sheet = sheetStore.getSheetByName(tab.sheetName);
    if (!sheet) {
      return;
    }
    if (sheet.starred) {
      sheetTab.value = "starred";
    } else if (sheet.creator != `users/${me.value.email}`) {
      sheetTab.value = "shared";
    }
  },
  { immediate: true }
);
</script>

<style lang="postcss">
.bb-sql-editor--sheet-tab-pane--tabs
  .n-tabs-nav-scroll-wrapper--shadow-start::before,
.bb-sql-editor--sheet-tab-pane--tabs
  .n-tabs-nav-scroll-wrapper--shadow-end::after {
  @apply hidden;
}

.bb-sql-editor--sheet-tab-pane--tabs .n-tabs-wrapper {
  @apply px-1;
}
</style>
