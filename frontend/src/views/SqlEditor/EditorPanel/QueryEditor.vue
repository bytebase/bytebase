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
import { computed } from "vue";
import { debounce } from "lodash-es";

import {
  EditorSelectorActions,
  EditorSelectorGetters,
  SqlEditorActions,
} from "../../../types";
import { useExecuteSQL } from "../../../composables/useExecuteSQL";
import {
  useNamespacedActions,
  useNamespacedGetters,
} from "vuex-composition-helpers";

const { currentTab } = useNamespacedGetters<EditorSelectorGetters>(
  "editorSelector",
  ["currentTab"]
);
const { updateActiveTab } = useNamespacedActions<EditorSelectorActions>(
  "editorSelector",
  ["updateActiveTab"]
);
const { createSavedQuery, patchSavedQuery } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
    "createSavedQuery",
    "patchSavedQuery",
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
  const { label, currentQueryId } = currentTab.value;

  if (currentQueryId) {
    patchSavedQuery({
      id: currentQueryId,
      statement,
    });
  } else {
    const newSavedQuery = await createSavedQuery({
      name: label,
      statement,
    });
    updateActiveTab({
      currentQueryId: newSavedQuery.id,
    });
  }
  updateActiveTab({
    isSaved: true,
  });
};

const handleRunQuery = () => {
  execute();
};
</script>
