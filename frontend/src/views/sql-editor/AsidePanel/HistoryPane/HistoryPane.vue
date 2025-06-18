<!-- eslint-disable vue/no-v-html -->
<template>
  <div class="relative w-full h-full flex flex-col justify-start items-start">
    <div class="w-full px-1">
      <SearchBox
        :value="state.search"
        size="small"
        :placeholder="$t('sql-editor.search-history-by-statement')"
        style="max-width: 100%"
        @update:value="onSearchUpdate"
      />
    </div>
    <div class="w-full flex flex-col justify-start items-start overflow-y-auto">
      <div
        v-for="history in queryHistoryData.queryHistories"
        :key="history.name"
        class="w-full p-2 space-y-1 border-b flex flex-col justify-start items-start cursor-pointer hover:bg-gray-50"
        @click="handleQueryHistoryClick(history)"
      >
        <div class="w-full flex flex-row justify-between items-center">
          <div class="flex items-start gap-x-1">
            <span class="text-xs text-gray-500">
              {{ titleOfQueryHistory(history) }}
            </span>
          </div>
          <CopyButton
            quaternary
            :text="false"
            :content="history.statement"
            @click.stop
          />
        </div>
        <p
          class="max-w-full text-xs break-words font-mono line-clamp-3"
          v-html="getFormattedStatement(history.statement)"
        ></p>
      </div>
      <div
        v-if="queryHistoryData.nextPageToken"
        class="w-full flex flex-col items-center my-2"
      >
        <NButton
          quaternary
          :size="'small'"
          :loading="state.loading"
          @click="fetchQueryHistoryListList"
        >
          <span class="textinfolabel">
            {{ $t("common.load-more") }}
          </span>
        </NButton>
      </div>
    </div>

    <template v-if="queryHistoryData.queryHistories.length === 0">
      <MaskSpinner v-if="state.loading" class="!bg-white/75" />
      <div
        v-else
        class="w-full flex items-center justify-center py-8 textinfolabel"
      >
        {{ $t("sql-editor.no-history-found") }}
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import dayjs from "dayjs";
import { escape } from "lodash-es";
import { NButton, useDialog } from "naive-ui";
import { computed, h, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { SearchBox } from "@/components/v2";
import { CopyButton } from "@/components/v2";
import {
  useSQLEditorQueryHistoryStore,
  useSQLEditorTabStore,
  useSQLEditorStore,
  type QueryHistoryFilter,
} from "@/store";
import {
  DEBOUNCE_SEARCH_DELAY,
  getDateForPbTimestamp,
  type SQLEditorTab,
} from "@/types";
import type { QueryHistory } from "@/types/proto/v1/sql_service";
import { getHighlightHTMLByKeyWords, defer } from "@/utils";

interface State {
  search: string;
  loading: boolean;
}

const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();
const queryHistoryStore = useSQLEditorQueryHistoryStore();
const $d = useDialog();

const state = reactive<State>({
  search: "",
  loading: false,
});

const historyQuery = computed((): QueryHistoryFilter => {
  const tab = tabStore.currentTab;
  return {
    database: tab?.connection.database,
    project: editorStore.project,
    statement: state.search,
  };
});

const onSearchUpdate = async (search: string) => {
  queryHistoryStore.resetPageToken(historyQuery.value);
  state.search = search;
  await fetchQueryHistoryListList();
};

const queryHistoryData = computed(() =>
  queryHistoryStore.getQueryHistoryList(historyQuery.value)
);

const fetchQueryHistoryListList = useDebounceFn(async () => {
  state.loading = true;
  try {
    await queryHistoryStore.fetchQueryHistoryList(historyQuery.value);
  } finally {
    state.loading = false;
  }
}, DEBOUNCE_SEARCH_DELAY);

watch(
  () => historyQuery.value,
  async () => {
    if (queryHistoryData.value.queryHistories.length === 0) {
      await fetchQueryHistoryListList();
    }
  },
  {
    immediate: true,
    deep: true,
  }
);

const getFormattedStatement = (statement: string) => {
  return state.search
    ? getHighlightHTMLByKeyWords(escape(statement), escape(state.search))
    : escape(statement);
};

const titleOfQueryHistory = (history: QueryHistory) => {
  return dayjs(getDateForPbTimestamp(history.createTime)).format(
    "YYYY-MM-DD HH:mm:ss"
  );
};

const confirmOverrideCurrentStatement = (): Promise<"CANCEL" | "OVERRIDE"> => {
  const d = defer<"CANCEL" | "OVERRIDE">();
  const dialog = $d.warning({
    title: t("common.warning"),
    content: t("sql-editor.current-editing-statement-is-not-empty"),
    contentClass: "whitespace-pre-wrap",
    style: "z-index: 100000",
    closable: false,
    closeOnEsc: false,
    maskClosable: false,
    action: () => {
      const buttons = [
        h(
          NButton,
          { size: "small", onClick: () => d.resolve("CANCEL") },
          { default: () => t("common.cancel") }
        ),
        h(
          NButton,
          {
            size: "small",
            type: "warning",
            onClick: () => d.resolve("OVERRIDE"),
          },
          { default: () => t("common.override") }
        ),
      ];
      return h(
        "div",
        { class: "flex items-center justify-end gap-2" },
        buttons
      );
    },
  });
  d.promise.then(() => dialog.destroy());
  return d.promise;
};

const handleQueryHistoryClick = async (queryHistory: QueryHistory) => {
  const { statement } = queryHistory;

  // Open a new tab with the connection and statement.
  const tab = tabStore.currentTab;

  const openInNewTab = () => {
    tabStore.addTab(
      {
        title: `Query history at ${titleOfQueryHistory(queryHistory)}`,
        statement,
      },
      /* beside */ true
    );
  };
  const openInCurrentTab = async (tab: SQLEditorTab) => {
    if (!tab.statement) {
      tab.statement = statement;
      return;
    }
    const choice = await confirmOverrideCurrentStatement();
    if (choice === "CANCEL") {
      return;
    }
    if (choice === "OVERRIDE") {
      tabStore.updateCurrentTab({
        statement,
      });
      return;
    }
  };

  if (tab) {
    openInCurrentTab(tab);
  } else {
    openInNewTab();
  }
};
</script>
