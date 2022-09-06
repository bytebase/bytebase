<template>
  <div class="w-full flex justify-between items-center p-4 border-b">
    <div class="action-left space-x-2 flex items-center">
      <NButton
        type="primary"
        :disabled="isEmptyStatement || executeState.isLoadingData"
        @click="handleRunQuery"
      >
        <mdi:play class="h-5 w-5" /> {{ $t("common.run") }} (⌘+⏎)
      </NButton>
      <NButton
        :disabled="isEmptyStatement || executeState.isLoadingData"
        @click="handleExplainQuery"
      >
        <mdi:play class="h-5 w-5" /> Explain (⌘+E)
      </NButton>
      <NButton
        :disabled="isEmptyStatement || executeState.isLoadingData"
        @click="handleFormatSQL"
      >
        {{ $t("sql-editor.format") }} (⇧+⌥+F)
      </NButton>
    </div>
    <div class="action-right space-x-2 flex justify-end items-center">
      <div class="flex items-center text-sm space-x-1">
        <NPopover
          v-if="
            connectionContext.instanceId !== UNKNOWN_ID &&
            !hasReadonlyDataSource
          "
          trigger="hover"
        >
          <template #trigger>
            <heroicons-outline:exclamation
              class="h-6 w-6 text-yellow-400 mr-2 mt-0.5"
            />
          </template>
          <p class="py-1">
            {{ $t("instance.no-read-only-data-source-warn") }}
            <NButton
              class="text-base underline text-accent"
              text
              @click="gotoInstanceDetailPage"
            >
              {{ $t("sql-editor.create-read-only-data-source") }}
            </NButton>
          </p>
        </NPopover>

        <div
          v-if="selectedInstance.id !== UNKNOWN_ID"
          class="flex items-center"
        >
          <InstanceEngineIcon :instance="selectedInstance" show-status />
          <span class="ml-2">{{ selectedInstance.name }}</span>
        </div>
        <div
          v-if="selectedDatabase.id !== UNKNOWN_ID"
          class="flex items-center"
        >
          <span class="mx-2">/</span>
          <heroicons-outline:database />
          <span class="ml-2">{{ selectedDatabase.name }}</span>
        </div>
        <div v-if="connectionContext.tableName" class="flex items-center">
          <span class="mx-2">/</span>
          <heroicons-outline:table />
          <span class="ml-2">{{ connectionContext.tableName }}</span>
        </div>
      </div>

      <NButton
        secondary
        strong
        type="primary"
        :disabled="!allowSave"
        @click="() => emit('save-sheet')"
      >
        <carbon:save class="h-5 w-5" /> &nbsp; {{ $t("common.save") }} (⌘+S)
      </NButton>
      <NPopover trigger="click" placement="bottom-end" :show-arrow="false">
        <template #trigger>
          <NButton :disabled="!allowShare">
            <carbon:share class="h-5 w-5" /> &nbsp; {{ $t("common.share") }}
          </NButton>
        </template>
        <template #default>
          <SharePopover />
        </template>
      </NPopover>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, defineEmits } from "vue";
import { useRouter } from "vue-router";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import {
  useInstanceStore,
  useTabStore,
  useSQLEditorStore,
  useDatabaseById,
  useInstanceById,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { instanceSlug } from "@/utils/slug";
import SharePopover from "./SharePopover.vue";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
}>();

const router = useRouter();
const instanceStore = useInstanceStore();
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();

const connectionContext = computed(() => sqlEditorStore.connectionContext);

const isEmptyStatement = computed(
  () => !tabStore.currentTab || tabStore.currentTab.statement === ""
);
const selectedInstance = useInstanceById(
  computed(() => sqlEditorStore.connectionContext.instanceId)
);
const selectedInstanceEngine = computed(() => {
  return instanceStore.formatEngine(selectedInstance.value);
});
const selectedDatabase = useDatabaseById(
  computed(() => sqlEditorStore.connectionContext.databaseId)
);

const hasReadonlyDataSource = computed(() => {
  for (const ds of selectedInstance.value.dataSourceList) {
    if (ds.type === "RO") {
      return true;
    }
  }
  return false;
});

const allowSave = computed(() => {
  return !isEmptyStatement.value && tabStore.currentTab.isModified;
});

const allowShare = computed(() => {
  return (
    typeof tabStore.currentTab.sheetId === "number" &&
    !isEmptyStatement.value &&
    !tabStore.currentTab.isModified &&
    !sqlEditorStore.isDisconnected
  );
});

const { execute, state: executeState } = useExecuteSQL();

const handleRunQuery = async () => {
  const currentTab = tabStore.currentTab;
  const statement = currentTab.statement;
  const selectedStatement = currentTab.selectedStatement;
  const query = selectedStatement || statement;
  await execute(query, { databaseType: selectedInstanceEngine.value });
};

const handleExplainQuery = () => {
  const currentTab = tabStore.currentTab;
  const statement = currentTab.statement;
  const selectedStatement = currentTab.selectedStatement;
  const query = selectedStatement || statement;
  execute(
    query,
    { databaseType: selectedInstanceEngine.value },
    { explain: true }
  );
};

const gotoInstanceDetailPage = () => {
  router.push({
    name: "workspace.instance.detail",
    params: {
      instanceSlug: instanceSlug(selectedInstance.value),
    },
  });
};

const handleFormatSQL = () => {
  sqlEditorStore.setShouldFormatContent(true);
};
</script>
