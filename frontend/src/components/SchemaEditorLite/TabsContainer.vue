<template>
  <div class="flex justify-between items-center">
    <div
      class="relative flex flex-1 flex-nowrap overflow-hidden overflow bg-gray-100 select-none shrink-0 rounded-sm"
    >
      <div
        ref="tabsContainerRef"
        class="flex flex-nowrap overflow-x-auto max-x-full hide-scrollbar overscroll-none px-1 gap-x-1"
      >
        <div
          v-for="tab in tabList"
          ref="tabsContainerRef"
          :key="tab.id"
          :class="[
            `relative px-1 py-1 my-1 rounded-sm w-40 flex flex-row justify-between items-center shrink-0 border border-transparent cursor-pointer`,
            `tab-${tab.id}`,
            tab.id === currentTab?.id && 'bg-white border-gray-200 shadow-sm',
          ]"
          @click="handleSelectTab(tab)"
        >
          <div class="flex flex-row justify-start items-center mr-1">
            <span class="mr-1">
              <DatabaseIcon
                v-if="tab.type === 'database'"
                class="w-4 h-4 text-gray-400"
              />
              <TableIcon
                v-if="tab.type === 'table'"
                class="w-4 h-4 text-gray-400"
              />
              <ViewIcon
                v-if="tab.type === 'view'"
                class="w-4 h-4 text-gray-400"
              />
              <ProcedureIcon
                v-if="tab.type === 'procedure'"
                class="w-4 h-4 text-gray-400"
              />
              <FunctionIcon
                v-if="tab.type === 'function'"
                class="w-4 h-4 text-gray-400"
              />
            </span>
            <NEllipsis
              class="text-sm w-20 leading-4"
              :class="getTabComputedClassList(tab)"
              >{{ getTabName(tab) }}</NEllipsis
            >
          </div>
          <span class="tab-close-button shrink-0 flex">
            <XIcon
              class="rounded-sm w-4 h-auto text-gray-400 hover:text-gray-600"
              @click.stop.prevent="handleCloseTab(tab)"
            />
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { XIcon } from "lucide-vue-next";
import { NEllipsis } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { nextTick, ref, watch } from "vue";
import {
  DatabaseIcon,
  FunctionIcon,
  ProcedureIcon,
  TableIcon,
  ViewIcon,
} from "../Icon";
import { useSchemaEditorContext } from "./context";
import type { TabContext } from "./types";

const { tabList, currentTab, setCurrentTab, closeTab, getTableStatus } =
  useSchemaEditorContext();
const tabsContainerRef = ref();
const searchPattern = ref("");
const originalTabContext = ref<TabContext | undefined>(undefined);

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
  if (tab.type === "table") {
    const { database, metadata } = tab;
    const status = getTableStatus(database, metadata);

    if (status === "dropped") {
      return ["text-red-700", "line-through"];
    }
    if (status === "created") {
      return ["text-green-700"];
    }
    if (status === "updated") {
      return ["text-yellow-700"];
    }
  }
  return [];
};

const getTabName = (tab: TabContext) => {
  if (tab.type === "database") {
    const { database } = tab;
    return database.databaseName;
  }
  if (tab.type === "table") {
    const {
      metadata: { schema, table },
    } = tab;
    const parts = [table.name];
    if (schema.name) {
      parts.unshift(schema.name);
    }
    return parts.join(".");
  }
  if (tab.type === "view") {
    const {
      metadata: { schema, view },
    } = tab;
    const parts = [view.name];
    if (schema.name) {
      parts.unshift(schema.name);
    }
    return parts.join(".");
  }
  if (tab.type === "procedure") {
    const {
      metadata: { schema, procedure },
    } = tab;
    const parts = [procedure.name];
    if (schema.name) {
      parts.unshift(schema.name);
    }
    return parts.join(".");
  }
  if (tab.type === "function") {
    const {
      metadata: { schema, function: func },
    } = tab;
    const parts = [func.name];
    if (schema.name) {
      parts.unshift(schema.name);
    }
    return parts.join(".");
  }
  // Should never reach here.
  return "unknown tab";
};

const handleSelectTab = (tab: TabContext) => {
  setCurrentTab(tab.id);
};

const handleCloseTab = (tab: TabContext) => {
  closeTab(tab.id);
};
</script>
