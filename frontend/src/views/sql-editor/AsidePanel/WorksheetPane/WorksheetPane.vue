<template>
  <div class="h-full flex flex-col gap-1 overflow-hidden pb-1 text-sm">
    <div class="flex items-center gap-x-1 px-1">
      <SearchBox
        v-model:value="keyword"
        size="small"
        :placeholder="$t('sheet.search-sheets')"
        :clearable="true"
        style="max-width: 100%"
      />
      <NButton
        quaternary
        style="--n-padding: 0 5px; --n-height: 28px"
        @click="showPanel = true"
      >
        <template #icon>
          <heroicons:arrow-left-on-rectangle />
        </template>
      </NButton>
    </div>
    <NScrollbar class="flex-1 overflow-hidden">
      <NCollapse v-model:expanded-names="expandedGroups" class="worksheet-pane">
        <NCollapseItem name="my" :title="$t('sheet.mine')">
          <SheetList view="my" :keyword="keyword" @ready="setReady('my')" />
        </NCollapseItem>
        <NCollapseItem name="starred" :title="$t('sheet.starred')">
          <SheetList
            view="starred"
            :keyword="keyword"
            @ready="setReady('starred')"
          />
        </NCollapseItem>
        <NCollapseItem name="shared" :title="$t('sheet.shared')">
          <SheetList
            view="shared"
            :keyword="keyword"
            @ready="setReady('shared')"
          />
        </NCollapseItem>
        <NCollapseItem name="draft" :title="$t('sheet.draft')">
          <DraftList :keyword="keyword" @ready="setReady('draft')" />
        </NCollapseItem>
      </NCollapse>
    </NScrollbar>
  </div>
</template>

<script setup lang="ts">
import { NButton, NCollapse, NCollapseItem, NScrollbar } from "naive-ui";
import { ref, watch } from "vue";
import { SearchBox } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useCurrentUserV1,
  useSQLEditorTabStore,
  useWorkSheetStore,
} from "@/store";
import { useSheetContext } from "../../Sheet";
import { SheetList } from "./SheetList";
import DraftList from "./SheetList/DraftList.vue";
import type { GroupType } from "./common";
import { useScrollLogic } from "./scroll-logic";

const { showPanel, events: sheetEvents } = useSheetContext();
const tabStore = useSQLEditorTabStore();
const sheetStore = useWorkSheetStore();
const me = useCurrentUserV1();
const keyword = ref("");
const expandedGroups = ref<GroupType[]>(["my", "starred", "shared", "draft"]);

const maybeExpandGroup = (group: GroupType) => {
  if (expandedGroups.value.includes(group)) return;
  expandedGroups.value.push(group);
};

useEmitteryEventListener(sheetEvents, "add-sheet", () => {
  maybeExpandGroup("draft");
});

const { setReady, scrollCurrentItemIntoView } = useScrollLogic();

watch(
  () => tabStore.currentTab,
  (tab) => {
    if (!tab) {
      return;
    }
    if (!tab.worksheet) {
      maybeExpandGroup("draft");
      scrollCurrentItemIntoView(tab);
      return;
    }
    const sheet = sheetStore.getWorksheetByName(tab.worksheet);
    if (!sheet) {
      return;
    }
    if (sheet.starred) {
      maybeExpandGroup("starred");
    } else if (sheet.creator != `users/${me.value.email}`) {
      maybeExpandGroup("shared");
    }

    scrollCurrentItemIntoView(tab);
  },
  { immediate: true }
);
</script>

<style lang="postcss" scoped>
.worksheet-pane {
  --n-title-padding: 0.5rem 0 0.5rem 6px !important;
  --n-item-margin: 0.25rem 0 0.25rem !important;
}
.worksheet-pane
  :deep(
    .n-collapse-item
      .n-collapse-item__content-wrapper
      .n-collapse-item__content-inner
  ) {
  padding-top: 0rem !important;
}
.worksheet-pane :deep(.n-collapse-item:first-child) {
  margin-top: 0.25rem;
}
</style>
