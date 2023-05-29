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
      </n-tab-pane>
      <n-tab-pane name="history" :tab="$t('common.history')">
        <QueryHistoryContainer />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watchEffect } from "vue";

import {
  useConnectionTreeStore,
  useCurrentUserV1,
  useInstanceV1Store,
  useTabStore,
} from "@/store";
import DatabaseTree from "./DatabaseTree.vue";
import QueryHistoryContainer from "./QueryHistoryContainer.vue";
import SchemaPanel from "./SchemaPanel/";
import { Splitpanes, Pane } from "splitpanes";
import { ConnectionTreeMode, UNKNOWN_ID } from "@/types";
import { hasWorkspacePermissionV1 } from "@/utils";
import { Engine } from "@/types/proto/v1/common";

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

const tab = ref<"projects" | "instances" | "history">(
  connectionTreeStore.tree.mode === ConnectionTreeMode.INSTANCE
    ? "instances"
    : "projects"
);

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
