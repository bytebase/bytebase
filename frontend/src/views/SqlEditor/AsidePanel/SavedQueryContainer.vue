<template>
  <div
    class="relative p-2 space-y-2 w-full h-full flex flex-col justify-start items-start"
  >
    <div class="w-full">
      <NInput
        v-model:value="state.search"
        :placeholder="$t('sql-editor.search-saved-queries')"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
    </div>
    <div class="w-full flex flex-col justify-start items-start overflow-y-auto">
      <div
        v-for="query in data"
        :key="query.id"
        class="w-full px-1 pr-2 py-2 border-b flex flex-col justify-start items-start"
        :class="`${
          currentTab.currentQueryId === query.id
            ? 'bg-gray-100 rounded border-b-0'
            : ''
        }`"
        @click="handleSavedQueryClick(query)"
      >
        <div class="pb-1 w-full flex flex-row justify-between items-center">
          <div
            class="w-full mr-2"
            @dblclick="handleEditSavedQuery(query.id, query.name)"
          >
            <input
              v-if="state.editingSavedQueryId === query.id"
              ref="queryNameInputerRef"
              v-model="state.currentSavedQueryName"
              type="text"
              class="rounded px-2 py-0 text-sm w-full"
              @blur="handleSavedQueryNameChanged"
              @keyup.enter="handleSavedQueryNameChanged"
              @keyup.esc="handleCancelEdit"
            />
            <span v-else class="text-sm" v-html="query.formatedName"></span>
          </div>
          <NDropdown
            trigger="click"
            :options="actionDropdownOptions"
            @select="(key: string) => handleActionBtnClick(key, query)"
            @clickoutside="handleActionBtnOutsideClick"
          >
            <NButton text @click.stop>
              <template #icon>
                <heroicons-outline:dots-horizontal
                  class="h-4 w-4 text-gray-500 flex-shrink-0"
                />
              </template>
            </NButton>
          </NDropdown>
        </div>
        <p
          class="max-w-full text-gray-400 break-words font-mono truncate"
          style="font-size: 10px"
          v-html="query.formatedStatement"
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
      v-show="isLoading && savedQueryList.length === 0"
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
      :content="$t('sql-editor.hint-tips.confirm-to-delete-this-saved-query')"
      @close="handleHintClose"
      @confirm="handleDeleteSavedQuery"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { escape } from "lodash-es";
import { computed, reactive, ref, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import {
  useNamespacedActions,
  useNamespacedGetters,
  useNamespacedState,
} from "vuex-composition-helpers";

import {
  TabActions,
  TabGetters,
  TabState,
  SavedQuery,
  SqlEditorActions,
  SqlEditorState,
} from "../../../types";
import { getHighlightHTMLByKeyWords } from "../../../utils";
import DeleteHint from "./DeleteHint.vue";

interface State {
  search: string;
  isShowDeletingHint: boolean;
  currentSavedQueryName: string;
  editingSavedQueryId: number | null;
  currentActionSavedQuery: SavedQuery | null;
}

const { t } = useI18n();

const { savedQueryList, isFetchingSavedQueries: isLoading } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "savedQueryList",
    "isFetchingSavedQueries",
  ]);
const { tabList } = useNamespacedState<TabState>("tab", ["tabList"]);
const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);
const { deleteSavedQuery, setShouldSetContent, patchSavedQuery } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
    "deleteSavedQuery",
    "setShouldSetContent",
    "patchSavedQuery",
  ]);
const { updateActiveTab, setCurrentTabId } = useNamespacedActions<TabActions>(
  "tab",
  ["updateActiveTab", "setCurrentTabId"]
);

const state = reactive<State>({
  search: "",
  currentSavedQueryName: "",
  editingSavedQueryId: null,
  isShowDeletingHint: false,
  currentActionSavedQuery: null,
});

const queryNameInputerRef = ref<HTMLInputElement>();

const data = computed(() => {
  const tempData =
    savedQueryList.value && savedQueryList.value.length > 0
      ? savedQueryList.value.filter((savedQuery) => {
          let t = false;

          if (savedQuery.name.includes(state.search)) {
            t = true;
          }
          if (savedQuery.statement.includes(state.search)) {
            t = true;
          }

          return t;
        })
      : [];

  return tempData.map((savedQuery) => {
    return {
      ...savedQuery,
      formatedName: state.search
        ? getHighlightHTMLByKeyWords(
            escape(savedQuery.name),
            escape(state.search)
          )
        : escape(savedQuery.name),
      formatedStatement: state.search
        ? getHighlightHTMLByKeyWords(
            escape(savedQuery.statement),
            escape(state.search)
          )
        : escape(savedQuery.statement),
    };
  });
});

const notifyMessage = computed(() => {
  if (isLoading.value) {
    return "";
  }
  if (savedQueryList.value.length === null) {
    return t("sql-editor.no-saved-query-found");
  }

  return "";
});

const actionDropdownOptions = computed(() => [
  {
    label: t("common.delete"),
    key: "delete",
  },
]);

const handleEditSavedQuery = (id: number, name: string) => {
  state.editingSavedQueryId = id;
  state.currentSavedQueryName = name;
  nextTick(() => {
    queryNameInputerRef.value?.focus();
  });
};

const handleCancelEdit = () => {
  state.editingSavedQueryId = null;
  state.currentSavedQueryName = "";
};

const handleSavedQueryNameChanged = () => {
  if (state.editingSavedQueryId) {
    patchSavedQuery({
      id: state.editingSavedQueryId,
      name: state.currentSavedQueryName,
    });

    if (currentTab.value.currentQueryId === state.editingSavedQueryId) {
      updateActiveTab({
        label: state.currentSavedQueryName,
      });
    }
    handleCancelEdit();
  }
};

const handleActionBtnClick = (key: string, savedQuery: SavedQuery) => {
  state.currentActionSavedQuery = savedQuery;

  if (key === "delete") {
    state.isShowDeletingHint = true;
  }
};

const handleActionBtnOutsideClick = () => {
  state.currentActionSavedQuery = null;
};

const handleHintClose = () => {
  state.currentActionSavedQuery = null;
  state.isShowDeletingHint = false;
};

const handleDeleteSavedQuery = () => {
  if (state.currentActionSavedQuery) {
    deleteSavedQuery(state.currentActionSavedQuery.id);

    if (currentTab.value.currentQueryId === state.currentActionSavedQuery.id) {
      updateActiveTab({
        currentQueryId: undefined,
      });
    }
  }
  state.isShowDeletingHint = false;
};

const handleSavedQueryClick = (savedQuery: SavedQuery) => {
  if (currentTab.value.currentQueryId !== savedQuery.id) {
    for (const tab of tabList.value) {
      if (tab.currentQueryId === savedQuery.id) {
        setCurrentTabId(tab.id);
        setShouldSetContent(true);
        return;
      }
    }

    updateActiveTab({
      label: savedQuery.name,
      queryStatement: savedQuery.statement,
      selectedStatement: "",
      currentQueryId: savedQuery.id,
    });
    setShouldSetContent(true);
  }
};
</script>
