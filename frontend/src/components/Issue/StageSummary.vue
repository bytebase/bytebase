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
import { computed } from "vue";
import type { Stage } from "@/types";

const props = defineProps<{
  stage: Stage;
}>();

const summary = computed(() => {
  const { taskList } = props.stage;
  const summary = {
    done: 0,
    failed: 0,
    canceled: 0,
    running: 0,
    total: taskList.length,
  };
  taskList.forEach((task) => {
    switch (task.status) {
      case "DONE":
        summary.done++;
        break;
      case "FAILED":
        summary.failed++;
        break;
      case "CANCELED":
        summary.canceled++;
        break;
      case "RUNNING":
        summary.running++;
        break;
    }
  });
  return summary;
});
</script>
