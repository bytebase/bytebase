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
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { RichDatabaseName } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
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
</script>
