<template>
  <NButton
    v-if="showButton"
    type="warning"
    :disabled="tabStore.isDisconnected"
    @click="enterAdminMode"
  >
    <heroicons-outline:wrench />
    <span class="ml-1"> {{ $t("sql-editor.admin-mode.self") }} </span>
  </NButton>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { TabMode } from "@/types";
import { useCurrentUser, useTabStore, useWebTerminalStore } from "@/store";
import {
  getDefaultTabNameFromConnection,
  hasWorkspacePermission,
} from "@/utils";
import { last } from "lodash-es";

const emit = defineEmits<{
  (e: "enter"): void;
}>();

const currentUser = useCurrentUser();

const allowAdmin = computed(() =>
  hasWorkspacePermission(
    "bb.permission.workspace.admin-sql-editor",
    currentUser.value.role
  )
);

const tabStore = useTabStore();

const showButton = computed(() => {
  if (!allowAdmin.value) return false;
  return tabStore.currentTab.mode === TabMode.ReadOnly;
});

const enterAdminMode = () => {
  const current = tabStore.currentTab;
  const statement = current.statement;
  const target = {
    connection: current.connection,
    mode: TabMode.Admin,
    name: getDefaultTabNameFromConnection(current.connection),
  };
  tabStore.selectOrAddSimilarTab(target, /* beside */ true);
  tabStore.updateCurrentTab({
    ...target,
    statement,
  });
  const queryItem = last(
    useWebTerminalStore().getQueryListByTab(tabStore.currentTab)
  );
  if (queryItem) {
    queryItem.sql = statement;
  }
  emit("enter");
};
</script>
