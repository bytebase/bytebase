<template>
  <div
    ref="tabsContainerRef"
    class="w-full max-w-full overflow-x-auto bg-gray-100 select-none shrink-0 rounded"
  >
    <div class="w-fit flex flex-nowrap">
      <div
        v-for="tab in tabList"
        :key="tab.id"
        class="relative px-1 pl-2 py-2 rounded w-40 flex flex-row justify-between items-center shrink-0 border border-transparent cursor-pointer"
        :class="[
          `tab-${tab.id}`,
          tab.id === currentTab?.id && 'bg-white border-gray-200',
        ]"
        @click="handleSelectTab(tab)"
      >
        <div class="flex flex-row justify-start items-center mr-1">
          <span class="mr-1">
            <heroicons-outline:table-cells
              v-if="tab.type === SchemaDesignerTabType.TabForTable"
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
        <div
          v-if="tab.id === currentTab?.id"
          class="absolute -bottom-px left-0 w-full h-[3px] bg-accent rounded"
        ></div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import scrollIntoView from "scroll-into-view-if-needed";
import {
  useSchemaDesignerContext,
  TabContext,
  SchemaDesignerTabType,
} from "./common";

const { tabState, getCurrentTab, getTable } = useSchemaDesignerContext();
const tabsContainerRef = ref();
const tabList = computed(() => {
  return Array.from(tabState.value.tabMap.values());
});
const currentTab = computed(() => {
  return getCurrentTab();
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
  return [];
};

const getTabName = (tab: TabContext) => {
  if (tab.type === SchemaDesignerTabType.TabForTable) {
    const table = getTable(tab.schemaId, tab.tableId);
    return table.name || "Uknown tab";
  } else {
    // Should never reach here.
    return "unknown structure";
  }
};

const handleSelectTab = (tab: TabContext) => {
  tabState.value.currentTabId = tab.id;
};

const handleCloseTab = (tab: TabContext) => {
  tabState.value.currentTabId = undefined;
};
</script>
