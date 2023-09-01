<template>
  <div class="aside-panel h-full">
    <NTabs
      v-model:value="databaseTab"
      type="segment"
      size="small"
      class="h-full"
      pane-style="height: calc(100% - 35px); padding: 0;"
    >
      <NTabPane name="projects" :tab="$t('common.projects')">
        <DatabaseTree
          key="sql-editor-database-tree"
          v-model:search-pattern="searchPattern"
          @alter-schema="$emit('alter-schema', $event)"
        />
      </NTabPane>
      <NTabPane
        v-if="hasInstanceView"
        name="instances"
        :tab="$t('common.instances')"
      >
        <DatabaseTree
          key="sql-editor-database-tree"
          v-model:search-pattern="searchPattern"
          @alter-schema="$emit('alter-schema', $event)"
        />
      </NTabPane>
    </NTabs>

    <NTabPane v-if="false" name="sheets" :tab="$t('sheet.sheets')">
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
  </div>
</template>

<script lang="ts" setup>
import { NTabs, NTabPane } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useConnectionTreeStore, useCurrentUserV1 } from "@/store";
import { ConnectionTreeMode } from "@/types";
import { hasWorkspacePermissionV1 } from "@/utils";
import { useSheetContext } from "../Sheet";
import DatabaseTree from "./DatabaseTree.vue";
import SheetList from "./SheetList";

defineEmits<{
  (
    event: "alter-schema",
    params: { databaseId: string; schema: string; table: string }
  ): void;
}>();

const currentUserV1 = useCurrentUserV1();
const connectionTreeStore = useConnectionTreeStore();
const searchPattern = ref("");
const { events: sheetEvents } = useSheetContext();

const databaseTab = ref<"projects" | "instances">(
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
