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
import { useNamespacedActions } from "vuex-composition-helpers";
import { useStore } from "vuex";

import { SqlEditorActions } from "../../../types";
import { useExecuteSQL } from "../../../composables/useExecuteSQL";

const store = useStore();
const sqlCode = ref("");

const { setSqlEditorState } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setSqlEditorState"]
);

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

const handleRunQuery = () => useExecuteSQL(store);
</script>
