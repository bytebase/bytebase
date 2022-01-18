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
        class="w-full px-1 pr-2 py-2 border-b flex flex-col justify-start items-start"
      >
        <div class="w-full flex flex-row justify-between items-center">
          <span class="text-xs text-gray-500">{{ history.createdAt }}</span>
          <NDropdown
            trigger="click"
            :options="historyDropdownOptions"
            @select="(key: string) => handleActionBtnClick(key, history)"
            @clickoutside="handleActionBtnOutsideClick"
          >
            <NButton text>
              <template #icon>
                <heroicons-outline:dots-horizontal
                  class="h-4 w-4 text-gray-500"
                />
              </template>
            </NButton>
          </NDropdown>
        </div>
        <p
          class="max-w-full mt-2 mb-1 text-sm break-words font-mono cursor-pointer line-clamp-3 hover:bg-gray-300"
          @click="handleQueryHistoryClick(history)"
        >
          {{ history.statement }}
        </p>
      </div>
    </div>

    <div
      v-show="notifyMessage"
      class="absolute w-full h-full flex justify-center items-center transition-all bg-transparent"
    >
      {{ notifyMessage }}
    </div>

    <div
      v-show="isLoading && queryHistoryList.length === 0"
      class="absolute w-full h-full flex justify-center items-center"
    >
      <BBSpin :title="$t('common.loading')" />
    </div>
  </div>

  <BBModal
    v-if="state.isShowDeletingHint"
    :title="$t('common.tips')"
    @close="handleHintClose"
  >
    <DeleteHint
      :content="$t('sql-editor.hint-tips.confirm-to-delete-this-history')"
      @close="handleHintClose"
      @confirm="handleDeleteHistory"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  useNamespacedActions,
  useNamespacedState,
} from "vuex-composition-helpers";
import {
  EditorSelectorActions,
  QueryHistory,
  SqlEditorActions,
  SqlEditorState,
} from "../../../types";
import DeleteHint from "./DeleteHint.vue";

interface State {
  search: string;
  isShowDeletingHint: boolean;
  currentActionHistory: QueryHistory | null;
}

const { t } = useI18n();

const { queryHistoryList, isFetchingQueryHistory: isLoading } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "queryHistoryList",
    "isFetchingQueryHistory",
  ]);
const { deleteQueryHistory, setShouldSetContent } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
    "deleteQueryHistory",
    "setShouldSetContent",
  ]);
const { updateActiveTab } = useNamespacedActions<EditorSelectorActions>(
  "editorSelector",
  ["updateActiveTab"]
);

const state = reactive<State>({
  search: "",
  isShowDeletingHint: false,
  currentActionHistory: null,
});

const data = computed(() => {
  const temp =
    queryHistoryList.value && queryHistoryList.value.length > 0
      ? queryHistoryList.value.filter((history) => {
          let t = false;

          if (history.statement.includes(state.search)) {
            t = true;
          }

          return t;
        })
      : [];
  return temp;
});

const notifyMessage = computed(() => {
  if (isLoading.value) {
    return "";
  }
  if (queryHistoryList.value.length === null) {
    return t("sql-editor.no-history-found");
  }

  return "";
});

const historyDropdownOptions = computed(() => [
  {
    label: t("common.delete"),
    key: "delete",
  },
]);

const handleActionBtnClick = (key: string, history: QueryHistory) => {
  state.currentActionHistory = history;

  if (key === "delete") {
    state.isShowDeletingHint = true;
  }
};

const handleActionBtnOutsideClick = () => {
  state.currentActionHistory = null;
};

const handleHintClose = () => {
  state.currentActionHistory = null;
  state.isShowDeletingHint = false;
};

const handleDeleteHistory = () => {
  if (state.currentActionHistory) {
    deleteQueryHistory(state.currentActionHistory.id);
  }
  state.isShowDeletingHint = false;
};

const handleQueryHistoryClick = async (queryHistory: QueryHistory) => {
  updateActiveTab({
    queryStatement: queryHistory.statement,
    selectedStatement: "",
  });
  setShouldSetContent(true);
};
</script>
