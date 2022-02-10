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
import { computed } from "vue";
import {
  useNamespacedState,
  useNamespacedActions,
} from "vuex-composition-helpers";

import { SqlEditorState, SqlEditorActions } from "../../../types";
import EditorAction from "./EditorAction.vue";
import QueryEditor from "./QueryEditor.vue";
import ExecuteHint from "./ExecuteHint.vue";
import ConnectionHolder from "./ConnectionHolder.vue";

const { isShowExecutingHint, connectionContext } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "isShowExecutingHint",
    "connectionContext",
  ]);

const { setSqlEditorState } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setSqlEditorState"]
);

const isDisconnected = computed(() => {
  const ctx = connectionContext.value;
  return ctx.instanceId === 0 || ctx.databaseId === 0;
});

const handleClose = () => {
  setSqlEditorState({
    isShowExecutingHint: false,
  });
};
</script>
