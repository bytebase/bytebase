<!-- eslint-disable vue/no-v-html -->
<template>
  <div
    class="relative py-2 px-0.5 w-full h-full flex flex-col justify-start items-start"
  >
    <div class="w-full">
      <NInput
        v-model:value="state.search"
        size="small"
        :placeholder="$t('sql-editor.search-history')"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
    </div>
    <div class="w-full flex flex-col justify-start items-start overflow-y-auto">
      <div
        v-for="history in data"
        :key="history.name"
        class="w-full p-2 space-y-1 border-b flex flex-col justify-start items-start cursor-pointer hover:bg-gray-50"
        @click="handleQueryHistoryClick(history)"
      >
        <div class="w-full flex flex-row justify-between items-center">
          <span class="text-xs text-gray-500">
            {{ titleOfQueryHistory(history) }}
          </span>
          <span
            class="rounded text-gray-500 hover:text-gray-700 hover:bg-gray-200"
          >
            <heroicons-outline:clipboard
              class="w-4 h-4"
              @click.stop="handleCopy(history)"
            />
          </span>
        </div>
        <p
          class="max-w-full text-xs break-words font-mono line-clamp-3"
          v-html="history.formattedStatement"
        ></p>
      </div>
    </div>

    <div
      v-show="notifyMessage"
      class="absolute w-full h-full flex justify-center items-center transition-all bg-transparent"
    >
      {{ notifyMessage }}
    </div>

    <div
      v-show="isLoading && sqlEditorStore.queryHistoryList.length === 0"
      class="absolute w-full h-full flex justify-center items-center"
    >
      <BBSpin :title="$t('common.loading')" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import dayjs from "dayjs";
import { escape } from "lodash-es";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useTabStore,
  useSQLEditorStore,
  searchConnectionByName,
} from "@/store";
import { QueryHistory } from "@/types";
import { getHighlightHTMLByKeyWords } from "@/utils";

interface State {
  search: string;
  currentActionHistory: QueryHistory | null;
}

const { t } = useI18n();
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();

const state = reactive<State>({
  search: "",
  currentActionHistory: null,
});

const { copy: copyTextToClipboard } = useClipboard();

const isLoading = computed(() => {
  return sqlEditorStore.isFetchingQueryHistory;
});

const data = computed(() => {
  const tempData =
    sqlEditorStore.queryHistoryList &&
    sqlEditorStore.queryHistoryList.length > 0
      ? sqlEditorStore.queryHistoryList.filter((history) => {
          let t = false;

          if (history.statement.includes(state.search)) {
            t = true;
          }

          return t;
        })
      : [];

  return tempData.map((history) => {
    return {
      ...history,
      formattedStatement: state.search
        ? getHighlightHTMLByKeyWords(
            escape(history.statement),
            escape(state.search)
          )
        : escape(history.statement),
    };
  });
});

const titleOfQueryHistory = (history: QueryHistory) => {
  return dayjs(history.createTime).format("YYYY-MM-DD HH:mm:ss");
};

const notifyMessage = computed(() => {
  if (isLoading.value) {
    return "";
  }
  if (sqlEditorStore.queryHistoryList.length === null) {
    return t("sql-editor.no-history-found");
  }

  return "";
});

const handleCopy = (history: QueryHistory) => {
  state.currentActionHistory = history;
  copyTextToClipboard(history.statement);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-editor.notify.copy-code-succeed"),
  });
};

const handleQueryHistoryClick = async (queryHistory: QueryHistory) => {
  const { instanceId, databaseId, instanceName, databaseName, statement } =
    queryHistory;
  const connection = searchConnectionByName(
    String(instanceId),
    String(databaseId),
    instanceName,
    databaseName
  );

  // Open a new tab with the connection and statement.
  tabStore.addTab({
    name: `Query history at ${titleOfQueryHistory(queryHistory)}`,
  });
  tabStore.updateCurrentTab({
    connection,
    statement,
  });
};
</script>
