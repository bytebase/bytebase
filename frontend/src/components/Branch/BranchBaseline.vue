<template>
  <div class="flex flex-row gap-x-1 text-sm">
    <RichDatabaseName
      :database="database"
      :show-instance="false"
      :show-instance-icon="!!showInstanceIcon"
      :show-arrow="false"
      :show-production-environment-icon="false"
      tooltip="instance"
    />
    <span class="text-control-light">@</span>
    <span>{{ baselineVersion }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { RichDatabaseName } from "@/components/v2";
import { useChangeHistoryStore, useDatabaseV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";

const props = withDefaults(
  defineProps<{
    branch: SchemaDesign;
    showInstanceIcon?: boolean;
  }>(),
  {
    showInstanceIcon: false,
  }
);

const database = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(props.branch.baselineDatabase);
});

const changeHistory = computed(() => {
  const { branch } = props;
  const { baselineChangeHistoryId } = branch;
  if (
    !baselineChangeHistoryId ||
    baselineChangeHistoryId === String(UNKNOWN_ID)
  ) {
    return undefined;
  }
  const name = `${database.value.name}/changeHistories/${branch.baselineChangeHistoryId}`;
  return useChangeHistoryStore().getChangeHistoryByName(name);
});

const baselineVersion = computed(() => {
  return changeHistory.value?.version ?? "Previously latest schema";
});
</script>
