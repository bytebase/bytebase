<template>
  <NProgress
    type="circle"
    :percentage="percent"
    style="width: 2.5rem"
    :stroke-width="10"
  >
    <span
      v-if="task.status !== Task_Status.DONE"
      class="text-xs text-info issue-debug"
    >
      {{ `${Math.round(percent)}%` }}
    </span>
    <span v-else>
      <heroicons-outline:check class="w-8 h-8 text-success" />
    </span>
  </NProgress>
</template>

<script setup lang="ts">
import { NProgress } from "naive-ui";
import { computed } from "vue";
import { Task, Task_Status } from "@/types/proto/v1/rollout_service";

defineProps<{
  task: Task;
}>();

const percent = computed(() => {
  return 0; // TODO
});
</script>
