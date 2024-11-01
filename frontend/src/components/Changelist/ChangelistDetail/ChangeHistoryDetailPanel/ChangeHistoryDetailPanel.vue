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
import { extractChangeHistoryUID, extractDatabaseResourceName } from "@/utils";
import ChangeHistoryDetail from "@/views/DatabaseDetail/ChangeHistoryDetail.vue";
import { provideChangelistDetailContext } from "../context";

const props = defineProps<{
  changeHistoryName?: string;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const { project } = provideChangelistDetailContext();

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
  const { instance, database } = extractDatabaseResourceName(
    changeHistory.value.name
  );
  return {
    project: project.value.name,
    instance,
    database,
    changeHistoryId: extractChangeHistoryUID(changeHistory.value.name),
  };
});

const show = computed(() => {
  return changeHistory.value !== undefined;
});
</script>
