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

const store = useStore();
const sqlCode = ref("");

const { setSqlEditorState, executeQueries } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
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

const handleRunQuery = async (statement: string) => {
  try {
    const res = await executeQueries({
      statement,
    });
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "SUCCESS",
      title: "Query executed successfully!",
    });
  } catch (error) {
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "CRITICAL",
      title: error,
    });
  }
};
</script>
