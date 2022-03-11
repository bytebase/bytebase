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

import {
  SqlEditorState,
  SqlEditorActions,
  SqlEditorGetters,
  TabGetters,
  TabActions,
  SheetActions,
} from "../../../types";
import EditorAction from "./EditorAction.vue";
import QueryEditor from "./QueryEditor.vue";
import ExecuteHint from "./ExecuteHint.vue";
import ConnectionHolder from "./ConnectionHolder.vue";
import SaveSheetModal from "./SaveSheetModal.vue";
import { getDefaultTab } from "../../../store/modules/tab";

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
const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);

// actions
const { upsertSheet } = useNamespacedActions<SheetActions>("sheet", [
  "upsertSheet",
]);
const { updateCurrentTab } = useNamespacedActions<TabActions>("tab", [
  "updateCurrentTab",
]);

const isShowSaveSheetModal = ref(false);

const handleClose = () => {
  setSqlEditorState({
    isShowExecutingHint: false,
  });
};

const handleSaveSheet = async (sheetName?: string) => {
  if (currentTab.value.name === getDefaultTab().name && !sheetName) {
    isShowSaveSheetModal.value = true;
    return;
  }
  isShowSaveSheetModal.value = false;

  const { name, statement, sheetId } = currentTab.value;

  const sheet = await upsertSheet({
    id: sheetId,
    name: sheetName ? sheetName : name,
    statement,
  });

  updateCurrentTab({
    sheetId: sheet.id,
    isSaved: true,
    name: sheetName ? sheetName : name,
  });
};

const handleCloseModal = () => {
  isShowSaveSheetModal.value = false;
};
</script>
