<template>
  <div class="sqleditor-editor-actions">
    <div class="actions-left">
      <n-button type="primary" @click="handleExecuteQueries">
        <mdi:play class="h-5 w-5" /> Run (⌘+⏎)
      </n-button>
    </div>
    <div class="actions-right space-x-2">
      <n-button secondary strong type="primary">
        <carbon:save class="h-5 w-5" /> &nbsp; Save (⌘+S)
      </n-button>
      <n-button> <carbon:share class="h-5 w-5" /> &nbsp; Share (⌘+S) </n-button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useStore } from "vuex";
import { useVuex } from "@vueblocks/vue-use-vuex";

const store = useStore();

const { useActions: useSqlActions } = useVuex("sql", store);
const { useState: useSqlEditorState, useActions: useSqlEditorActions } =
  useVuex("sqlEditor", store);

const { query } = useSqlActions(["query"]) as any;
const { setQueryResult } = useSqlEditorActions(["setQueryResult"]) as any;
const { queryStatement } = useSqlEditorState(["queryStatement"]) as any;

const handleExecuteQueries = async () => {
  const res = await query({
    instanceId: 6100,
    databaseName: "blog",
    statement: queryStatement.value,
  });

  console.log(res);
  setQueryResult(res.data);
};
</script>

<style scoped>
.sqleditor-editor-actions {
  @apply w-full flex justify-between items-center p-2;
}
</style>
