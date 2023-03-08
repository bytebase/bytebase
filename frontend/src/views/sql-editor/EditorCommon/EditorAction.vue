<template>
  <div
    class="w-full flex flex-wrap gap-y-2 justify-between sm:items-center p-4 border-b bg-white"
  >
    <div
      class="action-left gap-x-2 flex overflow-x-auto sm:overflow-x-hidden items-center"
    >
      <NButton
        type="primary"
        :disabled="isEmptyStatement || isExecutingSQL"
        @click="handleRunQuery"
      >
        <mdi:play class="h-5 w-5 -ml-1.5" />
        <span>
          {{
            showRunSelected ? $t("sql-editor.run-selected") : $t("common.run")
          }}
        </span>

        <span class="hidden sm:inline">(⌘+⏎)</span>
      </NButton>
      <NButton
        :disabled="isEmptyStatement || isExecutingSQL"
        @click="handleExplainQuery"
      >
        <mdi:play class="h-5 w-5 -ml-1.5" />
        <span>Explain</span>
        <span class="hidden sm:inline">(⌘+E)</span>
      </NButton>
      <NButton
        :disabled="isEmptyStatement || isExecutingSQL"
        @click="handleFormatSQL"
      >
        <span>{{ $t("sql-editor.format") }}</span>
        <span class="hidden sm:inline">(⇧+⌥+F)</span>
      </NButton>
      <NButton
        v-if="showClearScreen"
        :disabled="queryList.length <= 1 || isExecutingSQL"
        @click="handleClearScreen"
      >
        <span>{{ $t("sql-editor.clear-screen") }}</span>
        <span class="hidden sm:inline">(⇧+⌥+C)</span>
      </NButton>
    </div>
    <div
      class="action-right gap-x-2 flex overflow-x-auto sm:overflow-x-hidden sm:justify-end items-center"
    >
      <AdminModeButton />

      <template v-if="showSheetsFeature">
        <NButton
          secondary
          strong
          type="primary"
          :disabled="!allowSave"
          @click="() => emit('save-sheet')"
        >
          <carbon:save class="h-5 w-5 -ml-1" />
          <span class="ml-1">{{ $t("common.save") }}</span>
          <span class="hidden sm:inline ml-1">(⌘+S)</span>
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

    <div
      v-if="
        tabStore.currentTab.mode === TabMode.ReadOnly &&
        !tabStore.isDisconnected
      "
      class="w-full -mb-1"
    >
      <AIPromptButton
        :engine-type="instance.engine"
        :database-metadata="databaseMetadata"
        @apply-statement="handleApplySQL"
      />
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
  useWebTerminalStore,
  useMetadataByDatabaseId,
} from "@/store";
import type { ExecuteConfig, ExecuteOption } from "@/types";
import { TabMode, UNKNOWN_ID } from "@/types";
import { AIPromptButton } from "@/plugins/ai";
import SharePopover from "./SharePopover.vue";
import AdminModeButton from "./AdminModeButton.vue";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
  (
    e: "execute",
    sql: string,
    config: ExecuteConfig,
    option?: ExecuteOption
  ): void;
  (e: "clear-screen"): void;
}>();

const instanceStore = useInstanceStore();
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();
const webTerminalStore = useWebTerminalStore();

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

const showRunSelected = computed(() => {
  return (
    tabStore.currentTab.mode === TabMode.ReadOnly &&
    !!tabStore.currentTab.selectedStatement
  );
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

const showClearScreen = computed(() => {
  return tabStore.currentTab.mode === TabMode.Admin;
});

const queryList = computed(() => {
  return webTerminalStore.getQueryListByTab(tabStore.currentTab);
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

const handleClearScreen = () => {
  emit("clear-screen");
};

const databaseId = computed(() => tabStore.currentTab.connection.databaseId);
const databaseMetadata = useMetadataByDatabaseId(databaseId, false);
const instance = useInstanceById(
  computed(() => tabStore.currentTab.connection.instanceId)
);

const handleApplySQL = (sql: string, run: boolean) => {
  tabStore.currentTab.statement = sql;
  if (run) {
    emit("execute", sql, { databaseType: selectedInstanceEngine.value });
  }
};
</script>
