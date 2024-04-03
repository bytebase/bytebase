<!-- eslint-disable vue/no-v-html -->
<template>
  <div class="relative w-full h-full flex flex-col justify-start items-start">
    <div class="w-full px-1">
      <SearchBox
        v-model:value="state.search"
        size="small"
        :placeholder="$t('sql-editor.search-history')"
        style="max-width: 100%"
      />
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

    <MaskSpinner
      v-show="isFetching && queryHistoryList.length === 0"
      class="!bg-white/75"
    />
  </div>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import dayjs from "dayjs";
import { escape, uniqBy } from "lodash-es";
import { useDialog } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { SearchBox } from "@/components/v2";
import {
  pushNotification,
  useSQLEditorQueryHistoryStore,
  useSQLEditorTabStore,
} from "@/store";
import type { SQLEditorTab } from "@/types";
import type { QueryHistory } from "@/types/proto/v1/sql_service";
import { getHighlightHTMLByKeyWords, defer } from "@/utils";

interface State {
  search: string;
  currentActionHistory: QueryHistory | null;
}

const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const queryHistoryStore = useSQLEditorQueryHistoryStore();
const { isFetching, queryHistoryList } = storeToRefs(queryHistoryStore);
const $d = useDialog();

const state = reactive<State>({
  search: "",
  currentActionHistory: null,
});

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});

const data = computed(() => {
  const tempData = queryHistoryList.value.filter((history) => {
    let t = false;

    if (history.statement.includes(state.search)) {
      t = true;
    }

    return t;
  });
  const distinctData = uniqBy(tempData, (history) => history.statement);

  return distinctData.map((history) => {
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
  if (isFetching.value) {
    return "";
  }
  if (queryHistoryList.value.length === 0) {
    return t("sql-editor.no-history-found");
  }

  return "";
});

const handleCopy = (history: QueryHistory) => {
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

const confirmOverrideCurrentStatement = () => {
  const d = defer<boolean>();
  $d.warning({
    title: t("common.warning"),
    content: () => t("sql-editor.override-current-statement"),
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    onNegativeClick: () => {
      d.resolve(false);
    },
    onPositiveClick: () => {
      d.resolve(true);
    },
  });
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
    if (tab.statement) {
      if (!(await confirmOverrideCurrentStatement())) {
        return;
      }
    }
    tab.statement = statement;
  };

  if (tab) {
    openInCurrentTab(tab);
  } else {
    openInNewTab();
  }
};

onMounted(async () => {
  await queryHistoryStore.fetchQueryHistoryList();
});
</script>
