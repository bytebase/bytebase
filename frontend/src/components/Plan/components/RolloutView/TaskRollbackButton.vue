<template>
  <div>
    <NButton
      size="small"
      @click="showRollbackDrawer = true"
      icon-placement="left"
    >
      <template #icon>
        <Undo2Icon class="w-4 h-auto" />
      </template>
      {{ $t("common.rollback") }}
    </NButton>

    <TaskRunRollbackDrawer
      v-model:show="showRollbackDrawer"
      :rollout="rollout"
      :rollbackable-task-runs="rollbackableTaskRuns"
      @close="showRollbackDrawer = false"
    />
  </div>
</template>

<script setup lang="ts">
import { Undo2Icon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useCurrentProjectV1 } from "@/store";
import type {
  Rollout,
  Task,
  TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";
import TaskRunRollbackDrawer from "./TaskRunRollbackDrawer.vue";

const props = defineProps<{
  task: Task;
  taskRun: TaskRun;
  rollout: Rollout;
}>();

const { project } = useCurrentProjectV1();

const showRollbackDrawer = ref(false);

// Prepare rollbackable task run data for the drawer
const rollbackableTaskRuns = computed(() => {
  return [
    {
      task: props.task,
      taskRun: props.taskRun,
      database: databaseForTask(project.value, props.task),
    },
  ];
});
</script>
