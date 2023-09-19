<template>
  <div
    ref="tabsContainerRef"
    class="w-full max-w-full overflow-x-auto bg-gray-100 select-none shrink-0 rounded"
  >
    <div class="w-fit flex flex-nowrap">
      <div
        v-for="tab in tabList"
        :key="tab.id"
        class="relative px-1 pl-2 py-1 rounded w-40 flex flex-row justify-between items-center shrink-0 border border-transparent cursor-pointer"
        :class="[
          `tab-${tab.id}`,
          tab.id === currentTab?.id && 'bg-white border-gray-200',
        ]"
        @click="handleSelectTab(tab)"
      >
        <div class="flex flex-row justify-start items-center mr-1">
          <span class="mr-1">
            <heroicons-outline:table-cells
              v-if="tab.type === SchemaEditorTabType.TabForTable"
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
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, nextTick, ref, watch } from "vue";
import { useSchemaEditorV1Store } from "@/store";
import { SchemaEditorTabType, TabContext } from "@/types/v1/schemaEditor";
import { isTableChanged } from "./utils";

const schemaEditorV1Store = useSchemaEditorV1Store();
const tabsContainerRef = ref();
const tabList = computed(() => {
  return schemaEditorV1Store.tabList;
});
const currentTab = computed(() => {
  return schemaEditorV1Store.currentTab;
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
  if (tab.type === SchemaEditorTabType.TabForTable) {
    const table = schemaEditorV1Store.getTableWithTableTab(tab);
    if (!table) {
      return [];
    }

    if (table.status === "dropped") {
      return ["text-red-700", "line-through"];
    }
    if (table.status === "created") {
      return ["text-green-700"];
    }
    if (isTableChanged(tab.parentName, tab.schemaId, tab.tableId)) {
      return ["text-yellow-700"];
    }
  }
  return [];
};

const getTabName = (tab: TabContext) => {
  if (tab.type === SchemaEditorTabType.TabForDatabase) {
    const database = schemaEditorV1Store.databaseList.find(
      (database) => database.name === tab.parentName
    );
    return database?.databaseName || "Uknown database";
  } else if (tab.type === SchemaEditorTabType.TabForTable) {
    const table = schemaEditorV1Store.getTableWithTableTab(tab);
    return table?.name || "Uknown table";
  } else {
    // Should never reach here.
    return "unknown tab";
  }
};

const handleSelectTab = (tab: TabContext) => {
  schemaEditorV1Store.setCurrentTab(tab.id);
};

const handleCloseTab = (tab: TabContext) => {
  schemaEditorV1Store.closeTab(tab.id);
};
</script>
