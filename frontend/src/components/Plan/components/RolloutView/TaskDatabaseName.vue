<template>
  <DatabaseDisplay :database="task.target" />
</template>

<script setup lang="ts">
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";

defineProps<{
  task: Task;
}>();
</script>
