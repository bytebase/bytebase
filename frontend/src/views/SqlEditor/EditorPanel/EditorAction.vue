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
        @click="handleUpsertSheet"
      >
        <carbon:save class="h-5 w-5" /> &nbsp; {{ $t("common.save") }} (⌘+S)
      </NButton>
      <NPopover
        trigger="click"
        placement="bottom-end"
        :show-arrow="false"
        :show="isShowSharePopover"
      >
        <template #trigger>
          <NButton
            :disabled="isEmptyStatement || !isSelectedConnection"
            @click="isShowSharePopover = !isShowSharePopover"
          >
            <carbon:share class="h-5 w-5" /> &nbsp; {{ $t("common.share") }}
          </NButton>
        </template>
        <SharePopover @close="isShowSharePopover = false" />
      </NPopover>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch, onMounted, nextTick } from "vue";
import { cloneDeep } from "lodash-es";
import { CascaderOption } from "naive-ui";
import { useRouter } from "vue-router";
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
  TabGetters,
  SheetActions,
  TabActions,
} from "../../../types";
import { useExecuteSQL } from "../../../composables/useExecuteSQL";
import SharePopover from "./SharePopover.vue";

const router = useRouter();
const { connectionTree, connectionContext } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "connectionTree",
    "connectionContext",
  ]);
const { isDisconnected } = useNamespacedGetters<SqlEditorGetters>("sqlEditor", [
  "isDisconnected",
]);
const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);

// actions
const { setConnectionContext } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setConnectionContext"]
);

const { upsertSheet } = useNamespacedActions<SheetActions>("sheet", [
  "upsertSheet",
]);

const { updateActiveTab } = useNamespacedActions<TabActions>("tab", [
  "updateActiveTab",
]);

const selectedConnection = ref();
const isSeletedDatabase = ref(false);
const isShowSharePopover = ref(false);
const isEmptyStatement = computed(
  () => !currentTab.value || currentTab.value.queryStatement === ""
);
const isSelectedConnection = computed(
  () =>
    connectionContext.value.instanceId !== 0 &&
    connectionContext.value.databaseId !== 0
);

const { execute, state: executeState } = useExecuteSQL();

const handleRunQuery = () => {
  execute();
};

const setSelectedConnection = (ctx: ConnectionContext) => {
  if (ctx) {
    selectedConnection.value = ctx.hasSlug
      ? ctx.tableId || ctx.databaseId || ctx.instanceId
      : null;
  }
};

watch(
  () => connectionTree.value,
  (newVal) => {
    if (newVal) {
      setSelectedConnection(connectionContext.value);
    }
  }
);

watch(
  () => connectionContext.value,
  (newVal) => {
    if (newVal) {
      setSelectedConnection(newVal);
    }
  },
  { deep: true }
);

onMounted(() => {
  setSelectedConnection(connectionContext.value);
});

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

const handleUpsertSheet = async () => {
  const { label, queryStatement, sheetId } = currentTab.value;
  return upsertSheet({
    id: sheetId,
    name: label,
    statement: queryStatement,
  });
};

// selected a new connection
watch(
  () => isDisconnected.value,
  async (newVal) => {
    if (!newVal) {
      const newSheet = await handleUpsertSheet();
      updateActiveTab({
        sheetId: newSheet.id,
        isSaved: true,
      });
    }
  }
);
</script>

<style scoped>
.sqleditor-editor-actions {
  @apply w-full flex justify-between items-center p-2;
}
</style>
