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
import { computed, unref } from "vue";
import { useCurrentUserV1, useTabStore, useWebTerminalV1Store } from "@/store";
import { TabMode } from "@/types";
import {
  getSuggestedTabNameFromConnection,
  hasWorkspacePermissionV2,
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
  hasWorkspacePermissionV2(currentUserV1.value, "bb.instances.adminExecute")
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
    name: getSuggestedTabNameFromConnection(current.connection),
  };
  tabStore.selectOrAddSimilarTab(target, /* beside */ true);
  tabStore.updateCurrentTab({
    ...target,
    statement,
  });
  const queryItemList = unref(
    useWebTerminalV1Store().getQueryStateByTab(tabStore.currentTab)
      .queryItemList
  );
  const queryItem = last(queryItemList || []);
  if (queryItem) {
    queryItem.sql = statement;
  }
  emit("enter");
};
</script>
