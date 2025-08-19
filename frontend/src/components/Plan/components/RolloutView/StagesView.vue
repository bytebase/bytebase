<template>
  <div class="relative w-full flex flex-col items-stretch justify-start gap-6">
    <!-- Connecting lines between stages -->
    <div
      v-if="mergedStages.length > 1"
      class="absolute left-6 top-10 bottom-10 w-0.5 bg-gray-200 z-0"
    ></div>

    <StageCard
      v-for="stage in mergedStages"
      :key="stage.name"
      :rollout="rollout"
      :stage="stage"
      :task-status-filter="taskStatusFilter"
      :default-show-tasks="defaultShowTasksStage === stage.name"
      @run-tasks="(stage, tasks) => $emit('run-tasks', stage, tasks)"
      @create-rollout-to-stage="
        (stage) => $emit('create-rollout-to-stage', stage)
      "
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type {
  Stage,
  Task,
  Task_Status,
  Rollout,
} from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status as TaskStatus } from "@/types/proto-es/v1/rollout_service_pb";
import StageCard from "./StageCard.vue";

const props = defineProps<{
  rollout: Rollout;
  mergedStages: Stage[];
  taskStatusFilter?: Task_Status[];
}>();

defineEmits<{
  (event: "run-tasks", stage: Stage, tasks: Task[]): void;
  (event: "create-rollout-to-stage", stage: Stage): void;
}>();

// Determine which stage should show tasks by default
const defaultShowTasksStage = computed(() => {
  // First, check if any stage has running tasks
  const stageWithRunningTasks = props.mergedStages.find((stage) =>
    stage.tasks.some((task) => task.status === TaskStatus.RUNNING)
  );
  if (stageWithRunningTasks) {
    return stageWithRunningTasks.name;
  }

  // If no running tasks, find the first stage with unfinished tasks
  const firstUnfinishedStage = props.mergedStages.find((stage) =>
    stage.tasks.some(
      (task) =>
        task.status !== TaskStatus.DONE &&
        task.status !== TaskStatus.SKIPPED &&
        task.status !== TaskStatus.CANCELED
    )
  );
  if (firstUnfinishedStage) {
    return firstUnfinishedStage.name;
  }

  // If all stages are finished, show tasks for the last stage
  return props.mergedStages.length > 0
    ? props.mergedStages[props.mergedStages.length - 1].name
    : null;
});
</script>
