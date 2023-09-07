<template>
  <div class="h-full">
    <InfoTabPane
      v-if="tab === 'INFO'"
      @alter-schema="$emit('alter-schema', $event)"
    />
    <SheetTabPane v-if="tab === 'SHEET'" />
    <HistoryTabPane v-if="tab === 'HISTORY'" />
  </div>
</template>

<script setup lang="ts">
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

watch(
  () => tabStore.currentTab.id,
  () => {
    if (showInfoPane.value) {
      tab.value = "INFO";
    }
  }
);
</script>
