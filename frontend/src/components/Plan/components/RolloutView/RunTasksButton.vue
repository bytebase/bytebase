<template>
  <NButton
    v-if="hasRunnableTasks && canRunTasks"
    size="tiny"
    @click="handleRunTasks"
  >
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
import { useCurrentProjectV1 } from "@/store";
import type { Stage } from "@/types/proto/v1/rollout_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { useRolloutViewContext } from "./context";
import { useTaskActionPermissions } from "./taskPermissions";

const props = defineProps<{
  stage: Stage;
}>();

const emit = defineEmits<{
  (event: "run-tasks"): void;
}>();

const { project } = useCurrentProjectV1();
const { rollout } = useRolloutViewContext();
const { canPerformTaskAction } = useTaskActionPermissions();

const hasRunnableTasks = computed(() => {
  return props.stage.tasks.some(
    (task) =>
      task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.PENDING ||
      task.status === Task_Status.FAILED
  );
});

const canRunTasks = computed(() => {
  if (!rollout.value) {
    return false;
  }
  return canPerformTaskAction(props.stage.tasks, rollout.value, project.value);
});

const handleRunTasks = () => {
  emit("run-tasks");
};
</script>
