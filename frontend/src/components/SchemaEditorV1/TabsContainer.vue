<template>
  <div class="flex justify-between items-center">
    <div
      class="relative flex flex-1 flex-nowrap overflow-hidden overflow bg-gray-100 select-none shrink-0 p-1 rounded"
    >
      <div
        ref="tabsContainerRef"
        class="flex flex-nowrap overflow-x-auto max-x-full hide-scrollbar overscroll-none"
      >
        <div
          v-for="tab in tabList"
          ref="tabsContainerRef"
          :key="tab.id"
          class="relative px-1 pl-2 py-1 rounded w-40 flex flex-row justify-between items-center shrink-0 border border-transparent cursor-pointer"
          :class="[
            `tab-${tab.id}`,
            tab.id === currentTab?.id && 'bg-white border-gray-200 shadow',
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
        </div>
      </div>
    </div>
    <div v-if="currentTab" class="hidden lg:block -mt-0.5">
      <NInput
        v-if="currentTab.type === SchemaEditorTabType.TabForDatabase"
        v-model:value="searchPattern"
        class="!w-48"
        :placeholder="$t('schema-editor.search-table')"
        @input="$emit('onTableSearchPattern', $event)"
      >
        <template #prefix>
          <heroicons-outline:search class="w-4 h-auto text-gray-300" />
        </template>
      </NInput>
      <NInput
        v-if="currentTab.type === SchemaEditorTabType.TabForTable"
        v-model:value="searchPattern"
        class="!w-48"
        :placeholder="$t('schema-editor.search-column')"
        @input="$emit('onColumnSearchPattern', $event)"
      >
        <template #prefix>
          <heroicons-outline:search class="w-4 h-auto text-gray-300" />
        </template>
      </NInput>
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

defineEmits<{
  (event: "onTableSearchPattern", tableSearchPattern: string): void;
  (event: "onColumnSearchPattern", columnSearchPattern: string): void;
}>();

const searchPattern = ref("");

const originalTabContext = ref<TabContext | undefined>(undefined);

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
    if (currentTab.value !== originalTabContext.value) {
      searchPattern.value = "";
      originalTabContext.value = currentTab.value;
    }

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
