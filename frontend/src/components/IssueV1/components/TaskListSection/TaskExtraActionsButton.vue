<template>
  <div class="issue-debug">
    <heroicons:ellipsis-vertical-solid class="w-4 h-4" />
  </div>
</template>

<script setup lang="ts">
import { Task } from "@/types/proto/v1/rollout_service";

defineProps<{
  task: Task;
}>();
</script>
