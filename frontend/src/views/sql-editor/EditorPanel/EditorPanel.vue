<template>
  <div class="editor-pane h-full">
    <EditorAction @save-sheet="handleSaveSheet" />

    <template v-if="!sqlEditorStore.isDisconnected">
      <SQLEditor @save-sheet="handleSaveSheet" />
    </template>
    <template v-else>
      <ConnectionHolder />
    </template>

    <BBModal
      v-if="sqlEditorStore.isShowExecutingHint"
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

import { useTabStore, useSQLEditorStore, useSheetStore } from "@/store";
import { defaultTabName } from "@/utils/tab";
import EditorAction from "./EditorAction.vue";
import SQLEditor from "./SQLEditor.vue";
import ExecuteHint from "./ExecuteHint.vue";
import ConnectionHolder from "./ConnectionHolder.vue";
import SaveSheetModal from "./SaveSheetModal.vue";

const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();
const sheetStore = useSheetStore();

const isShowSaveSheetModal = ref(false);

const handleClose = () => {
  sqlEditorStore.setSQLEditorState({
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
  sheetName = sheetName ? sheetName : name;

  const ctx = sqlEditorStore.connectionContext;
  const sheetUpsert = {
    id: sheetId,
    projectId: ctx.projectId,
    databaseId: ctx.databaseId,
    name: sheetName,
    statement: statement,
  };

  const sheet = await sheetStore.upsertSheet(sheetUpsert);

  tabStore.updateCurrentTab({
    sheetId: sheet.id,
    isSaved: true,
    name: sheetName,
  });
};

const handleCloseModal = () => {
  isShowSaveSheetModal.value = false;
};
</script>
