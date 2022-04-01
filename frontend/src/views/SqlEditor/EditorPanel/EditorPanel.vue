<template>
  <div class="editor-pane h-full">
    <EditorAction @save-sheet="handleSaveSheet" />

    <template v-if="!isDisconnected">
      <QueryEditor @save-sheet="handleSaveSheet" />
    </template>
    <template v-else>
      <ConnectionHolder />
    </template>

    <BBModal
      v-if="isShowExecutingHint"
      :title="$t('common.tips')"
      @close="handleClose"
    >
      <ExecuteHint @close="handleClose" />
    </BBModal>
    <BBModal
      v-if="isShowSaveSheetModal"
      :title="$t('sql-editor.save-sheet')"
      @close="handleCloseModal"
    >
      <SaveSheetModal @close="handleCloseModal" @save-sheet="handleSaveSheet" />
    </BBModal>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import {
  useNamespacedState,
  useNamespacedGetters,
  useNamespacedActions,
} from "vuex-composition-helpers";

import { useTabStore } from "@/store/pinia-modules/tab";
import {
  SqlEditorState,
  SqlEditorActions,
  SqlEditorGetters,
  SheetActions,
} from "@/types";
import EditorAction from "./EditorAction.vue";
import QueryEditor from "./QueryEditor.vue";
import ExecuteHint from "./ExecuteHint.vue";
import ConnectionHolder from "./ConnectionHolder.vue";
import SaveSheetModal from "./SaveSheetModal.vue";
import { defaultTabName } from "../../../utils/tab";

const tabStore = useTabStore();

const { isShowExecutingHint } = useNamespacedState<SqlEditorState>(
  "sqlEditor",
  ["isShowExecutingHint"]
);

const { isDisconnected } = useNamespacedGetters<SqlEditorGetters>("sqlEditor", [
  "isDisconnected",
]);

const { setSqlEditorState } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setSqlEditorState"]
);

// actions
const { upsertSheet } = useNamespacedActions<SheetActions>("sheet", [
  "upsertSheet",
]);

const isShowSaveSheetModal = ref(false);

const handleClose = () => {
  setSqlEditorState({
    isShowExecutingHint: false,
  });
};

const handleSaveSheet = async (sheetName?: string) => {
  if (tabStore.currentTab.name === defaultTabName.value && !sheetName) {
    isShowSaveSheetModal.value = true;
    return;
  }
  isShowSaveSheetModal.value = false;

  const { name, statement, sheetId } = tabStore.currentTab;

  const sheet = await upsertSheet({
    sheet: {
      id: sheetId,
      name: sheetName ? sheetName : name,
      statement,
    },
    currentTab: tabStore.currentTab,
  });

  tabStore.updateCurrentTab({
    sheetId: sheet.id,
    isSaved: true,
    name: sheetName ? sheetName : name,
  });
};

const handleCloseModal = () => {
  isShowSaveSheetModal.value = false;
};
</script>
