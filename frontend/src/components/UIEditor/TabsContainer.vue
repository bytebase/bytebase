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
            :class="getTabComputedClassList(tab)"
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
import { NEllipsis } from "naive-ui";
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

watch(
  () => currentTab.value,
  () => {
    nextTick(() => {
      const element = tabsContainerRef.value?.querySelector(
        `.tab-${currentTab.value?.id}`
      );
      if (element) {
        scrollIntoView(element, {
          scrollMode: "if-needed",
        });
      }
    });
  }
);

const getTabComputedClassList = (tab: TabContext) => {
  if (tab.type === UIEditorTabType.TabForTable) {
    const table = editorStore.getTableWithTableTab(tab);
    if (!table) {
      return [];
    }

    if (table.status === "dropped") {
      return ["text-red-700", "line-through"];
    }
    if (table.status === "created") {
      return ["text-green-700"];
    }

    const originTable = editorStore.originTableList.find(
      (item) =>
        item.databaseId === table.databaseId && item.oldName === table.oldName
    );
    if (!isEqual(originTable, table)) {
      return ["text-yellow-700"];
    }
  }

  return [];
};

const getTabName = (tab: TabContext) => {
  if (tab.type === UIEditorTabType.TabForDatabase) {
    const database = editorStore.databaseList.find(
      (database) => database.id === tab.databaseId
    );
    return `${database?.name || "unknown database"}`;
  } else if (tab.type === UIEditorTabType.TabForTable) {
    return `${tab.tableName}`;
  } else {
    // Should never reach here.
    return "unknown structure";
  }
};

const handleSelectTab = (tab: TabContext) => {
  editorStore.setCurrentTab(tab.id);
};

const handleCloseTab = (tab: TabContext) => {
  editorStore.closeTab(tab.id);
};
</script>

<style scoped>
.tab-container:hover > .tab-close-button {
  @apply flex;
}
</style>
