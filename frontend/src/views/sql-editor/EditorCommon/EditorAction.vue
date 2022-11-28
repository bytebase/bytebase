<template>
  <div class="w-full flex justify-between items-center p-4 border-b">
    <div class="action-left space-x-2 flex items-center">
      <NButton
        type="primary"
        :disabled="isEmptyStatement || isExecutingSQL"
        @click="handleRunQuery"
      >
        <mdi:play class="h-5 w-5" /> {{ $t("common.run") }} (⌘+⏎)
      </NButton>
      <NButton
        :disabled="isEmptyStatement || isExecutingSQL"
        @click="handleExplainQuery"
      >
        <mdi:play class="h-5 w-5" /> Explain (⌘+E)
      </NButton>
      <NButton
        :disabled="isEmptyStatement || isExecutingSQL"
        @click="handleFormatSQL"
      >
        {{ $t("sql-editor.format") }} (⇧+⌥+F)
      </NButton>
    </div>
    <div class="action-right space-x-2 flex justify-end items-center">
      <TabModeSelect />

      <template v-if="showSheetsFeature">
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
            <NButton
              :disabled="
                isEmptyStatement ||
                tabStore.isDisconnected ||
                !tabStore.currentTab.isSaved
              "
            >
              <carbon:share class="h-5 w-5" /> &nbsp; {{ $t("common.share") }}
            </NButton>
          </template>
          <template #default>
            <SharePopover />
          </template>
        </NPopover>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, defineEmits } from "vue";
import {
  useInstanceStore,
  useTabStore,
  useSQLEditorStore,
  useInstanceById,
} from "@/store";
import { ExecuteConfig, ExecuteOption, TabMode } from "@/types";
import { UNKNOWN_ID } from "@/types";
import SharePopover from "./SharePopover.vue";
import TabModeSelect from "./TabModeSelect.vue";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
  (
    e: "execute",
    sql: string,
    config: ExecuteConfig,
    option?: ExecuteOption
  ): void;
}>();

const instanceStore = useInstanceStore();
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();

const connection = computed(() => tabStore.currentTab.connection);

const isEmptyStatement = computed(
  () => !tabStore.currentTab || tabStore.currentTab.statement === ""
);
const isExecutingSQL = computed(() => tabStore.currentTab.isExecutingSQL);
const selectedInstance = useInstanceById(
  computed(() => connection.value.instanceId)
);
const selectedInstanceEngine = computed(() => {
  return instanceStore.formatEngine(selectedInstance.value);
});

const showSheetsFeature = computed(() => {
  return tabStore.currentTab.mode === TabMode.ReadOnly;
});

const allowSave = computed(() => {
  if (!showSheetsFeature.value) {
    return false;
  }

  if (isEmptyStatement.value) {
    return false;
  }
  if (tabStore.currentTab.isSaved) {
    return false;
  }
  // Temporarily disable saving and sharing if we are connected to an instance
  // but not a database.
  if (tabStore.currentTab.connection.databaseId === UNKNOWN_ID) {
    return false;
  }
  return true;
});

const handleRunQuery = async () => {
  const currentTab = tabStore.currentTab;
  const statement = currentTab.statement;
  const selectedStatement = currentTab.selectedStatement;
  const query = selectedStatement || statement;
  await emit("execute", query, { databaseType: selectedInstanceEngine.value });
};

const handleExplainQuery = () => {
  const currentTab = tabStore.currentTab;
  const statement = currentTab.statement;
  const selectedStatement = currentTab.selectedStatement;
  const query = selectedStatement || statement;
  emit(
    "execute",
    query,
    { databaseType: selectedInstanceEngine.value },
    { explain: true }
  );
};

const handleFormatSQL = () => {
  sqlEditorStore.setShouldFormatContent(true);
};
</script>
