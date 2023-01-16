<template>
  <div class="aside-panel h-full">
    <n-tabs v-model:value="tab" type="segment" class="h-full">
      <n-tab-pane name="projects" :tab="$t('common.projects')">
        <Splitpanes
          horizontal
          class="default-theme"
          :dbl-click-splitter="false"
        >
          <Pane :size="databasePaneSize">
            <DatabaseTree />
          </Pane>
          <Pane :size="FULL_HEIGHT - databasePaneSize">
            <SchemaPanel v-if="showSchemaPanel" />
          </Pane>
        </Splitpanes>
      </n-tab-pane>
      <n-tab-pane
        v-if="hasInstanceView"
        name="instances"
        :tab="$t('common.instances')"
      >
        <Splitpanes
          horizontal
          class="default-theme"
          :dbl-click-splitter="false"
        >
          <Pane :size="databasePaneSize">
            <DatabaseTree />
          </Pane>
          <Pane :size="FULL_HEIGHT - databasePaneSize">
            <SchemaPanel v-if="showSchemaPanel" />
          </Pane>
        </Splitpanes>
      </n-tab-pane>
      <n-tab-pane name="history" :tab="$t('common.history')">
        <QueryHistoryContainer />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watchEffect } from "vue";

import { useConnectionTreeStore, useCurrentUser, useTabStore } from "@/store";
import DatabaseTree from "./DatabaseTree.vue";
import QueryHistoryContainer from "./QueryHistoryContainer.vue";
import SchemaPanel from "./SchemaPanel/";
import { Splitpanes, Pane } from "splitpanes";
import { ConnectionTreeMode, UNKNOWN_ID } from "@/types";
import { hasWorkspacePermission } from "@/utils";

const FULL_HEIGHT = 100;
const DATABASE_PANE_SIZE = 60;

const currentUser = useCurrentUser();
const tabStore = useTabStore();

const tab = ref<"projects" | "instances" | "history">("projects");

const hasInstanceView = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-database",
    currentUser.value.role
  );
});

const connectionTreeStore = useConnectionTreeStore();

const showSchemaPanel = computed(() => {
  return tabStore.currentTab.connection.databaseId !== UNKNOWN_ID;
});

const databasePaneSize = computed(() => {
  return showSchemaPanel.value ? DATABASE_PANE_SIZE : FULL_HEIGHT;
});

watchEffect(() => {
  if (tab.value === "projects") {
    connectionTreeStore.tree.mode = ConnectionTreeMode.PROJECT;
  }
  if (tab.value === "instances") {
    connectionTreeStore.tree.mode = ConnectionTreeMode.INSTANCE;
  }
});
</script>

<style scoped>
.aside-panel .n-tab-pane {
  height: calc(100% - 40px);
  @apply pt-0;
}
</style>
