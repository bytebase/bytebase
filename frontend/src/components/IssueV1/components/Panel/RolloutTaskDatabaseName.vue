<template>
  <div class="flex items-center flex-1 gap-x-1 overflow-hidden">
    <InstanceV1Name
      :instance="db.instanceEntity"
      :plain="true"
      :link="false"
      text-class="shrink-0 whitespace-nowrap"
    />

    <heroicons-outline:chevron-right class="shrink-0 text-control-light" />

    <NPerformantEllipsis class="flex-1">
      {{ db.databaseName }}
    </NPerformantEllipsis>
  </div>
</template>

<script setup lang="ts">
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { Task } from "@/types/proto/v1/rollout_service";
import { databaseForTask, useIssueContext } from "../../logic";

const props = defineProps<{
  task: Task;
}>();

const { issue } = useIssueContext();

const db = computed(() => {
  return databaseForTask(issue.value, props.task);
});
</script>
