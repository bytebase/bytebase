<template>
  <MonacoEditor
    v-model:value="sqlCode"
    :language="selectedLanguage"
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
import { useNamespacedState } from "vuex-composition-helpers";

import { useTabStore } from "@/store";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { SqlEditorState } from "@/types";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
}>();

const store = useStore();
const tabStore = useTabStore();

const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);

const { execute } = useExecuteSQL();

const sqlCode = computed(() => tabStore.currentTab.statement);
const selectedInstance = computed(() => {
  const ctx = connectionContext.value;
  return store.getters["instance/instanceById"](ctx.instanceId);
});
const selectedInstanceEngine = computed(() => {
  return store.getters["instance/instanceFormatedEngine"](
    selectedInstance.value
  ) as string;
});

const selectedLanguage = computed(() => {
  const engine = selectedInstanceEngine.value;
  if (engine === "MySQL") {
    return "mysql";
  }
  if (engine === "PostgreSQL") {
    return "pgsql";
  }
  return "sql";
});

const handleChange = debounce((value: string) => {
  tabStore.updateCurrentTab({
    statement: value,
    isSaved: false,
  });
}, 300);

const handleChangeSelection = debounce((value: string) => {
  tabStore.updateCurrentTab({
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
