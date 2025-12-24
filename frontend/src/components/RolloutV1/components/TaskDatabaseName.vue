<template>
  <DatabaseDisplay :database="task.target" />
</template>

<script setup lang="ts">
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";

defineProps<{
  task: Task;
}>();
</script>
