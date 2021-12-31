<template>
  <div class="sqleditor-editor-actions">
    <div class="actions-left w-1/2">
      <NButton
        type="primary"
        @click="handleExecuteQueries"
        :disabled="isEmptyStatement"
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
      <NButton secondary strong type="primary" :disabled="isEmptyStatement">
        <carbon:save class="h-5 w-5" /> &nbsp; {{ $t("common.save") }} (⌘+S)
      </NButton>
      <NButton :disabled="isEmptyStatement">
        <carbon:share class="h-5 w-5" /> &nbsp; {{ $t("common.share") }} (⌘+S)
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch } from "vue";
import { CascaderOption } from "naive-ui";
import { useStore } from "vuex";
import { cloneDeep } from "lodash-es";

import {
  useNamespacedState,
  useNamespacedActions,
  useNamespacedGetters,
} from "vuex-composition-helpers";
import {
  SqlEditorState,
  SqlEditorGetters,
  SqlEditorActions,
  ConnectionContext,
} from "../../../types";

const { connectionTree, connectionContext } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "connectionTree",
    "connectionContext",
  ]);
const { isEmptyStatement } = useNamespacedGetters<SqlEditorGetters>(
  "sqlEditor",
  ["isEmptyStatement", "connectionInfo"]
);
const { executeQuery, setConnectionContext } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
    "executeQuery",
    "setConnectionContext",
  ]);

const selectedConnection = ref();
const isSeletedDatabase = ref(false);
const store = useStore();

const handleExecuteQueries = async () => {
  try {
    const res = await executeQuery();
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "SUCCESS",
      title: "Query executed successfully!",
    });
  } catch (error) {
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "CRITICAL",
      title: error,
    });
  }
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
</script>

<style scoped>
.sqleditor-editor-actions {
  @apply w-full flex justify-between items-center p-2;
}
</style>
