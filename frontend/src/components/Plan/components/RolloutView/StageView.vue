<template>
  <div class="w-full h-full flex flex-col p-4">
    <div class="w-full flex grow overflow-auto bg-zinc-50 p-6 rounded-lg">
      <div
        class="relative w-auto flex flex-row items-start justify-start gap-8"
      >
        <!-- Connecting line -->
        <div class="absolute top-5 border-t-2 border-gray-200 w-full z-0"></div>

        <template v-for="stage in mergedStages" :key="stage.name">
          <StageCard :stage="stage" :task-status-filter="taskStatusFilter" />
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Stage, Task_Status } from "@/types/proto/v1/rollout_service";
import StageCard from "./StageCard.vue";

defineProps<{
  mergedStages: Stage[];
  taskStatusFilter: Task_Status[];
}>();
</script>
