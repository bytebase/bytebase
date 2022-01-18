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

import { EditorSelectorActions, TabInfo } from "../../../types";
import { useExecuteSQL } from "../../../composables/useExecuteSQL";
import { useStore } from "vuex";
import { useNamespacedActions } from "vuex-composition-helpers";

const store = useStore();
const activeTab = computed<TabInfo>(
  () => store.getters["editorSelector/currentTab"]
);
const { updateActiveTab } = useNamespacedActions<EditorSelectorActions>(
  "editorSelector",
  ["updateActiveTab"]
);

const { execute } = useExecuteSQL();

const sqlCode = computed(() => activeTab.value.queryStatement);

const handleChange = debounce((value: string) => {
  updateActiveTab({
    queryStatement: value,
  });
}, 300);

const handleChangeSelection = debounce((value: string) => {
  updateActiveTab({
    selectedStatement: value,
  });
}, 300);

const handleRunQuery = () => {
  execute();
};
</script>
