<template>
  <div class="editor-pane h-full">
    <EditorAction />

    <template v-if="!isDisconnected">
      <QueryEditor />
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
  </div>
</template>

<script lang="ts" setup>
import {
  useNamespacedState,
  useNamespacedGetters,
  useNamespacedActions,
} from "vuex-composition-helpers";

import {
  SqlEditorState,
  SqlEditorActions,
  SqlEditorGetters,
} from "../../../types";
import EditorAction from "./EditorAction.vue";
import QueryEditor from "./QueryEditor.vue";
import ExecuteHint from "./ExecuteHint.vue";
import ConnectionHolder from "./ConnectionHolder.vue";

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

const handleClose = () => {
  setSqlEditorState({
    isShowExecutingHint: false,
  });
};
</script>
