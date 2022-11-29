<template>
  <div
    ref="tabsContainerRef"
    class="w-full max-w-full overflow-x-auto bg-gray-100 select-none shrink-0 rounded"
  >
    <div class="w-fit flex flex-nowrap">
      <div
        v-for="tab in tabList"
        :key="tab.id"
        class="tab-container px-1 pl-2 py-2 rounded w-40 flex flex-row justify-between items-center shrink-0 border border-transparent cursor-pointer"
        :class="[
          `tab-${tab.id}`,
          tab.id === currentTab?.id && 'bg-white border-gray-200',
        ]"
        @click="handleSelectTab(tab)"
      >
        <div class="flex flex-row justify-start items-center mr-1">
          <span class="mr-1">
            <heroicons-outline:circle-stack
              v-if="tab.type === UIEditorTabType.TabForDatabase"
              class="rounded w-4 h-auto text-gray-400"
            />
            <heroicons-outline:table-cells
              v-else-if="tab.type === UIEditorTabType.TabForTable"
              class="rounded w-4 h-auto text-gray-400"
            />
          </span>
          <NEllipsis
            class="text-sm w-24"
            :class="checkTabSaved(tab) ? '' : 'italic'"
            >{{ getTabName(tab) }}</NEllipsis
          >
        </div>
        <span class="tab-close-button shrink-0 flex">
          <heroicons-outline:x
            class="rounded w-4 h-auto text-gray-400 hover:text-gray-600"
            @click.stop.prevent="handleCloseTab(tab)"
          />
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { NEllipsis, useDialog } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import scrollIntoView from "scroll-into-view-if-needed";
import { useUIEditorStore } from "@/store";
import { TabContext, UIEditorTabType } from "@/types";

const editorStore = useUIEditorStore();
const tabsContainerRef = ref();
const tabList = computed(() => {
  return Array.from(editorStore.tabState.tabMap.values());
});
const currentTab = computed(() => {
  return editorStore.currentTab;
});
const dialog = useDialog();

watch(
  () => currentTab.value,
  () => {
    nextTick(() => {
      const element = tabsContainerRef.value?.querySelector(
        `.tab-${currentTab.value?.id}`
      );
      scrollIntoView(element);
    });
  }
);

const checkTabSaved = (tab: TabContext) => {
  if (tab.type === UIEditorTabType.TabForTable) {
    const table = editorStore.tableList.find((table) => {
      return (
        table.id === tab.tableCache.id &&
        table.database.id === tab.tableCache.database.id &&
        table.name === tab.tableCache.name
      );
    });
    if (!isEqual(tab.tableCache, table)) {
      return false;
    }
  } else if (tab.type === UIEditorTabType.TabForDatabase) {
    // Edit database metadata is not allowed.
    return true;
  }

  return true;
};

const getTabName = (tab: TabContext) => {
  if (tab.type === UIEditorTabType.TabForDatabase) {
    const database = editorStore.databaseList.find(
      (database) => database.id === tab.databaseId
    );
    return `${database?.name || "unknown database"}`;
  } else if (tab.type === UIEditorTabType.TabForTable) {
    return `${tab.tableCache.name}`;
  } else {
    // Should never reach here.
    return "unknown structure";
  }
};

const handleSelectTab = (tab: TabContext) => {
  editorStore.setCurrentTab(tab.id);
};

const handleCloseTab = (tab: TabContext) => {
  if (tab.type === UIEditorTabType.TabForDatabase) {
    editorStore.closeTab(tab.id);
  } else if (tab.type === UIEditorTabType.TabForTable) {
    const table = editorStore.tableList.find((table) => {
      return (
        table.id === tab.tableCache.id &&
        table.database.id === tab.tableCache.database.id &&
        table.name === tab.tableCache.name
      );
    });
    if (!table) {
      return;
    }
    if (isEqual(tab.tableCache, table)) {
      editorStore.closeTab(tab.id);
    } else {
      dialog.warning({
        title: "Confirm to close",
        content:
          "There are unsaved changes. Are you sure you want to close the panel? Your changes will be lost.",
        negativeText: "Cancel",
        positiveText: "Confirm",
        onPositiveClick: () => {
          editorStore.closeTab(tab.id);
        },
      });
    }
  }
};
</script>

<style scoped>
.tab-container:hover > .tab-close-button {
  @apply flex;
}
</style>
