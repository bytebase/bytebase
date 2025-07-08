<template>
  <div class="relative w-fit flex flex-row items-start justify-start gap-8">
    <!-- Connecting line -->
    <div class="absolute top-5 border-t-2 border-gray-200 w-full z-0"></div>

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
