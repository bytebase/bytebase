<template>
  <MonacoEditor
    v-model:value="sqlCode"
    @change="handleChange"
    @change-selection="handleChangeSelection"
    @run-query="handleRunQuery"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { debounce } from "lodash-es";
import {
  useNamespacedActions,
  useNamespacedState,
} from "vuex-composition-helpers";

import { SqlEditorActions, EditorSelectorState } from "../../../types";
import { useExecuteSQL } from "../../../composables/useExecuteSQL";

const { activeTab } = useNamespacedState<EditorSelectorState>(
  "editorSelector",
  ["activeTab"]
);
const { setSqlEditorState } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setSqlEditorState"]
);

const { execute } = useExecuteSQL();

const sqlCode = computed(() => activeTab.value.queries);

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

const handleRunQuery = () => {
  execute();
};
</script>
