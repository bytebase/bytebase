<template>
  <div
    ref="containerRef"
    class="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1"
  >
    <div class="flex flex-col gap-y-1">
      <TabItem
        tab="WORKSHEET"
        :size="size"
        @click="handleClickTab('WORKSHEET')"
      />
      <TabItem
        tab="SCHEMA"
        :size="size"
        :disabled="!showSchemaPane"
        @click="handleClickTab('SCHEMA')"
      />
      <TabItem tab="HISTORY" :size="size" @click="handleClickTab('HISTORY')" />
    </div>

    <OpenAIButton :size="size" />
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
import { useSQLEditorContext, type AsidePanelTab } from "../../context";
import OpenAIButton from "./OpenAIButton.vue";
import TabItem from "./TabItem.vue";
import type { Size } from "./common";

withDefaults(
  defineProps<{
    size?: Size;
  }>(),
  {
    size: "medium",
  }
);

const me = useCurrentUserV1();
const { currentTab, isDisconnected } = storeToRefs(useSQLEditorTabStore());
const { asidePanelTab } = useSQLEditorContext();
const { instance, database } = useConnectionOfCurrentSQLEditorTab();

const isSchemalessInstance = computed(() => {
  if (instance.value.uid === String(UNKNOWN_ID)) {
    return false;
  }

  return !instanceV1HasAlterSchema(instance.value);
});

const showSchemaPane = computed(() => {
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

const handleClickTab = (target: AsidePanelTab) => {
  if (target === "SCHEMA" && !showSchemaPane.value) {
    return;
  }

  asidePanelTab.value = target;
};

watch(
  showSchemaPane,
  (show) => {
    if (!show && asidePanelTab.value === "SCHEMA") {
      asidePanelTab.value = "WORKSHEET";
    }
  },
  { immediate: true }
);
</script>
