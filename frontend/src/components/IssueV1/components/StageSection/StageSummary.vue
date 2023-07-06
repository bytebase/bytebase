<template>
  <div class="!no-underline">
    <span class="!no-underline">(</span>
    <span class="!no-underline">{{ summary.done + summary.canceled }}</span>
    <span class="!no-underline">/</span>
    <span class="!no-underline">{{ summary.total }}</span>
    <span class="!no-underline">)</span>
  </div>
</template>

<script lang="ts" setup>
import { Stage, Task_Status } from "@/types/proto/v1/rollout_service";
import { computed } from "vue";

const props = defineProps<{
  stage: Stage;
}>();

const summary = computed(() => {
  const tasks = props.stage.tasks;
  const summary = {
    done: 0,
    failed: 0,
    canceled: 0,
    running: 0,
    total: tasks.length,
  };
  tasks.forEach((task) => {
    switch (task.status) {
      case Task_Status.DONE:
        summary.done++;
        break;
      case Task_Status.FAILED:
        summary.failed++;
        break;
      case Task_Status.CANCELED:
        summary.canceled++;
        break;
      case Task_Status.RUNNING:
        summary.running++;
        break;
    }
  });
  return summary;
});
</script>
