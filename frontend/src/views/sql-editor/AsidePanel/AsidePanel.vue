<template>
  <div class="aside-panel h-full">
    <n-tabs v-model:value="tab" type="segment" class="h-full">
      <n-tab-pane name="projects" :tab="$t('common.projects')">
        <Splitpanes
          horizontal
          class="default-theme"
          :dbl-click-splitter="false"
        >
          <Pane>
            <DatabaseTree />
          </Pane>
          <Pane v-if="showSchemaPanel" :size="40">
            <SchemaPanel />
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
          <Pane>
            <DatabaseTree />
          </Pane>
          <Pane v-if="showSchemaPanel" :size="40">
            <SchemaPanel />
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

const currentUser = useCurrentUser();
const tabStore = useTabStore();
const connectionTreeStore = useConnectionTreeStore();

const tab = ref<"projects" | "instances" | "history">(
  connectionTreeStore.tree.mode === ConnectionTreeMode.INSTANCE
    ? "instances"
    : "projects"
);

const hasInstanceView = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-database",
    currentUser.value.role
  );
});

const showSchemaPanel = computed(() => {
  return tabStore.currentTab.connection.databaseId !== UNKNOWN_ID;
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
