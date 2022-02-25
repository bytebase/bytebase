<template>
  <div class="sqleditor-editor-actions">
    <div class="actions-left w-1/3">
      <NButton
        type="primary"
        :disabled="isEmptyStatement || executeState.isLoadingData"
        @click="handleRunQuery"
      >
        <mdi:play class="h-5 w-5" /> {{ $t("common.run") }} (⌘+⏎)
      </NButton>
    </div>
    <div class="actions-right space-x-2 flex w-2/3 justify-end">
      <NPopover trigger="hover" placement="bottom-center" :show-arrow="false">
        <template #trigger>
          <label class="flex items-center text-sm space-x-1">
            <div class="flex items-center">
              <InstanceEngineIcon
                v-if="connectionContext.instanceId !== UNKNOWN_ID"
                :instance="selectedInstance"
                show-status
              />
              <span class="ml-2">{{ connectionContext.instanceName }}</span>
            </div>
            <div
              v-if="connectionContext.databaseName"
              class="flex items-center"
            >
              &nbsp; / &nbsp;
              <heroicons-outline:database />
              <span class="ml-2">{{ connectionContext.databaseName }}</span>
            </div>
            <div v-if="connectionContext.tableName" class="flex items-center">
              &nbsp; / &nbsp;
              <heroicons-outline:table />
              <span class="ml-2">{{ connectionContext.tableName }}</span>
            </div>
          </label>
        </template>
        <section>
          <div class="space-y-2">
            <div v-if="connectionContext.instanceName" class="flex flex-col">
              <h1 class="text-gray-400">{{ $t("common.instance") }}:</h1>
              <span>{{ connectionContext.instanceName }}</span>
            </div>
            <div v-if="connectionContext.databaseName" class="flex flex-col">
              <h1 class="text-gray-400">{{ $t("common.database") }}:</h1>
              <span>{{ connectionContext.databaseName }}</span>
            </div>
            <div v-if="connectionContext.tableName" class="flex flex-col">
              <h1 class="text-gray-400">{{ $t("common.table") }}:</h1>
              <span>{{ connectionContext.tableName }}</span>
            </div>
          </div>
        </section>
      </NPopover>
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
            :disabled="isEmptyStatement || isDisconnected"
            @click="isShowSharePopover = !isShowSharePopover"
          >
            <carbon:share class="h-5 w-5" /> &nbsp; {{ $t("common.share") }}
          </NButton>
        </template>
        <SharePopover ref="sharePopover" @close="isShowSharePopover = false" />
      </NPopover>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import {
  useNamespacedState,
  useNamespacedActions,
  useNamespacedGetters,
} from "vuex-composition-helpers";
import { useStore } from "vuex";
import { onClickOutside } from "@vueuse/core";

import {
  SqlEditorState,
  SqlEditorGetters,
  TabGetters,
  SheetActions,
  UNKNOWN_ID,
} from "../../../types";
import { useExecuteSQL } from "../../../composables/useExecuteSQL";
import SharePopover from "./SharePopover.vue";

const store = useStore();

const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);
const { isDisconnected } = useNamespacedGetters<SqlEditorGetters>("sqlEditor", [
  "isDisconnected",
]);

const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);

// actions
const { upsertSheet } = useNamespacedActions<SheetActions>("sheet", [
  "upsertSheet",
]);

const isShowSharePopover = ref(false);
const sharePopover = ref(null);
const isEmptyStatement = computed(
  () => !currentTab.value || currentTab.value.statement === ""
);
const selectedInstance = computed(() => {
  const ctx = connectionContext.value;
  return store.getters["instance/instanceById"](ctx.instanceId);
});

const { execute, state: executeState } = useExecuteSQL();

const handleRunQuery = () => {
  execute();
};

const handleUpsertSheet = async () => {
  const { name, statement, sheetId } = currentTab.value;
  return upsertSheet({
    id: sheetId,
    name,
    statement,
  });
};

onMounted(() => {
  onClickOutside(sharePopover, (event) => (isShowSharePopover.value = false));
});
</script>

<style scoped>
.sqleditor-editor-actions {
  @apply w-full flex justify-between items-center p-2;
}
</style>
