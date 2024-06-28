<template>
  <div
    ref="containerRef"
    class="w-full flex flex-wrap gap-y-2 justify-between sm:items-center p-2 border-b bg-white"
  >
    <div
      class="action-left gap-x-2 flex overflow-x-auto sm:overflow-x-hidden items-center"
    >
      <NButtonGroup>
        <NButton
          type="primary"
          size="small"
          :disabled="!allowQuery"
          @click="handleRunQuery"
        >
          <template #icon>
            <mdi:play class="-ml-1.5" />
          </template>
          <span>
            {{
              showRunSelected ? $t("sql-editor.run-selected") : $t("common.run")
            }}
          </span>

          <span v-show="showShortcutText" class="ml-1">
            ({{ keyboardShortcutStr("cmd_or_ctrl+⏎") }})
          </span>
        </NButton>
        <QueryContextSettingPopover
          v-if="showQueryContextSettingPopover && allowQuery"
        />
      </NButtonGroup>
      <NButton size="small" :disabled="!allowQuery" @click="handleExplainQuery">
        <mdi:play class="-ml-1.5" />
        <span>Explain</span>
        <span v-show="showShortcutText" class="ml-1">
          ({{ keyboardShortcutStr("cmd_or_ctrl+E") }})
        </span>
      </NButton>
      <NButton
        v-if="showClearScreen"
        size="small"
        :disabled="queryList.length <= 1 || isExecutingSQL"
        @click="handleClearScreen"
      >
        <span>{{ $t("sql-editor.clear-screen") }}</span>
        <span v-show="showShortcutText" class="ml-1">
          ({{ keyboardShortcutStr("shift+opt_or_alt+C") }})
        </span>
      </NButton>
      <ResultLimitSelect />
      <QueryModeSelect v-if="showQueryModeSelect" :disabled="isExecutingSQL" />
    </div>
    <div
      class="action-right gap-x-2 flex overflow-x-auto sm:overflow-x-hidden sm:justify-end items-center"
    >
      <AdminModeButton :size="'small'" />

      <template v-if="showSheetsFeature">
        <NButton
          secondary
          strong
          type="primary"
          size="small"
          :disabled="!allowSave"
          @click="handleClickSave"
        >
          <carbon:save class="-ml-1" />
          <span class="ml-1">{{ $t("common.save") }}</span>
          <span v-show="showShortcutText" class="ml-1">
            ({{ keyboardShortcutStr("cmd_or_ctrl+S") }})
          </span>
        </NButton>
        <NPopover
          v-if="!isStandaloneMode"
          trigger="click"
          placement="bottom-end"
          :show-arrow="false"
          :disabled="!hasSharedSQLScriptFeature"
        >
          <template #trigger>
            <NButton
              size="small"
              :disabled="!allowShare"
              @click="handleShareButtonClick"
            >
              <carbon:share class="" /> &nbsp; {{ $t("common.share") }}
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
    :open="!!state.requiredFeatureName"
    :feature="state.requiredFeatureName"
    @cancel="state.requiredFeatureName = undefined"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { NButtonGroup, NButton, NPopover } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive, ref } from "vue";
import {
  useUIStateStore,
  featureToRef,
  usePageMode,
  useActuatorV1Store,
  useSQLEditorTabStore,
  useConnectionOfCurrentSQLEditorTab,
  useWorkSheetStore,
} from "@/store";
import type { FeatureType, SQLEditorQueryParams } from "@/types";
import { keyboardShortcutStr } from "@/utils";
import { useSQLEditorContext } from "../context";
import AdminModeButton from "./AdminModeButton.vue";
import QueryContextSettingPopover from "./QueryContextSettingPopover.vue";
import QueryModeSelect from "./QueryModeSelect.vue";
import ResultLimitSelect from "./ResultLimitSelect.vue";
import SharePopover from "./SharePopover.vue";

interface LocalState {
  requiredFeatureName?: FeatureType;
}

const emit = defineEmits<{
  (e: "execute", params: SQLEditorQueryParams): void;
  (e: "clear-screen"): void;
}>();

const actuatorStore = useActuatorV1Store();
const state = reactive<LocalState>({});
const tabStore = useSQLEditorTabStore();
const uiStateStore = useUIStateStore();
const { standardModeEnabled, events } = useSQLEditorContext();
const containerRef = ref<HTMLDivElement>();
const { width: containerWidth } = useElementSize(containerRef);
const hasSharedSQLScriptFeature = featureToRef("bb.feature.shared-sql-script");
const pageMode = usePageMode();

const isStandaloneMode = computed(() => pageMode.value === "STANDALONE");
const { currentTab, isDisconnected } = storeToRefs(tabStore);

const isEmptyStatement = computed(() => {
  const tab = currentTab.value;
  if (!tab) {
    return true;
  }
  return tab.statement === "";
});
const isExecutingSQL = computed(
  () => currentTab.value?.queryContext?.status === "EXECUTING"
);
const { instance } = useConnectionOfCurrentSQLEditorTab();

const showSheetsFeature = computed(() => {
  const mode = currentTab.value?.mode;
  return mode === "READONLY" || mode === "STANDARD";
});

const showRunSelected = computed(() => {
  const tab = currentTab.value;
  if (!tab) {
    return false;
  }
  return (
    (tab.mode === "READONLY" || tab.mode === "STANDARD") &&
    tab.selectedStatement !== ""
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
  const tab = currentTab.value;
  if (!tab) {
    return false;
  }

  if (tab.worksheet) {
    const sheet = useWorkSheetStore().getWorksheetByName(tab.worksheet);
    if (sheet && sheet.database !== tab.connection.database) {
      return true;
    }
  }
  if (tab.status === "NEW" || tab.status === "CLEAN") {
    return false;
  }

  return true;
});

const allowShare = computed(() => {
  const tab = currentTab.value;
  if (!tab) return false;
  if (tab.status === "NEW" || tab.status === "DIRTY") return false;
  if (isEmptyStatement.value) return false;
  if (isDisconnected.value) return false;
  return true;
});

const showClearScreen = computed(() => {
  return currentTab.value?.mode === "ADMIN";
});

const queryList = computed(() => {
  const tab = currentTab.value;
  if (!tab) {
    return [];
  }
  // TODO: refactor WebTerminal store types
  // return unref(webTerminalStore.getQueryStateByTab(tab).queryItemList) || [];
  return [];
});

const showQueryContextSettingPopover = computed(() => {
  const tab = currentTab.value;
  if (!tab) {
    return false;
  }
  return (
    Boolean(instance.value) &&
    tab.mode !== "ADMIN" &&
    actuatorStore.customTheme === "lixiang"
  );
});

const showQueryModeSelect = computed(() => {
  const tab = currentTab.value;
  if (!tab) {
    return false;
  }
  if (!standardModeEnabled.value) {
    return false;
  }
  return tab.mode !== "ADMIN";
});

const handleRunQuery = () => {
  const tab = currentTab.value;
  if (!tab) {
    return;
  }
  const statement = tab.selectedStatement || tab.statement;
  emit("execute", {
    statement,
    connection: { ...tab.connection },
    engine: instance.value.engine,
    explain: false,
  });
  uiStateStore.saveIntroStateByKey({
    key: "data.query",
    newState: true,
  });
};

const handleExplainQuery = () => {
  const tab = currentTab.value;
  if (!tab) {
    return;
  }
  const statement = tab.selectedStatement || tab.statement;
  emit("execute", {
    statement,
    connection: { ...tab.connection },
    engine: instance.value.engine,
    explain: true,
  });
};

const handleClearScreen = () => {
  emit("clear-screen");
};

const handleClickSave = () => {
  const tab = currentTab.value;
  if (!tab) {
    return;
  }
  events.emit("save-sheet", {
    tab,
  });
};

const handleShareButtonClick = () => {
  if (!hasSharedSQLScriptFeature.value) {
    state.requiredFeatureName = "bb.feature.shared-sql-script";
  }
};

const showShortcutText = computed(() => {
  return containerWidth.value > 800;
});
</script>
