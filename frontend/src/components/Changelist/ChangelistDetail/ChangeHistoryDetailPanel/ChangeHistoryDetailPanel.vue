<template>
  <Drawer :show="show" @close="$emit('close')">
    <DrawerContent
      style="width: 75vw; max-width: calc(100vw - 8rem)"
      :title="$t('common.change-history')"
    >
      <ChangeHistoryDetail v-if="detailBindings" v-bind="detailBindings" />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useChangeHistoryStore } from "@/store";
import { extractDatabaseResourceName } from "@/utils";
import ChangeHistoryDetail from "@/views/ChangeHistoryDetail.vue";

const props = defineProps<{
  changeHistoryName?: string;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const changeHistory = computed(() => {
  const { changeHistoryName } = props;
  if (!changeHistoryName) {
    return undefined;
  }
  return useChangeHistoryStore().getChangeHistoryByName(changeHistoryName);
});

const detailBindings = computed(() => {
  if (!changeHistory.value) {
    return undefined;
  }
  const { instance: instanceId, database: databaseName } =
    extractDatabaseResourceName(changeHistory.value.name);
  const { uid } = changeHistory.value;
  return {
    instanceId,
    databaseName,
    changeHistoryId: uid,
  };
});

const show = computed(() => {
  return changeHistory.value !== undefined;
});
</script>
