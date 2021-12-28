<template>
  <MonacoEditor
    v-model="sqlCode"
    @change="handleChange"
    @change-selection="handleChangeSelection"
    @run-query="handleRunQuery"
  />
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { debounce } from "lodash-es";
import { useNamespacedActions } from "vuex-composition-helpers"

import { SqlEditorActions } from "../../../types";

const sqlCode = ref('');

const { setSqlEditorState, executeQueries } = useNamespacedActions<SqlEditorActions>("sqlEditor", [
  "setSqlEditorState",
  "executeQueries",
]);

const handleChange = debounce((value: string) => {
  setSqlEditorState({
    queryStatement: value,
  });
}, 300);

const handleChangeSelection = debounce((value: string) => {
  setSqlEditorState({
    selectedStatement: value,
  });
}, 300);

const handleRunQuery = (statement: string) => {
  return executeQueries({
    databaseName: "blog",
    statement,
  });
};
</script>
