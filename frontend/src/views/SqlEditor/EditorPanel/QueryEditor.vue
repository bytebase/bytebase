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
import {
  EditorSelectorActions,
  EditorSelectorGetters,
  SqlEditorActions,
} from "../../../types";

const { currentTab } = useNamespacedGetters<EditorSelectorGetters>(
  "editorSelector",
  ["currentTab"]
);
const { updateActiveTab } = useNamespacedActions<EditorSelectorActions>(
  "editorSelector",
  ["updateActiveTab"]
);
const { createSavedQuery, patchSavedQuery, checkSavedQueryExistById } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
    "createSavedQuery",
    "patchSavedQuery",
    "checkSavedQueryExistById",
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
  const isQueryExist = await checkSavedQueryExistById(currentQueryId || -1);

  if (isQueryExist && currentQueryId) {
    patchSavedQuery({
      id: currentQueryId,
      name: label,
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
