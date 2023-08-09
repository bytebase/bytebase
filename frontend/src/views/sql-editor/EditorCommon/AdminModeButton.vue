<template>
  <NButton
    v-if="showButton"
    :size="props.size"
    type="warning"
    :disabled="tabStore.isDisconnected"
    @click="enterAdminMode"
  >
    <heroicons-outline:wrench class="-ml-1" />
    <span class="ml-1"> {{ $t("sql-editor.admin-mode.self") }} </span>
  </NButton>
</template>

<script lang="ts" setup>
import { last } from "lodash-es";
import { computed } from "vue";
import { useCurrentUserV1, useTabStore, useWebTerminalStore } from "@/store";
import { TabMode } from "@/types";
import {
  getDefaultTabNameFromConnection,
  hasWorkspacePermissionV1,
} from "@/utils";

const emit = defineEmits<{
  (e: "enter"): void;
}>();

const props = defineProps({
  size: {
    type: String,
    default: "medium",
  },
});

const currentUserV1 = useCurrentUserV1();

const allowAdmin = computed(() =>
  hasWorkspacePermissionV1(
    "bb.permission.workspace.admin-sql-editor",
    currentUserV1.value.userRole
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
