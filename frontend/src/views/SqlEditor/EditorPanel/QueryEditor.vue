<template>
  <MonacoEditor
    v-model:value="sqlCode"
    @change="handleChange"
    @change-selection="handleChangeSelection"
    @run-query="handleRunQuery"
    @save="(e) => emit('save-sheet')"
  />
</template>

<script lang="ts" setup>
import { debounce } from "lodash-es";
import { computed, defineEmits } from "vue";
import { useStore } from "vuex";
import {
  useNamespacedActions,
  useNamespacedGetters,
  useNamespacedState,
} from "vuex-composition-helpers";

import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { TabActions, TabGetters, SqlEditorState } from "@/types";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
}>();

const store = useStore();
const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);
const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);
const { updateCurrentTab } = useNamespacedActions<TabActions>("tab", [
  "updateCurrentTab",
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
