<template>
  <BBProgressPie
    :percent="percent"
    class="w-10 h-10 issue-debug"
    :class="task.status === Task_Status.DONE ? 'text-success' : 'text-info'"
    :thickness="3"
  >
    <!-- eslint-disable vue/no-template-shadow -->
    <template #default="{ percent }">
      <span v-if="task.status !== Task_Status.DONE" class="text-xs">
        {{ percent }}%
      </span>
      <span v-else>
        <heroicons-outline:check class="w-8 h-8" />
      </span>
    </template>
  </BBProgressPie>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { BBProgressPie } from "@/bbkit";
import { Task, Task_Status } from "@/types/proto/v1/rollout_service";

defineProps<{
  task: Task;
}>();

const percent = computed(() => {
  return 66; // todo
});
</script>
