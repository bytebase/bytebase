<template>
  <div class="flex flex-col items-stretch justify-start text-sm gap-y-1">
    <div
      v-if="showInfoPane"
      class="gutter-bar--tab"
      :class="[activeTab === 'INFO' && 'gutter-bar--tab-active']"
      @click="handleClickTab('INFO')"
    >
      {{ $t("common.info") }}
    </div>
    <div
      class="gutter-bar--tab"
      :class="[activeTab === 'SHEET' && 'gutter-bar--tab-active']"
      @click="handleClickTab('SHEET')"
    >
      {{ $t("sheet.sheet") }}
    </div>
    <div
      class="gutter-bar--tab"
      :class="[activeTab === 'HISTORY' && 'gutter-bar--tab-active']"
      @click="handleClickTab('HISTORY')"
    >
      {{ $t("common.history") }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from "vue";
import { useInstanceV1Store, useTabStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { instanceV1HasAlterSchema, isDisconnectedTab } from "@/utils";
import { TabView, useSecondarySidebarContext } from "./context";

const { show, tab } = useSecondarySidebarContext();

const activeTab = computed(() => {
  if (!show.value) {
    return undefined;
  }
  return tab.value;
});

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

const handleClickTab = (target: TabView) => {
  if (target === activeTab.value) {
    show.value = false;
    return;
  }

  tab.value = target;
  show.value = true;
};

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

<style lang="postcss" scoped>
.gutter-bar--tab {
  @apply writing-vertical-rl px-1 py-4 border-y bg-gray-50 cursor-pointer select-none;
}
.gutter-bar--tab:first-child {
  @apply border-t-0;
}
.gutter-bar--tab.gutter-bar--tab-active {
  @apply text-accent bg-white;
}
</style>
