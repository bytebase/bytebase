<template>
  <div class="sqleditor-editor-actions">
    <div class="actions-left w-1/2">
      <NButton
        type="primary"
        :disabled="isEmptyStatement || executeState.isLoadingData"
        @click="handleRunQuery"
      >
        <mdi:play class="h-5 w-5" /> {{ $t("common.run") }} (⌘+⏎)
      </NButton>
    </div>
    <div class="actions-right space-x-2 flex w-1/2 justify-end">
      <NCascader
        v-model:value="selectedConnection"
        expand-trigger="hover"
        label-field="label"
        value-field="id"
        filterable
        :options="connectionTree"
        :placeholder="$t('sql-editor.select-connection')"
        :style="{ width: '400px' }"
        @update:value="handleConnectionChange"
      />
      <NButton
        secondary
        strong
        type="primary"
        :disabled="isEmptyStatement || currentTab.isSaved"
        @click="handleSaveQueryBtnClick"
      >
        <carbon:save class="h-5 w-5" /> &nbsp; {{ $t("common.save") }} (⌘+S)
      </NButton>
      <NButton v-if="isDev()" :disabled="isEmptyStatement">
        <carbon:share class="h-5 w-5" /> &nbsp; {{ $t("common.share") }} (⌘+S)
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { CascaderOption } from "naive-ui";
import { cloneDeep } from "lodash-es";

import {
  useNamespacedState,
  useNamespacedActions,
  useNamespacedGetters,
} from "vuex-composition-helpers";
import {
  SqlEditorState,
  SqlEditorActions,
  ConnectionContext,
  EditorSelectorGetters,
  EditorSelectorActions,
} from "../../../types";
import { useExecuteSQL } from "../../../composables/useExecuteSQL";
import { isDev } from "../../../utils";

const { connectionTree, connectionContext } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "connectionTree",
    "connectionContext",
  ]);
const { currentTab } = useNamespacedGetters<EditorSelectorGetters>(
  "editorSelector",
  ["currentTab"]
);
const { createSavedQuery, setConnectionContext, patchSavedQuery } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
    "createSavedQuery",
    "setConnectionContext",
    "patchSavedQuery",
  ]);
const { updateActiveTab } = useNamespacedActions<EditorSelectorActions>(
  "editorSelector",
  ["updateActiveTab"]
);

const isEmptyStatement = computed(
  () => !currentTab.value || currentTab.value.queryStatement === ""
);

const selectedConnection = ref();
const isSeletedDatabase = ref(false);
const { execute, state: executeState } = useExecuteSQL();

const handleRunQuery = () => {
  execute();
};

watch(
  () => connectionTree.value,
  (newVal) => {
    if (newVal) {
      const ctx = connectionContext.value;
      selectedConnection.value = ctx.hasSlug
        ? ctx.tableId || ctx.databaseId || ctx.instanceId
        : null;
    }
  }
);

const handleConnectionChange = (
  value: number,
  option: CascaderOption,
  pathValues: Array<CascaderOption>
) => {
  isSeletedDatabase.value = pathValues.length >= 2;

  if (pathValues.length >= 1) {
    let ctx: ConnectionContext = cloneDeep(connectionContext.value);
    const [instanceInfo, databaseInfo, tableInfo] = pathValues;

    if (pathValues.length >= 1) {
      ctx.instanceId = instanceInfo.id as number;
      ctx.instanceName = instanceInfo.label as string;
    }
    if (pathValues.length >= 2) {
      ctx.databaseId = databaseInfo.id as number;
      ctx.databaseName = databaseInfo.label as string;
    }
    if (pathValues.length >= 3) {
      ctx.tableId = tableInfo.id as number;
      ctx.tableName = tableInfo.label as string;
    }

    setConnectionContext({
      instanceId: ctx.instanceId,
      instanceName: ctx.instanceName,
      databaseId: ctx.databaseId,
      databaseName: ctx.databaseName,
      tableId: ctx.tableId,
      tableName: ctx.tableName,
    });
  }
};

const handleSaveQueryBtnClick = async () => {
  const { queryStatement, label, currentQueryId } = currentTab.value;

  if (currentQueryId) {
    patchSavedQuery({
      id: currentQueryId,
      statement: queryStatement,
    });
  } else {
    const newSavedQuery = await createSavedQuery({
      name: label,
      statement: queryStatement,
    });
    updateActiveTab({
      currentQueryId: newSavedQuery.id,
    });
  }
  updateActiveTab({
    isSaved: true,
  });
};
</script>

<style scoped>
.sqleditor-editor-actions {
  @apply w-full flex justify-between items-center p-2;
}
</style>
