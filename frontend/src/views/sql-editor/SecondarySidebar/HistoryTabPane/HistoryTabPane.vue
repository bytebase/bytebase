<!-- eslint-disable vue/no-v-html -->
<template>
  <div
    class="relative py-0.5 w-full h-full flex flex-col justify-start items-start"
  >
    <div class="w-full px-2 pt-1.5">
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
          <div class="flex items-start gap-x-1">
            <HistoryConnectionIcon :query-history="history" />
            <span class="text-xs text-gray-500">
              {{ titleOfQueryHistory(history) }}
            </span>
          </div>
          <span
            class="rounded text-gray-500 hover:text-gray-700 hover:bg-gray-200"
          >
            <heroicons-outline:clipboard-document
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
      v-show="isFetching && queryHistoryList.length === 0"
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
import { storeToRefs } from "pinia";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useSQLEditorQueryHistoryStore,
  useSQLEditorTabStore,
} from "@/store";
import { SQLEditorQueryHistory } from "@/types";
import { getHighlightHTMLByKeyWords } from "@/utils";
import HistoryConnectionIcon from "./HistoryConnectionIcon.vue";

interface State {
  search: string;
  currentActionHistory: SQLEditorQueryHistory | null;
}

const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const queryHistoryStore = useSQLEditorQueryHistoryStore();
const { isFetching, queryHistoryList } = storeToRefs(queryHistoryStore);

const state = reactive<State>({
  search: "",
  currentActionHistory: null,
});

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});

const data = computed(() => {
  const tempData =
    queryHistoryList.value.length > 0
      ? queryHistoryList.value.filter((history) => {
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

const titleOfQueryHistory = (history: SQLEditorQueryHistory) => {
  return dayjs(history.createTime).format("YYYY-MM-DD HH:mm:ss");
};

const notifyMessage = computed(() => {
  if (isFetching.value) {
    return "";
  }
  if (queryHistoryList.value.length === 0) {
    return t("sql-editor.no-history-found");
  }

  return "";
});

const handleCopy = (history: SQLEditorQueryHistory) => {
  if (!isSupported.value) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Copy to clipboard is not enabled in your browser.",
    });
    return;
  }

  state.currentActionHistory = history;
  copyTextToClipboard(history.statement);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-editor.notify.copy-code-succeed"),
  });
};

const handleQueryHistoryClick = async (queryHistory: SQLEditorQueryHistory) => {
  const { instance, database, statement } = queryHistory;

  // Open a new tab with the connection and statement.
  tabStore.addTab({
    title: `Query history at ${titleOfQueryHistory(queryHistory)}`,
    connection: {
      instance,
      database,
    },
    statement,
  });
};

onMounted(async () => {
  await queryHistoryStore.fetchQueryHistoryList();
});
</script>
