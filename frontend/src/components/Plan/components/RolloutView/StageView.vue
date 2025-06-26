<template>
  <div class="w-full h-full flex flex-col p-4">
    <div class="w-full flex grow overflow-auto bg-zinc-50 p-6 rounded-lg">
      <div
        class="relative w-auto flex flex-row items-start justify-start gap-6"
      >
        <div class="absolute top-5 border-2 w-full"></div>
        <StageCard
          v-for="stage in rollout.stages"
          :key="stage.name"
          :stage="stage"
          :rollout="rollout"
          :task-status-filter="taskStatusFilter"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Rollout, Task_Status } from "@/types/proto/v1/rollout_service";
import StageCard from "./StageCard.vue";

defineProps<{
  rollout: Rollout;
  taskStatusFilter: Task_Status[];
}>();
</script>
