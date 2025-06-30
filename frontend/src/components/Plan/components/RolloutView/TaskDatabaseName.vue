<template>
  <DatabaseDisplay :database="task.target" />
</template>

<script setup lang="ts">
import type { Task } from "@/types/proto/v1/rollout_service";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";

defineProps<{
  task: Task;
}>();
</script>
