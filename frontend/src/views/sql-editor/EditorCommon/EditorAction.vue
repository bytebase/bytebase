<template>
  <div
    class="w-full flex flex-wrap gap-y-2 justify-between sm:items-center p-2 border-b bg-white"
    v-bind="$attrs"
  >
    <div
      class="action-left gap-x-2 flex overflow-x-auto sm:overflow-x-hidden items-center"
    >
      <NButton
        v-if="currentTab?.mode === 'ADMIN'"
        size="small"
        type="default"
        :dashed="true"
        style="--n-padding: 0 8px 0 3px"
        @click.stop="exitAdminMode"
      >
        <template #icon>
          <ChevronLeftIcon class="w-4 h-4 text-control" />
        </template>
        <span>{{ $t("sql-editor.admin-mode.exit") }}</span>
      </NButton>

      <NButtonGroup v-if="currentTab?.mode !== 'ADMIN'">
        <NTooltip :disabled="!queryTip">
          <template #trigger>
            <NButton
              :disabled="!allowQuery"
              type="primary"
              size="small"
              style="--n-padding: 0 5px"
              @click="handleRunQuery"
            >
              <template #icon>
                <PlayIcon class="w-4 h-4 fill-current!" />
              </template>
              <template #default>
                <div class="inline-flex items-center">
                  <span>(</span>
                  <span>limit&nbsp;{{ resultRowsLimit }}</span>
                  <span>)</span>
                </div>
              </template>
            </NButton>
          </template>
          <span class="text-sm">
            {{ queryTip }}
          </span>
        </NTooltip>
        <QueryContextSettingPopover
          :disabled="!showQueryContextSettingPopover || !allowQuery"
        />
      </NButtonGroup>

      <NPopover placement="bottom">
        <template #trigger>
          <AdminModeButton
            size="small"
            :hide-text="true"
            style="--n-padding: 0 5px"
          />
        </template>
        <template #default>
          <span>{{ $t("sql-editor.admin-mode.self") }}</span>
        </template>
      </NPopover>

      <template v-if="showSheetsFeature">
        <NPopover placement="bottom">
          <template #trigger>
            <NButton
              :strong="allowSave"
              :disabled="!allowSave"
              size="small"
              style="--n-padding: 0 5px"
              @click="handleClickSave"
            >
              <template #icon>
                <SaveIcon class="w-4 h-4" />
              </template>
            </NButton>
          </template>
          <template #default>
            <div class="flex items-center gap-1">
              <span>{{ $t("common.save") }}</span>
              <span>({{ keyboardShortcutStr("cmd_or_ctrl+S") }})</span>
            </div>
          </template>
        </NPopover>
        <NPopover
          trigger="click"
          placement="bottom-end"
          :show-arrow="false"
          :disabled="false"
        >
          <template #trigger>
            <NPopover placement="bottom" trigger="hover">
              <template #trigger>
                <NButton
                  :strong="allowShare"
                  :disabled="!allowShare"
                  size="small"
                  style="--n-padding: 0 5px"
                >
                  <template #icon>
                    <Share2Icon class="w-4 h-4" />
                  </template>
                </NButton>
              </template>
              <template #default>
                <div class="flex items-center gap-1">
                  <span>{{ $t("common.share") }}</span>
                </div>
              </template>
            </NPopover>
          </template>
          <template #default>
            <SharePopover :worksheet="sheetAndTabStore.currentSheet" />
          </template>
        </NPopover>
      </template>
    </div>
    <div
      class="action-right gap-x-2 flex overflow-x-auto sm:overflow-x-hidden sm:justify-end items-center"
    >
      <NButtonGroup>
        <DatabaseChooser />
        <SchemaChooser />
        <ContainerChooser />
      </NButtonGroup>

      <OpenAIButton
        size="small"
        :statement="currentTab?.selectedStatement || currentTab?.statement"
      />
    </div>
  </div>

  <FeatureModal
    :open="!!state.requiredFeatureName"
    :feature="state.requiredFeatureName"
    @cancel="state.requiredFeatureName = undefined"
  />
</template>

<script lang="ts" setup>
import {
  ChevronLeftIcon,
  PlayIcon,
  SaveIcon,
  Share2Icon,
} from "lucide-vue-next";
import { NButton, NButtonGroup, NPopover, NTooltip } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { FeatureModal } from "@/components/FeatureGuard";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useUIStateStore,
  useWorkSheetAndTabStore,
  useWorkSheetStore,
} from "@/store";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  type SQLEditorQueryParams,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  isDatabaseV1Queryable,
  isWorksheetWritableV1,
  keyboardShortcutStr,
} from "@/utils";
import { useSQLEditorContext } from "../context";
import AdminModeButton from "./AdminModeButton.vue";
import ContainerChooser from "./ContainerChooser.vue";
import DatabaseChooser from "./DatabaseChooser.vue";
import OpenAIButton from "./OpenAIButton";
import QueryContextSettingPopover from "./QueryContextSettingPopover.vue";
import SchemaChooser from "./SchemaChooser.vue";
import SharePopover from "./SharePopover.vue";

interface LocalState {
  requiredFeatureName?: PlanFeature;
}

defineOptions({
  inheritAttrs: false,
});

const emit = defineEmits<{
  (e: "execute", params: SQLEditorQueryParams): void;
}>();

const state = reactive<LocalState>({});
const tabStore = useSQLEditorTabStore();
const uiStateStore = useUIStateStore();
const { events } = useSQLEditorContext();
const { resultRowsLimit } = storeToRefs(useSQLEditorStore());
const sheetAndTabStore = useWorkSheetAndTabStore();

const { currentTab, isDisconnected } = storeToRefs(tabStore);

const isEmptyStatement = computed(() => {
  const tab = currentTab.value;
  if (!tab) {
    return true;
  }
  return tab.statement === "";
});
const { instance, database } = useConnectionOfCurrentSQLEditorTab();
const { t } = useI18n();

const showSheetsFeature = computed(() => {
  const mode = currentTab.value?.mode;
  return mode === "WORKSHEET";
});

const queryTip = computed(() => {
  if (
    instance.value.engine === Engine.COSMOSDB &&
    !currentTab.value?.connection.table
  ) {
    return t("database.table.select-tip");
  }
  return "";
});

const allowQuery = computed(() => {
  if (isDisconnected.value) return false;
  if (isEmptyStatement.value) return false;

  if (instance.value.engine === Engine.COSMOSDB) {
    return !!currentTab.value?.connection.table;
  }

  return isDatabaseV1Queryable(database.value);
});

const canWriteSheet = computed(() => {
  if (!currentTab.value || !currentTab.value.worksheet) {
    return false;
  }
  const sheet = useWorkSheetStore().getWorksheetByName(
    currentTab.value.worksheet
  );
  if (!sheet) {
    return false;
  }
  return isWorksheetWritableV1(sheet);
});

const allowSave = computed(() => {
  if (!showSheetsFeature.value) {
    return false;
  }
  const tab = currentTab.value;
  if (!tab) {
    return false;
  }

  if (tab.worksheet) {
    if (!canWriteSheet.value) {
      return false;
    }
    const sheet = useWorkSheetStore().getWorksheetByName(tab.worksheet);
    if (sheet && sheet.database !== tab.connection.database) {
      return true;
    }
  }
  if (tab.status === "CLEAN") {
    return false;
  }

  return true;
});

const allowShare = computed(() => {
  const tab = currentTab.value;
  if (!tab) return false;
  if (tab.status === "DIRTY") return false;
  if (isEmptyStatement.value) return false;
  if (isDisconnected.value) return false;
  if (tab.worksheet) {
    if (!canWriteSheet.value) {
      return false;
    }
  }
  return true;
});

const showQueryContextSettingPopover = computed(() => {
  const tab = currentTab.value;
  if (!tab) {
    return false;
  }
  return instance.value && tab.mode !== "ADMIN";
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
    selection: tab.editorState.selection,
  });
  uiStateStore.saveIntroStateByKey({
    key: "data.query",
    newState: true,
  });
};

const exitAdminMode = () => {
  tabStore.updateCurrentTab({
    mode: DEFAULT_SQL_EDITOR_TAB_MODE,
  });
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
</script>
