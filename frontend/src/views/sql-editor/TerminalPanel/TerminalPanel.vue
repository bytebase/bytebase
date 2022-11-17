<template>
  <div
    class="flex h-full w-full flex-col justify-start items-start overflow-hidden"
  >
    <EditorAction @save-sheet="trySaveSheet" />

    <ConnectionPathBar />

    <template v-if="!tabStore.isDisconnected">
      <div ref="queryListContainerRef" class="w-full flex-1 overflow-y-auto">
        <div
          ref="queryListRef"
          class="w-full flex flex-col"
          :data-height="queryListSize.height"
        >
          <div v-for="(query, i) in state.queryList" :key="i" class="relative">
            <CompactSQLEditor
              v-model:sql="query.sql"
              class="border-b"
              :readonly="query !== currentQuery"
              @save-sheet="trySaveSheet"
              @execute="handleExecute"
            />
            <div
              v-if="query.queryResult"
              class="max-h-[21rem] overflow-y-auto border-b"
            >
              <TableView :query-result="query.queryResult.data" />
            </div>

            <div
              v-if="query.isExecutingSQL"
              class="absolute inset-0 bg-white/50 flex justify-center items-center"
            >
              <BBSpin />
            </div>
          </div>
        </div>
      </div>
    </template>
    <template v-else>
      <ConnectionHolder />
    </template>

    <SaveSheetModal ref="saveSheetModal" />
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watch } from "vue";
import { useElementSize } from "@vueuse/core";

import { ExecuteConfig, ExecuteOption, SQLResultSet } from "@/types";
import { useSQLEditorStore, useTabStore } from "@/store";
import CompactSQLEditor from "./CompactSQLEditor.vue";
import EditorAction from "../EditorCommon/EditorAction.vue";
import ConnectionPathBar from "../EditorCommon/ConnectionPathBar.vue";
import ConnectionHolder from "../EditorCommon/ConnectionHolder.vue";
import TableView from "../EditorCommon/TableView.vue";
import SaveSheetModal from "../EditorCommon/SaveSheetModal.vue";

type QueryItem = {
  sql: string;
  executeParams?: {
    query: string;
    config: ExecuteConfig;
    option?: Partial<ExecuteOption>;
  };
  isExecutingSQL: boolean;
  queryResult?: SQLResultSet;
};

type LocalState = {
  queryList: QueryItem[];
};

const createEmptyQueryItem = (): QueryItem => ({
  sql: "",
  isExecutingSQL: false,
});

const tabStore = useTabStore();

const state = reactive<LocalState>({
  queryList: [
    {
      // The first query is created from the sheet's SQL statement.
      sql: tabStore.currentTab.statement,
      isExecutingSQL: false,
    },
  ],
});

const queryListContainerRef = ref<HTMLDivElement>();
const queryListRef = ref<HTMLDivElement>();

const currentQuery = computed(
  () => state.queryList[state.queryList.length - 1]
);

const handleExecute = async (
  query: string,
  config: ExecuteConfig,
  option?: ExecuteOption
) => {
  const queryItem = currentQuery.value;
  if (queryItem.isExecutingSQL) {
    return;
  }

  queryItem.executeParams = { query, config, option };
  queryItem.isExecutingSQL = true;
  try {
    const result = await useSQLEditorStore().executeAdminQuery({
      statement: query,
    });
    queryItem.queryResult = result;
    state.queryList.push(createEmptyQueryItem());
  } finally {
    queryItem.isExecutingSQL = false;
  }
};

const saveSheetModal = ref<InstanceType<typeof SaveSheetModal>>();

const trySaveSheet = (sheetName?: string) => {
  saveSheetModal.value?.trySaveSheet(sheetName);
};

const queryListSize = useElementSize(queryListRef);

watch(queryListSize.height, () => {
  // Always scroll to the bottom as if we are in a real terminal.
  requestAnimationFrame(() => {
    const container = queryListContainerRef.value;
    if (container) {
      container.scrollTo(0, container.scrollHeight);
    }
  });
});
</script>
