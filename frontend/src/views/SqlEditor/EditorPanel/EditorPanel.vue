<template>
  <div class="editor-pane h-full">
    <EditorAction />

    <QueryEditor />

    <BBModal
      v-if="isShowExecutingHint"
      :title="$t('common.tips')"
      @close="handleClose"
    >
      <ExecuteHint @close="handleClose" />
    </BBModal>
  </div>
</template>

<script lang="ts" setup>
import {
  useNamespacedState,
  useNamespacedActions,
} from "vuex-composition-helpers";

import { SqlEditorState, SqlEditorActions } from "../../../types";
import EditorAction from "./EditorAction.vue";
import QueryEditor from "./QueryEditor.vue";
import ExecuteHint from "./ExecuteHint.vue";

const { isShowExecutingHint } = useNamespacedState<SqlEditorState>(
  "sqlEditor",
  ["isShowExecutingHint"]
);

const { setSqlEditorState } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setSqlEditorState"]
);

const handleClose = () => {
  setSqlEditorState({
    isShowExecutingHint: false,
  });
};
</script>
