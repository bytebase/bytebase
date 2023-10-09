<template>
  <div class="flex flex-row gap-x-1 text-sm">
    <RichDatabaseName
      :database="database"
      :show-instance="false"
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

const props = defineProps<{
  branch: SchemaDesign;
}>();

const database = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(props.branch.baselineDatabase);
});

const baselineVersion = computed(() => {
  const { branch } = props;
  const changeHistory =
    branch.baselineChangeHistoryId &&
    branch.baselineChangeHistoryId !== String(UNKNOWN_ID)
      ? useChangeHistoryStore().getChangeHistoryByName(
          `${database.value.name}/changeHistories/${branch.baselineChangeHistoryId}`
        )
      : undefined;
  return changeHistory?.version ?? "Previously latest schema";
});
</script>
