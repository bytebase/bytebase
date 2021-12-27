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
import { useStore } from "vuex";
import { useVuex } from "@vueblocks/vue-use-vuex";

const sqlCode = ref(`SELECT * FROM author WHERE author.name LIKE "y%"`);
const store = useStore();
const { useActions } = useVuex("sqlEditor", store);

const { setSqlEditorState, executeQueries } = useActions([
  "setSqlEditorState",
  "executeQueries",
]) as any;

const handleChange = debounce((value: string) => {
  console.log("handleChange", value);
  setSqlEditorState({
    queryStatement: value,
  });
}, 300);

const handleChangeSelection = debounce((value: string) => {
  console.log("handleChangeSelection", value);
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
