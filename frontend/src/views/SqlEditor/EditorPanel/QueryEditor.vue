<template>
  <MonacoEditor
    v-model:value="sqlCode"
    @change="handleChange"
    @change-selection="handleChangeSelection"
    @run-query="handleRunQuery"
    @save="handleSave"
  />
</template>

<script lang="ts" setup>
import { debounce } from "lodash-es";
import { computed } from "vue";
import {
  useNamespacedActions,
  useNamespacedGetters,
} from "vuex-composition-helpers";

import { useExecuteSQL } from "../../../composables/useExecuteSQL";
import { TabActions, TabGetters, SheetActions } from "../../../types";

const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);
const { updateActiveTab } = useNamespacedActions<TabActions>("tab", [
  "updateActiveTab",
]);
const { upsertSheet } = useNamespacedActions<SheetActions>("sheet", [
  "upsertSheet",
]);

const { execute } = useExecuteSQL();

const sqlCode = computed(() => currentTab.value.queryStatement);

const handleChange = debounce((value: string) => {
  updateActiveTab({
    queryStatement: value,
    isSaved: false,
  });
}, 300);

const handleChangeSelection = debounce((value: string) => {
  updateActiveTab({
    selectedStatement: value,
  });
}, 300);

const handleSave = async (statement: string) => {
  const { label, sheetId } = currentTab.value;

  const newSheet = await upsertSheet({
    id: sheetId,
    name: label,
    statement,
  });

  updateActiveTab({
    sheetId: newSheet.id,
    isSaved: true,
  });
};

const handleRunQuery = () => {
  execute();
};
</script>
