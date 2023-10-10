<template>
  <div class="aside-panel h-full">
    <NTabs
      v-if="hasInstanceView"
      v-model:value="databaseTab"
      type="segment"
      size="small"
      class="primary-sidebar h-full"
      pane-style="height: calc(100% - 35px); padding: 0;"
    >
      <NTabPane name="projects" :tab="projectTitle">
        <DatabaseTree
          key="sql-editor-database-tree"
          v-model:search-pattern="searchPattern"
        />
      </NTabPane>
      <NTabPane name="instances" :tab="$t('common.instance')">
        <DatabaseTree
          key="sql-editor-database-tree"
          v-model:search-pattern="searchPattern"
        />
      </NTabPane>
    </NTabs>
    <div v-else class="primary-sidebar h-full">
      <DatabaseTree
        key="sql-editor-database-tree"
        v-model:search-pattern="searchPattern"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTabs, NTabPane } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useConnectionTreeStore, useCurrentUserV1 } from "@/store";
import { ConnectionTreeMode } from "@/types";
import { hasWorkspacePermissionV1 } from "@/utils";
import { getCustomProjectTitle } from "@/utils/customTheme";
import DatabaseTree from "./DatabaseTree.vue";

const currentUserV1 = useCurrentUserV1();
const connectionTreeStore = useConnectionTreeStore();
const searchPattern = ref("");

const databaseTab = ref<"projects" | "instances">(
  connectionTreeStore.tree.mode === ConnectionTreeMode.INSTANCE
    ? "instances"
    : "projects"
);

const projectTitle = computed(() => {
  return getCustomProjectTitle();
});

const hasInstanceView = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-database",
    currentUserV1.value.userRole
  );
});

watchEffect(() => {
  if (databaseTab.value === "projects") {
    connectionTreeStore.tree.mode = ConnectionTreeMode.PROJECT;
  }
  if (databaseTab.value === "instances") {
    connectionTreeStore.tree.mode = ConnectionTreeMode.INSTANCE;
  }
});
</script>

<style lang="postcss">
.primary-sidebar .n-tabs-rail {
  @apply pt-1;
}
</style>
