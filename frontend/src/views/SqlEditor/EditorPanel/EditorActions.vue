<template>
  <div class="sqleditor-editor-actions">
    <div class="actions-left w-3/5">
      <NButton
        type="primary"
        @click="handleExecuteQueries"
        :disabled="isEmptyStatement"
      >
        <mdi:play class="h-5 w-5" /> {{ $t("common.run") }} (⌘+⏎)
      </NButton>
    </div>
    <div class="actions-right space-x-2 flex w-2/5 justify-end">
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

import {
  useNamespacedState,
  useNamespacedActions,
  useNamespacedGetters,
} from "vuex-composition-helpers";
import {
  SqlEditorState,
  SqlEditorGetters,
  SqlEditorActions,
  ConnectionMeta,
} from "../../../types";

const { connectionTree, connectionMeta } = useNamespacedState<SqlEditorState>(
  "sqlEditor",
  ["connectionTree", "connectionMeta"]
);
const { isEmptyStatement } = useNamespacedGetters<SqlEditorGetters>(
  "sqlEditor",
  ["isEmptyStatement", "connectionInfo"]
);
const { executeQueries, setSqlEditorState } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
    "executeQueries",
    "setSqlEditorState",
  ]);

const selectedConnection = ref();
const isSeletedDatabase = ref(false);
const store = useStore();

const handleExecuteQueries = async () => {
  try {
    const res = await executeQueries();
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
      selectedConnection.value =
        connectionMeta.value.tableId ||
        connectionMeta.value.databaseId ||
        connectionMeta.value.instanceId;
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
    let connectionMeta: ConnectionMeta = {
      instanceId: 0,
      instanceName: "",
    };
    const [instanceInfo, databaseInfo, tableInfo] = pathValues;

    if (pathValues.length >= 1) {
      connectionMeta.instanceId = instanceInfo.id as number;
      connectionMeta.instanceName = instanceInfo.label as string;
    }
    if (pathValues.length >= 2) {
      connectionMeta.databaseId = databaseInfo.id as number;
      connectionMeta.databaseName = databaseInfo.label as string;
    }
    if (pathValues.length >= 3) {
      connectionMeta.tableId = tableInfo.id as number;
      connectionMeta.tableName = tableInfo.label as string;
    }

    setSqlEditorState({
      connectionMeta,
    });
  }
};
</script>

<style scoped>
.sqleditor-editor-actions {
  @apply w-full flex justify-between items-center p-2;
}
</style>
