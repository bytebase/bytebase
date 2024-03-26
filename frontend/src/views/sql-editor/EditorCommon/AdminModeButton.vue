<template>
  <NButton
    v-if="showButton"
    :size="props.size"
    type="warning"
    :disabled="isDisconnected"
    @click="enterAdminMode"
  >
    <heroicons-outline:wrench class="-ml-1" />
    <span class="ml-1"> {{ $t("sql-editor.admin-mode.self") }} </span>
  </NButton>
</template>

<script lang="ts" setup>
import { last } from "lodash-es";
import { storeToRefs } from "pinia";
import { computed, nextTick, unref } from "vue";
import {
  useCurrentUserV1,
  useSQLEditorTabStore,
  useWebTerminalStore,
} from "@/store";
import type { CoreSQLEditorTab } from "@/types";
import {
  hasWorkspacePermissionV2,
  suggestedTabTitleForSQLEditorConnection,
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

const tabStore = useSQLEditorTabStore();
const { currentTab, isDisconnected } = storeToRefs(tabStore);

const showButton = computed(() => {
  if (!allowAdmin.value) return false;
  const mode = currentTab.value?.mode;
  return mode === "READONLY" || mode === "STANDARD";
});

const enterAdminMode = async () => {
  const tab = currentTab.value;
  if (!tab) {
    return;
  }
  const statement = tab.statement;
  const target: CoreSQLEditorTab = {
    connection: { ...tab.connection },
    mode: "ADMIN",
    sheet: "",
  };
  tabStore.selectOrAddSimilarNewTab(
    target,
    /* beside */ true,
    /* title */ suggestedTabTitleForSQLEditorConnection(tab.connection)
  );
  tabStore.updateCurrentTab({
    ...target,
    statement,
  });

  await nextTick();
  const current = currentTab.value;
  if (!current) {
    return;
  }
  const queryItemList = unref(
    useWebTerminalStore().getQueryStateByTab(current).queryItemList
  );
  const queryItem = last(queryItemList || []);
  if (queryItem) {
    queryItem.sql = statement;
  }
  emit("enter");
};
</script>
