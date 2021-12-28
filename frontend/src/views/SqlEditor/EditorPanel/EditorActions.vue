<template>
  <div class="sqleditor-editor-actions">
    <div class="actions-left">
      <NButton
        type="primary"
        @click="handleExecuteQueries"
        :disabled="isEmptyStatement"
      >
        <mdi:play class="h-5 w-5" /> Run (⌘+⏎)
      </NButton>
    </div>
    <div class="actions-right space-x-2">
      <NButton secondary strong type="primary" :disabled="isEmptyStatement">
        <carbon:save class="h-5 w-5" /> &nbsp; Save (⌘+S)
      </NButton>
      <NButton :disabled="isEmptyStatement">
        <carbon:share class="h-5 w-5" /> &nbsp; Share (⌘+S)
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useNamespacedActions, useNamespacedGetters } from "vuex-composition-helpers"
import { SqlEditorGetters, SqlEditorActions } from "../../../types"

const { isEmptyStatement } = useNamespacedGetters<SqlEditorGetters>("sqlEditor", ["isEmptyStatement", "connectionInfo"]);
const { executeQueries } = useNamespacedActions<SqlEditorActions>("sqlEditor", ["executeQueries"]);

const handleExecuteQueries = async () => {
  const res = await executeQueries({
    databaseName: "blog",
  });
  console.log(res);
  // store.dispatch("notification/pushNotification", {
  //   module: "sqlEditor",
  //   style: "SUCCESS",
  //   title: "Query executed successfully",
  // });
};
</script>

<style scoped>
.sqleditor-editor-actions {
  @apply w-full flex justify-between items-center p-2;
}
</style>
