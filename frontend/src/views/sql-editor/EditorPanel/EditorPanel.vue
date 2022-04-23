<template>
  <div class="editor-pane h-full">
    <EditorAction @save-sheet="handleSaveSheet" />

    <template v-if="!sqlEditorStore.isDisconnected">
      <QueryEditor @save-sheet="handleSaveSheet" />
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
import EditorAction from "./EditorAction.vue";
import QueryEditor from "./QueryEditor.vue";
import ExecuteHint from "./ExecuteHint.vue";
import ConnectionHolder from "./ConnectionHolder.vue";
import SaveSheetModal from "./SaveSheetModal.vue";
import { defaultTabName } from "../../../utils/tab";

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

  const sheet = await sheetStore.upsertSheet({
    sheet: {
      id: sheetId as number,
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
