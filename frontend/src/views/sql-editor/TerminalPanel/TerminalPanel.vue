<template>
  <div
    class="flex h-full w-full flex-col justify-start items-start overflow-hidden"
  >
    <EditorAction @execute="handleExecute" @save-sheet="trySaveSheet" />

    <ConnectionPathBar />

    <div
      v-if="!tabStore.isDisconnected"
      ref="queryListContainerRef"
      class="w-full flex-1 overflow-y-auto bg-dark-bg"
    >
      <div
        ref="queryListRef"
        class="w-full flex flex-col"
        :data-height="queryListHeight"
      >
        <div v-for="(query, i) in queryList" :key="i" class="relative">
          <CompactSQLEditor
            v-model:sql="query.sql"
            class="min-h-[2rem]"
            :readonly="!isEditableQueryItem(query)"
            @save-sheet="trySaveSheet"
            @execute="handleExecute"
          />
          <div v-if="query.queryResult" class="max-h-[20rem] overflow-y-auto">
            <TableView
              :query-result="query.queryResult.data"
              :loading="query.isExecutingSQL"
              :dark="true"
            />
          </div>

          <div
            v-if="query.isExecutingSQL"
            class="absolute inset-0 bg-black/20 flex justify-center items-center"
          >
            <BBSpin />
          </div>
        </div>
      </div>
    </div>
    <ConnectionHolder v-else />

    <SaveSheetModal ref="saveSheetModal" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { useElementSize } from "@vueuse/core";

import { ExecuteConfig, ExecuteOption, WebTerminalQueryItem } from "@/types";
import { useTabStore, useWebTerminalStore } from "@/store";
import CompactSQLEditor from "./CompactSQLEditor.vue";
import {
  EditorAction,
  ConnectionPathBar,
  ConnectionHolder,
  TableView,
  SaveSheetModal,
} from "../EditorCommon";
import { useExecuteSQL } from "@/composables/useExecuteSQL";

const tabStore = useTabStore();
const webTerminalStore = useWebTerminalStore();

const queryList = computed(() => {
  return webTerminalStore.getQueryListByTab(tabStore.currentTab);
});

const queryListContainerRef = ref<HTMLDivElement>();
const queryListRef = ref<HTMLDivElement>();

const currentQuery = computed(
  () => queryList.value[queryList.value.length - 1]
);

const { execute } = useExecuteSQL();

const isEditableQueryItem = (query: WebTerminalQueryItem): boolean => {
  return query === currentQuery.value && !query.isExecutingSQL;
};

const handleExecute = async (
  query: string,
  config: ExecuteConfig,
  option?: ExecuteOption
) => {
  const queryItem = currentQuery.value;
  if (queryItem.isExecutingSQL) {
    return;
  }

  // Prevent executing empty query;
  if (!query) {
    return;
  }

  try {
    queryItem.executeParams = { query, config, option };
    queryItem.isExecutingSQL = true;
    const sqlResultSet = await execute(query, config, option);

    queryItem.queryResult = sqlResultSet;
    queryList.value.push({
      sql: "",
      isExecutingSQL: false,
    });
    // Clear the tab's statement and keep it sync with the latest query
    tabStore.currentTab.statement = "";
    tabStore.currentTab.selectedStatement = "";
  } finally {
    queryItem.isExecutingSQL = false;
  }
};

const saveSheetModal = ref<InstanceType<typeof SaveSheetModal>>();

const trySaveSheet = (sheetName?: string) => {
  saveSheetModal.value?.trySaveSheet(sheetName);
};

const { height: queryListHeight } = useElementSize(queryListRef);

watch(queryListHeight, () => {
  // Always scroll to the bottom as if we are in a real terminal.
  requestAnimationFrame(() => {
    const container = queryListContainerRef.value;
    if (container) {
      container.scrollTo(0, container.scrollHeight);
    }
  });
});
</script>
