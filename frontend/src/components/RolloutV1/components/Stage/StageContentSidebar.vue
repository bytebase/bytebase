<template>
  <div class="w-full px-4 py-2">
    <StageTaskRunsRollback class="pb-2" :stage="stage" />
    <StageTimeline
      :stage="stage"
      :task-runs="taskRuns"
      mode="inline"
    />
  </div>
</template>

<script lang="ts" setup>
import type { Stage, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import StageTaskRunsRollback from "./StageTaskRunsRollback.vue";
import StageTimeline from "./StageTimeline.vue";

defineProps<{
  stage: Stage | null | undefined;
  taskRuns: TaskRun[];
}>();
</script>
