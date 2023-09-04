<template>
  <NTabs
    v-model:value="tab"
    type="segment"
    size="small"
    class="h-full"
    pane-style="height: calc(100% - 35px); padding: 0;"
  >
    <NTabPane v-if="showInfoPane" name="INFO" :tab="$t('common.info')">
      <InfoTabPane @alter-schema="$emit('alter-schema', $event)" />
    </NTabPane>
    <NTabPane name="SHEET" :tab="$t('sheet.sheet')">
      <SheetTabPane />
    </NTabPane>
    <NTabPane name="HISTORY" :tab="$t('common.history')">
      <HistoryTabPane />
    </NTabPane>
  </NTabs>
</template>

<script setup lang="ts">
import { NTabs, NTabPane } from "naive-ui";
import { computed, watch } from "vue";
import { useInstanceV1Store, useTabStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { instanceV1HasAlterSchema, isDisconnectedTab } from "@/utils";
import HistoryTabPane from "./HistoryTabPane";
import InfoTabPane from "./InfoTabPane";
import SheetTabPane from "./SheetTabPane";
import { useSecondarySidebarContext } from "./context";

defineEmits<{
  (
    event: "alter-schema",
    params: { databaseId: string; schema: string; table: string }
  ): void;
}>();

const { tab } = useSecondarySidebarContext();

const tabStore = useTabStore();

const isDisconnected = computed(() => {
  return isDisconnectedTab(tabStore.currentTab);
});

const isSchemalessInstance = computed(() => {
  if (isDisconnected.value) {
    return false;
  }
  const { instanceId } = tabStore.currentTab.connection;

  if (instanceId === String(UNKNOWN_ID)) {
    return false;
  }

  const instance = useInstanceV1Store().getInstanceByUID(instanceId);

  return !instanceV1HasAlterSchema(instance);
});

const showInfoPane = computed(() => {
  if (isDisconnected.value) {
    return false;
  }

  const conn = tabStore.currentTab.connection;
  if (conn.databaseId === String(UNKNOWN_ID)) {
    return false;
  }

  return !isSchemalessInstance.value;
});

watch(
  showInfoPane,
  (show) => {
    if (!show && tab.value === "INFO") {
      tab.value = "SHEET";
    }
  },
  { immediate: true }
);
</script>
