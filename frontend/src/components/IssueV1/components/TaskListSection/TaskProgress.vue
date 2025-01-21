<template>
  <NProgress
    type="circle"
    :percentage="percent"
    style="width: 2.5rem"
    :stroke-width="10"
  >
    <span>
      <heroicons-outline:check class="w-8 h-8 text-success" />
    </span>
  </NProgress>
</template>

<script setup lang="ts">
import { NProgress } from "naive-ui";
import { computed } from "vue";
import type { Task } from "@/types/proto/v1/rollout_service";

defineProps<{
  task: Task;
}>();

const percent = computed(() => {
  return 0; // TODO
});
</script>
