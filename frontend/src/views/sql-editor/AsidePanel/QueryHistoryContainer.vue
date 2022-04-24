<template>
  <div
    class="relative p-2 space-y-2 w-full h-full flex flex-col justify-start items-start"
  >
    <div class="w-full">
      <NInput
        v-model:value="state.search"
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
        :key="history.id"
        class="w-full px-1 pr-2 py-2 border-b flex flex-col justify-start items-start cursor-pointer hover:bg-gray-100"
        @click="handleQueryHistoryClick(history)"
      >
        <div class="w-full flex flex-row justify-between items-center">
          <span class="text-xs text-gray-500">{{ history.createdAt }}</span>
          <NDropdown
            trigger="click"
            :options="actionDropdownOptions"
            @select="(key: string) => handleActionBtnClick(key, history)"
            @clickoutside="handleActionBtnOutsideClick"
          >
            <NButton text @click.stop>
              <template #icon>
                <heroicons-outline:dots-horizontal
                  class="h-4 w-4 text-gray-500"
                />
              </template>
            </NButton>
          </NDropdown>
        </div>
        <p
          class="max-w-full mt-2 mb-1 text-sm break-words font-mono line-clamp-3"
          v-html="history.formatedStatement"
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
import { escape } from "lodash-es";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useClipboard } from "@vueuse/core";

import { useDialog } from "naive-ui";

import { pushNotification, useTabStore, useSQLEditorStore } from "@/store";
import { QueryHistory } from "@/types";
import { getHighlightHTMLByKeyWords } from "@/utils";

interface State {
  search: string;
  currentActionHistory: QueryHistory | null;
}

const { t } = useI18n();
const dialog = useDialog();
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();

const state = reactive<State>({
  search: "",
  currentActionHistory: null,
});

const { copy: copyTextToClipboard, isSupported: isCopySupported } =
  useClipboard();

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
      formatedStatement: state.search
        ? getHighlightHTMLByKeyWords(
            escape(history.statement),
            escape(state.search)
          )
        : escape(history.statement),
    };
  });
});

const notifyMessage = computed(() => {
  if (isLoading.value) {
    return "";
  }
  if (sqlEditorStore.queryHistoryList.length === null) {
    return t("sql-editor.no-history-found");
  }

  return "";
});

const actionDropdownOptions = computed(() => {
  const options = [];

  if (isCopySupported) {
    options.push({
      label: t("sql-editor.copy-code"),
      key: "copy",
    });
  }

  options.push({
    label: t("common.delete"),
    key: "delete",
  });

  return options;
});

const handleDeleteHistory = () => {
  if (state.currentActionHistory) {
    sqlEditorStore.deleteQueryHistory(state.currentActionHistory.id);
  }
};

const handleActionBtnClick = (key: string, history: QueryHistory) => {
  state.currentActionHistory = history;

  if (key === "delete") {
    const $dialog = dialog.create({
      title: t("sql-editor.hint-tips.confirm-to-delete-this-history"),
      type: "info",
      onPositiveClick() {
        handleDeleteHistory();
        $dialog.destroy();
      },
      async onNegativeClick() {
        state.currentActionHistory = null;
        $dialog.destroy();
      },
      negativeText: t("common.cancel"),
      positiveText: t("common.confirm"),
      showIcon: false,
    });
  } else if (key === "copy") {
    copyTextToClipboard(history.statement);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("sql-editor.notify.copy-code-succeed"),
    });
  }
};

const handleActionBtnOutsideClick = () => {
  state.currentActionHistory = null;
};

const handleQueryHistoryClick = async (queryHistory: QueryHistory) => {
  tabStore.addTab({
    statement: queryHistory.statement,
    selectedStatement: "",
  });
};
</script>
