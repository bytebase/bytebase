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
      :readonly="readonly"
      @run-tasks="(stage, tasks) => $emit('run-tasks', stage, tasks)"
      @create-rollout-to-stage="
        (stage) => $emit('create-rollout-to-stage', stage)
      "
    />
  </div>
</template>

<script setup lang="ts">
import type {
  Stage,
  Task,
  Task_Status,
  Rollout,
} from "@/types/proto-es/v1/rollout_service_pb";
import StageCard from "./StageCard.vue";

defineProps<{
  rollout: Rollout;
  mergedStages: Stage[];
  taskStatusFilter?: Task_Status[];
  readonly?: boolean;
}>();

defineEmits<{
  (event: "run-tasks", stage: Stage, tasks: Task[]): void;
  (event: "create-rollout-to-stage", stage: Stage): void;
}>();
</script>
