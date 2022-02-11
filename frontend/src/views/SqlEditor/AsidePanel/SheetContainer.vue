<template>
  <div
    class="relative p-2 space-y-2 w-full h-full flex flex-col justify-start items-start"
  >
    <div class="w-full">
      <NInput
        v-model:value="state.search"
        :placeholder="$t('sql-editor.search-sheet')"
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
        class="w-full px-1 pr-2 py-2 border-b flex flex-col justify-start items-start cursor-pointer"
        :class="`${
          currentTab.sheetId === query.id
            ? 'bg-gray-100 rounded border-b-0'
            : ''
        }`"
        @click="handleSheetClick(query)"
      >
        <div class="pb-1 w-full flex flex-row justify-between items-center">
          <div
            class="w-full mr-2"
            @dblclick="handleEditSheet(query.id, query.name)"
          >
            <input
              v-if="state.editingSheetId === query.id"
              ref="queryNameInputerRef"
              v-model="state.currentSheetName"
              type="text"
              class="rounded px-2 py-0 text-sm w-full"
              @blur="handleSheetNameChanged"
              @keyup.enter="handleSheetNameChanged"
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
      v-show="isLoading && sheetList.length === 0"
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
      :content="$t('sql-editor.hint-tips.confirm-to-delete-this-sheet')"
      @close="handleHintClose"
      @confirm="handleDeleteSheet"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  useNamespacedActions,
  useNamespacedGetters,
  useNamespacedState,
} from "vuex-composition-helpers";

import {
  TabActions,
  TabGetters,
  TabState,
  SheetActions,
  SheetState,
  Sheet,
  SqlEditorActions,
} from "../../../types";
import { getHighlightHTMLByKeyWords } from "../../../utils";
import DeleteHint from "./DeleteHint.vue";

interface State {
  search: string;
  isShowDeletingHint: boolean;
  currentSheetName: string;
  editingSheetId: number | null;
  currentActionSheet: Sheet | null;
}

const { t } = useI18n();
const router = useRouter();

const { sheetList, isFetchingSheet: isLoading } =
  useNamespacedState<SheetState>("sheet", ["sheetList", "isFetchingSheet"]);
const { tabList } = useNamespacedState<TabState>("tab", ["tabList"]);
const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);

// actions
const { setShouldSetContent, setActiveConnectionByTab } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
    "setShouldSetContent",
    "setActiveConnectionByTab",
  ]);
const { updateActiveTab, setActiveTabId, addTab } =
  useNamespacedActions<TabActions>("tab", [
    "updateActiveTab",
    "setActiveTabId",
    "addTab",
  ]);
const { deleteSheet, patchSheetById } = useNamespacedActions<SheetActions>(
  "sheet",
  ["deleteSheet", "patchSheetById"]
);

const state = reactive<State>({
  search: "",
  currentSheetName: "",
  editingSheetId: null,
  isShowDeletingHint: false,
  currentActionSheet: null,
});

const queryNameInputerRef = ref<HTMLInputElement>();

const data = computed(() => {
  const tempData =
    sheetList.value && sheetList.value.length > 0
      ? sheetList.value.filter((sheet) => {
          let t = false;

          if (sheet.name.includes(state.search)) {
            t = true;
          }
          if (sheet.statement.includes(state.search)) {
            t = true;
          }

          return t;
        })
      : [];

  return tempData.map((sheet) => {
    return {
      ...sheet,
      formatedName: state.search
        ? getHighlightHTMLByKeyWords(sheet.name, state.search)
        : sheet.name,
      formatedStatement: state.search
        ? getHighlightHTMLByKeyWords(sheet.statement, state.search)
        : sheet.statement,
    };
  });
});

const notifyMessage = computed(() => {
  if (isLoading.value) {
    return "";
  }
  if (sheetList.value.length === null) {
    return t("sql-editor.no-sheet-found");
  }

  return "";
});

const actionDropdownOptions = computed(() => [
  {
    label: t("common.delete"),
    key: "delete",
  },
]);

const handleEditSheet = (id: number, name: string) => {
  state.editingSheetId = id;
  state.currentSheetName = name;
  nextTick(() => {
    queryNameInputerRef.value?.focus();
  });
};

const handleCancelEdit = () => {
  state.editingSheetId = null;
  state.currentSheetName = "";
};

const handleSheetNameChanged = () => {
  if (state.editingSheetId) {
    patchSheetById({
      id: state.editingSheetId,
      name: state.currentSheetName,
    });

    if (currentTab.value.sheetId === state.editingSheetId) {
      updateActiveTab({
        label: state.currentSheetName,
      });
    }
    handleCancelEdit();
  }
};

const handleActionBtnClick = (key: string, sheet: Sheet) => {
  state.currentActionSheet = sheet;

  if (key === "delete") {
    state.isShowDeletingHint = true;
  }
};

const handleActionBtnOutsideClick = () => {
  state.currentActionSheet = null;
};

const handleHintClose = () => {
  state.currentActionSheet = null;
  state.isShowDeletingHint = false;
};

const handleDeleteSheet = () => {
  if (state.currentActionSheet) {
    deleteSheet(state.currentActionSheet.id);

    if (currentTab.value.sheetId === state.currentActionSheet.id) {
      updateActiveTab({
        sheetId: undefined,
      });
    }
  }
  state.isShowDeletingHint = false;
};

const handleSheetClick = async (sheet: Sheet) => {
  if (currentTab.value.sheetId !== sheet.id) {
    // open opened tab first
    for (const tab of tabList.value) {
      if (tab.sheetId === sheet.id) {
        setActiveTabId(tab.id);
        setActiveConnectionByTab(router);
        setShouldSetContent(true);
        return;
      }
    }

    // otherwise, add new tab
    addTab({
      label: sheet.name,
      queryStatement: sheet.statement,
      selectedStatement: "",
      sheetId: sheet.id,
    });
    setActiveConnectionByTab(router);
    setShouldSetContent(true);
  }
};
</script>
