<template>
  <NButton v-if="hasRunnableTasks" size="small" @click="handleRunTasks">
    <template #icon>
      <PlayIcon class="w-4 h-4" />
    </template>
    {{ $t("common.rollout") }}
  </NButton>
</template>

<script setup lang="ts">
import { PlayIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import type { Stage } from "@/types/proto/v1/rollout_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";

const props = defineProps<{
  stage: Stage;
}>();

const emit = defineEmits<{
  (event: "run-tasks"): void;
}>();

const hasRunnableTasks = computed(() => {
  return props.stage.tasks.some(
    (task) =>
      task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.PENDING ||
      task.status === Task_Status.FAILED
  );
});

const handleRunTasks = () => {
  emit("run-tasks");
};
</script>
