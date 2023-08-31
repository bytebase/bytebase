<template>
  <div class="aside-panel h-full">
    <NTabs
      v-model:value="tabStore.asidePanelTab"
      class="h-full overflow-hidden"
      pane-style="height: calc(100% - 33px); padding: 0 4px;"
      :tabs-padding="8"
    >
      <NTabPane name="databases" :tab="$t('common.databases')">
        <NTabs
          v-model:value="databaseTab"
          type="segment"
          size="small"
          class="h-full"
          pane-style="height: calc(100% - 35px); padding: 0;"
        >
          <NTabPane name="projects" :tab="$t('common.projects')">
            <Splitpanes
              horizontal
              class="default-theme"
              :dbl-click-splitter="false"
            >
              <Pane>
                <DatabaseTree
                  key="sql-editor-database-tree"
                  v-model:search-pattern="searchPattern"
                  @alter-schema="$emit('alter-schema', $event)"
                />
              </Pane>
              <Pane v-if="showSchemaPanel" :size="40">
                <SchemaPanel @alter-schema="$emit('alter-schema', $event)" />
              </Pane>
            </Splitpanes>
          </NTabPane>
          <NTabPane
            v-if="hasInstanceView"
            name="instances"
            :tab="$t('common.instances')"
          >
            <Splitpanes
              horizontal
              class="default-theme"
              :dbl-click-splitter="false"
            >
              <Pane>
                <DatabaseTree
                  key="sql-editor-database-tree"
                  v-model:search-pattern="searchPattern"
                  @alter-schema="$emit('alter-schema', $event)"
                />
              </Pane>
              <Pane v-if="showSchemaPanel" :size="40">
                <SchemaPanel @alter-schema="$emit('alter-schema', $event)" />
              </Pane>
            </Splitpanes>
          </NTabPane>
          <NTabPane name="history" :tab="$t('common.history')">
            <QueryHistoryContainer />
          </NTabPane>
        </NTabs>
      </NTabPane>
      <NTabPane name="sheets" :tab="$t('sheet.sheets')">
        <NTabs
          v-model:value="sheetTab"
          size="small"
          type="segment"
          class="h-full"
          pane-style="height: calc(100% - 35px); padding: 0;"
        >
          <NTabPane name="my" :tab="$t('sheet.mine')">
            <SheetList view="my" />
          </NTabPane>
          <NTabPane name="starred" :tab="$t('sheet.starred')">
            <SheetList view="starred" />
          </NTabPane>
          <NTabPane name="shared" :tab="$t('sheet.shared-with-me')">
            <SheetList view="shared" />
          </NTabPane>
        </NTabs>
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { NTabs, NTabPane } from "naive-ui";
import { Splitpanes, Pane } from "splitpanes";
import { computed, ref, watchEffect } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useConnectionTreeStore,
  useCurrentUserV1,
  useInstanceV1Store,
  useTabStore,
} from "@/store";
import { ConnectionTreeMode, UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV1 } from "@/utils";
import { useSheetContext } from "../Sheet";
import DatabaseTree from "./DatabaseTree.vue";
import QueryHistoryContainer from "./QueryHistoryContainer.vue";
import SchemaPanel from "./SchemaPanel/";
import SheetList from "./SheetList";

defineEmits<{
  (
    event: "alter-schema",
    params: { databaseId: string; schema: string; table: string }
  ): void;
}>();

const currentUserV1 = useCurrentUserV1();
const tabStore = useTabStore();
const connectionTreeStore = useConnectionTreeStore();
const searchPattern = ref("");
const { events: sheetEvents } = useSheetContext();

const databaseTab = ref<"projects" | "instances" | "history">(
  connectionTreeStore.tree.mode === ConnectionTreeMode.INSTANCE
    ? "instances"
    : "projects"
);
const sheetTab = ref<"my" | "shared" | "starred">("my");

const hasInstanceView = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-database",
    currentUserV1.value.userRole
  );
});

const showSchemaPanel = computed(() => {
  const conn = tabStore.currentTab.connection;
  if (conn.databaseId === String(UNKNOWN_ID)) {
    return false;
  }
  const instance = useInstanceV1Store().getInstanceByUID(conn.instanceId);
  if (instance.engine === Engine.REDIS) {
    return false;
  }
  return true;
});

watchEffect(() => {
  if (databaseTab.value === "projects") {
    connectionTreeStore.tree.mode = ConnectionTreeMode.PROJECT;
  }
  if (databaseTab.value === "instances") {
    connectionTreeStore.tree.mode = ConnectionTreeMode.INSTANCE;
  }
});

useEmitteryEventListener(sheetEvents, "add-sheet", () => {
  sheetTab.value = "my";
});
</script>
