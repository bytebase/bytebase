<template>
  <div class="h-full flex flex-col items-stretch justify-between text-sm pb-1">
    <div class="divide-y border-b">
      <div
        v-if="showInfoPane"
        class="gutter-bar--tab writing-vertical-rl"
        :class="[activeTab === 'INFO' && 'gutter-bar--tab-active']"
        @click="handleClickTab('INFO')"
      >
        {{ $t("common.info") }}
      </div>
      <div
        class="gutter-bar--tab writing-vertical-rl"
        :class="[activeTab === 'SHEET' && 'gutter-bar--tab-active']"
        @click="handleClickTab('SHEET')"
      >
        {{ $t("sheet.sheet") }}
      </div>
      <div
        class="gutter-bar--tab writing-vertical-rl"
        :class="[activeTab === 'HISTORY' && 'gutter-bar--tab-active']"
        @click="handleClickTab('HISTORY')"
      >
        {{ $t("common.history") }}
      </div>
    </div>

    <OpenAIButton class="self-center" />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { computed, watch } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useCurrentUserV1,
  useSQLEditorTabStore,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { hasProjectPermissionV2, instanceV1HasAlterSchema } from "@/utils";
import type { TabView } from "../context";
import { useSecondarySidebarContext } from "../context";
import OpenAIButton from "./OpenAIButton.vue";

const { show, tab } = useSecondarySidebarContext();

const me = useCurrentUserV1();
const activeTab = computed(() => {
  if (!show.value) {
    return undefined;
  }
  return tab.value;
});

const tabStore = useSQLEditorTabStore();
const { currentTab, isDisconnected } = storeToRefs(tabStore);
const { instance, database } = useConnectionOfCurrentSQLEditorTab();

const isSchemalessInstance = computed(() => {
  if (!currentTab.value) {
    return false;
  }
  if (isDisconnected.value) {
    return false;
  }
  if (instance.value.uid === String(UNKNOWN_ID)) {
    return false;
  }

  return !instanceV1HasAlterSchema(instance.value);
});

const showInfoPane = computed(() => {
  if (!currentTab.value) {
    return false;
  }
  if (isDisconnected.value) {
    return false;
  }

  if (isSchemalessInstance.value) {
    return false;
  }
  if (database.value.uid === String(UNKNOWN_ID)) {
    return false;
  }

  return hasProjectPermissionV2(
    database.value.projectEntity,
    me.value,
    "bb.databases.getSchema"
  );
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
</script>

<style lang="postcss" scoped>
.gutter-bar--tab {
  @apply px-1 py-4 bg-white cursor-pointer select-none;
}
.gutter-bar--tab.gutter-bar--tab-active {
  @apply bg-gray-100;
}
</style>
