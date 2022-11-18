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
        <div v-for="(query, i) in state.queryList" :key="i" class="relative">
          <CompactSQLEditor
            v-model:sql="query.sql"
            class="border-b min-h-[2rem]"
            :readonly="query !== currentQuery"
            @save-sheet="trySaveSheet"
            @execute="handleExecute"
          />
          <div
            v-if="query.queryResult"
            class="max-h-[20rem] overflow-y-auto border-b"
          >
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
import { computed, reactive, ref, watch } from "vue";
import { useElementSize } from "@vueuse/core";

import { ExecuteConfig, ExecuteOption, SQLResultSet } from "@/types";
import { useTabStore } from "@/store";
import CompactSQLEditor from "./CompactSQLEditor.vue";
import EditorAction from "../EditorCommon/EditorAction.vue";
import ConnectionPathBar from "../EditorCommon/ConnectionPathBar.vue";
import ConnectionHolder from "../EditorCommon/ConnectionHolder.vue";
import TableView from "../EditorCommon/TableView.vue";
import SaveSheetModal from "../EditorCommon/SaveSheetModal.vue";
import { useExecuteSQL } from "@/composables/useExecuteSQL";

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

const { execute } = useExecuteSQL();

const handleExecute = async (
  query: string,
  config: ExecuteConfig,
  option?: ExecuteOption
) => {
  const queryItem = currentQuery.value;
  if (queryItem.isExecutingSQL) {
    return;
  }

  try {
    queryItem.executeParams = { query, config, option };
    queryItem.isExecutingSQL = true;
    const sqlResultSet = await execute(query, config, option);

    queryItem.queryResult = sqlResultSet;
    state.queryList.push(createEmptyQueryItem());
  } finally {
    queryItem.isExecutingSQL = false;
  }

  // try {
  //   const result = await useSQLEditorStore().executeAdminQuery({
  //     statement: query,
  //   });
  //   queryItem.queryResult = result;
  //   state.queryList.push(createEmptyQueryItem());
  // } finally {
  //   queryItem.isExecutingSQL = false;
  // }
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
