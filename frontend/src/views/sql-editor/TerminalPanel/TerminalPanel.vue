<template>
  <div
    class="flex h-full w-full flex-col justify-start items-start overflow-hidden"
  >
    <EditorAction />

    <div
      v-if="isProtectedEnvironment"
      class="w-full py-1 px-4 bg-warning text-white"
    >
      {{ $t("sql-editor.sql-execute-in-protected-environment") }}
    </div>

    <template v-if="!tabStore.isDisconnected">
      <div ref="queryListContainerRef" class="w-full flex-1 overflow-y-auto">
        <div class="w-full flex flex-col">
          <template v-for="(query, i) in state.queryList" :key="i">
            <CompactSQLEditor
              v-model:sql="query.sql"
              class="border-b"
              :readonly="query !== currentQuery"
              @execute="handleExecute"
            />
            <div
              v-if="query.queryResult"
              class="max-h-[20rem] overflow-y-auto border-b"
            >
              <TableView
                :query-result="query.queryResult.data as [string[], string[], any[][]]"
              />
            </div>
          </template>
        </div>
      </div>
    </template>
    <template v-else>
      <ConnectionHolder />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";

import { ExecuteConfig, ExecuteOption, SQLResultSet } from "@/types";
import { useSQLEditorStore, useTabStore, useInstanceStore } from "@/store";
import CompactSQLEditor from "./CompactSQLEditor.vue";
import EditorAction from "../EditorPanel/EditorAction.vue";
import ConnectionHolder from "../EditorPanel/ConnectionHolder.vue";
import TableView from "./TableView.vue";

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
const instanceStore = useInstanceStore();

const state = reactive<LocalState>({
  queryList: [createEmptyQueryItem()],
});

const queryListContainerRef = ref<HTMLDivElement>();

const isProtectedEnvironment = computed(() => {
  const { instanceId } = tabStore.currentTab.connection;
  const instance = instanceStore.getInstanceById(instanceId);
  return instance.environment.tier === "PROTECTED";
});

const currentQuery = computed(
  () => state.queryList[state.queryList.length - 1]
);

const handleExecute = async (
  query: string,
  config: ExecuteConfig,
  option?: ExecuteOption
) => {
  const queryItem = currentQuery.value;
  queryItem.executeParams = { query, config, option };
  queryItem.isExecutingSQL = true;
  try {
    const result = await useSQLEditorStore().executeAdminQuery({
      statement: query,
    });
    queryItem.queryResult = result;
    state.queryList.push(createEmptyQueryItem());

    setTimeout(() => {
      const container = queryListContainerRef.value;
      if (container) {
        container.scrollTo(0, container.scrollHeight);
      }
    }, 100);
  } finally {
    queryItem.isExecutingSQL = false;
  }
};
</script>
