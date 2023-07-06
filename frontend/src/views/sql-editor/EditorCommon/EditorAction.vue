<template>
  <div
    class="w-full flex flex-wrap gap-y-2 justify-between sm:items-center p-4 border-b bg-white"
  >
    <div
      class="action-left gap-x-2 flex overflow-x-auto sm:overflow-x-hidden items-center"
    >
      <NButton type="primary" :disabled="!allowQuery" @click="handleRunQuery">
        <mdi:play class="h-5 w-5 -ml-1.5" />
        <span>
          {{
            showRunSelected ? $t("sql-editor.run-selected") : $t("common.run")
          }}
        </span>

        <span class="hidden sm:inline ml-1">
          ({{ keyboardShortcutStr("cmd_or_ctrl+⏎") }})
        </span>
      </NButton>
      <NButton :disabled="!allowQuery" @click="handleExplainQuery">
        <mdi:play class="h-5 w-5 -ml-1.5" />
        <span>Explain</span>
        <span class="hidden sm:inline ml-1">
          ({{ keyboardShortcutStr("cmd_or_ctrl+E") }})
        </span>
      </NButton>
      <NButton :disabled="!allowQuery" @click="handleFormatSQL">
        <span>{{ $t("sql-editor.format") }}</span>
        <span class="hidden sm:inline ml-1">
          ({{ keyboardShortcutStr("shift+opt_or_alt+F") }})
        </span>
      </NButton>
      <NButton
        v-if="showClearScreen"
        :disabled="queryList.length <= 1 || isExecutingSQL"
        @click="handleClearScreen"
      >
        <span>{{ $t("sql-editor.clear-screen") }}</span>
        <span class="hidden sm:inline ml-1">
          ({{ keyboardShortcutStr("shift+opt_or_alt+C") }})
        </span>
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
          <span class="hidden sm:inline ml-1">
            ({{ keyboardShortcutStr("cmd_or_ctrl+S") }})
          </span>
        </NButton>
        <NPopover
          trigger="click"
          placement="bottom-end"
          :show-arrow="false"
          :disabled="!hasSharedSQLScriptFeature"
        >
          <template #trigger>
            <NButton
              :disabled="
                isEmptyStatement ||
                tabStore.isDisconnected ||
                !tabStore.currentTab.isSaved
              "
              @click="handleShareButtonClick"
            >
              <carbon:share class="h-5 w-5" /> &nbsp; {{ $t("common.share") }}
              <FeatureBadge
                :feature="'bb.feature.shared-sql-script'"
                custom-class="ml-2"
              />
            </NButton>
          </template>
          <template #default>
            <SharePopover />
          </template>
        </NPopover>
      </template>
    </div>
  </div>

  <FeatureModal
    :open="state.requiredFeatureName"
    :feature="state.requiredFeatureName"
    @cancel="state.requiredFeatureName = undefined"
  />
</template>

<script lang="ts" setup>
import { computed, defineEmits, reactive } from "vue";
import {
  useTabStore,
  useSQLEditorStore,
  useUIStateStore,
  useWebTerminalStore,
  featureToRef,
  useInstanceV1ByUID,
} from "@/store";
import type { ExecuteConfig, ExecuteOption, FeatureType } from "@/types";
import { TabMode, UNKNOWN_ID } from "@/types";
import SharePopover from "./SharePopover.vue";
import AdminModeButton from "./AdminModeButton.vue";
import { formatEngineV1, keyboardShortcutStr } from "@/utils";

interface LocalState {
  requiredFeatureName?: FeatureType;
}

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

const state = reactive<LocalState>({});
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();
const uiStateStore = useUIStateStore();
const webTerminalStore = useWebTerminalStore();
const hasSharedSQLScriptFeature = featureToRef("bb.feature.shared-sql-script");

const connection = computed(() => tabStore.currentTab.connection);

const isDisconnected = computed(() => {
  return tabStore.isDisconnected;
});
const isEmptyStatement = computed(
  () => !tabStore.currentTab || tabStore.currentTab.statement === ""
);
const isExecutingSQL = computed(() => tabStore.currentTab.isExecutingSQL);
const { instance: selectedInstance } = useInstanceV1ByUID(
  computed(() => connection.value.instanceId)
);
const selectedInstanceEngine = computed(() => {
  return formatEngineV1(selectedInstance.value);
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

const allowQuery = computed(() => {
  if (isDisconnected.value) return false;
  if (isEmptyStatement.value) return false;
  if (isExecutingSQL.value) return false;
  return true;
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
  if (tabStore.currentTab.connection.databaseId === String(UNKNOWN_ID)) {
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
  uiStateStore.saveIntroStateByKey({
    key: "data.query",
    newState: true,
  });
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

const handleShareButtonClick = () => {
  if (!hasSharedSQLScriptFeature.value) {
    state.requiredFeatureName = "bb.feature.shared-sql-script";
  }
};
</script>
