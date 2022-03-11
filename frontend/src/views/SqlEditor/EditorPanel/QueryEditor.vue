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
import { useStore } from "vuex";
import {
  useNamespacedActions,
  useNamespacedGetters,
  useNamespacedState,
} from "vuex-composition-helpers";

import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { TabActions, TabGetters, SheetActions, SqlEditorState } from "@/types";

const store = useStore();
const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);
const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);
const { updateCurrentTab } = useNamespacedActions<TabActions>("tab", [
  "updateCurrentTab",
]);
const { upsertSheet } = useNamespacedActions<SheetActions>("sheet", [
  "upsertSheet",
]);

const { execute } = useExecuteSQL();

const sqlCode = computed(() => currentTab.value.statement);
const selectedInstance = computed(() => {
  const ctx = connectionContext.value;
  return store.getters["instance/instanceById"](ctx.instanceId);
});
const selectedInstanceEngine = computed(() => {
  return store.getters["instance/instanceFormatedEngine"](
    selectedInstance.value
  ) as string;
});

const handleChange = debounce((value: string) => {
  updateCurrentTab({
    statement: value,
    isSaved: false,
  });
}, 300);

const handleChangeSelection = debounce((value: string) => {
  updateCurrentTab({
    selectedStatement: value,
  });
}, 300);

const handleSave = async (statement: string) => {
  const { name, sheetId } = currentTab.value;

  const sheet = await upsertSheet({
    id: sheetId,
    name,
    statement,
  });

  updateCurrentTab({
    sheetId: sheet.id,
    isSaved: true,
  });
};

const handleRunQuery = ({
  explain,
  query,
}: {
  explain: boolean;
  query: string;
}) => {
  execute({ databaseType: selectedInstanceEngine.value }, { explain });
};
</script>
