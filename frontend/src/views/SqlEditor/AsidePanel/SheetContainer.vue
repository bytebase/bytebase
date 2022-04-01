<template>
  <div
    class="relative p-2 space-y-2 w-full h-full flex flex-col justify-start items-start"
  >
    <div class="w-full">
      <NInput
        v-model:value="state.search"
        :placeholder="$t('sql-editor.search-sheets')"
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
          tabStore.currentTab.sheetId === query.id
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
</template>

<script lang="ts" setup>
import { escape } from "lodash-es";
import { computed, reactive, ref, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import {
  useNamespacedActions,
  useNamespacedState,
} from "vuex-composition-helpers";
import { useDialog } from "naive-ui";

import { useTabStore } from "@/store";
import {
  SheetActions,
  SheetState,
  Sheet,
  SqlEditorActions,
  SqlEditorState,
  UNKNOWN_ID,
} from "@/types";
import { getHighlightHTMLByKeyWords } from "@/utils";
import { useSQLEditorConnection } from "@/composables/useSQLEditorConnection";

interface State {
  search: string;
  currentSheetName: string;
  editingSheetId: number | null;
  currentActionSheet: Sheet | null;
}

const { t } = useI18n();
const { setConnectionContextFromCurrentTab } = useSQLEditorConnection();
const dialog = useDialog();
const tabStore = useTabStore();

const { sharedSheet, isFetchingSheet: isLoading } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "sharedSheet",
    "isFetchingSheet",
  ]);
const { sheetList } = useNamespacedState<SheetState>("sheet", ["sheetList"]);

// actions
const { setShouldSetContent } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setShouldSetContent"]
);

const { deleteSheet, patchSheetById } = useNamespacedActions<SheetActions>(
  "sheet",
  ["deleteSheet", "patchSheetById"]
);

const state = reactive<State>({
  search: "",
  currentSheetName: "",
  editingSheetId: null,
  currentActionSheet: null,
});

const queryNameInputerRef = ref<HTMLInputElement>();

const data = computed(() => {
  const filterSheetList =
    sharedSheet.value.id !== UNKNOWN_ID
      ? [...sheetList.value, sharedSheet.value]
      : sheetList.value;
  const tempData =
    filterSheetList && filterSheetList.length > 0
      ? filterSheetList.filter((sheet) => {
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
        ? getHighlightHTMLByKeyWords(escape(sheet.name), escape(state.search))
        : escape(sheet.name),
      formatedStatement: state.search
        ? getHighlightHTMLByKeyWords(
            escape(sheet.statement),
            escape(state.search)
          )
        : escape(sheet.statement),
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

    if (tabStore.currentTab.sheetId === state.editingSheetId) {
      tabStore.updateCurrentTab({
        name: state.currentSheetName,
      });
    }
    handleCancelEdit();
  }
};

const handleDeleteSheet = () => {
  if (state.currentActionSheet) {
    deleteSheet(state.currentActionSheet.id);

    if (tabStore.currentTab.sheetId === state.currentActionSheet.id) {
      tabStore.updateCurrentTab({
        sheetId: undefined,
      });
    }
  }
};

const handleActionBtnClick = (key: string, sheet: Sheet) => {
  state.currentActionSheet = sheet;

  if (key === "delete") {
    const $dialog = dialog.create({
      title: t("sql-editor.hint-tips.confirm-to-delete-this-sheet"),
      type: "info",
      onPositiveClick() {
        handleDeleteSheet();
        $dialog.destroy();
      },
      async onNegativeClick() {
        state.currentActionSheet = null;
        $dialog.destroy();
      },
      negativeText: t("common.cancel"),
      positiveText: t("common.confirm"),
      showIcon: false,
    });
  }
};

const handleActionBtnOutsideClick = () => {
  state.currentActionSheet = null;
};

const handleSheetClick = async (sheet: Sheet) => {
  if (tabStore.currentTab.sheetId !== sheet.id) {
    // open opened tab first
    for (const tab of tabStore.tabList) {
      if (tab.sheetId === sheet.id) {
        tabStore.setCurrentTabId(tab.id);
        setConnectionContextFromCurrentTab();
        setShouldSetContent(true);
        return;
      }
    }

    // otherwise, add new tab
    tabStore.addTab({
      name: sheet.name,
      statement: sheet.statement,
      selectedStatement: "",
      sheetId: sheet.id,
    });
    setConnectionContextFromCurrentTab();
    setShouldSetContent(true);
  }
};
</script>
