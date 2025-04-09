<template>
  <div class="text-gray-500">
    <span>(</span>
    <template v-if="!isCreating">
      <span>{{ summary.done + summary.canceled }}</span>
      <span>/</span>
    </template>
    <span>{{ summary.total }}</span>
    <span>)</span>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Stage } from "@/types/proto/v1/rollout_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { useIssueContext } from "../../logic";

const props = defineProps<{
  stage: Stage;
}>();

const { isCreating } = useIssueContext();

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
