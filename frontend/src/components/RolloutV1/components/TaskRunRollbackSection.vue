<template>
  <div v-if="rollbackableTaskRuns.length > 0" class="w-full mt-3 pl-4">
    <NButton
      size="small"
      text
      @click="showRollbackDrawer = true"
      icon-placement="right"
    >
      <template #icon>
        <DatabaseBackupIcon class="w-4 h-auto" />
      </template>
      {{ $t("task-run.rollback.available", rollbackableTaskRuns.length) }}
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
import { DatabaseBackupIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { usePlanContextWithRollout } from "@/components/Plan/logic";
import { taskRunNamePrefix, useCurrentProjectV1 } from "@/store";
import type {
  Rollout,
  Task,
  TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";
import TaskRunRollbackDrawer from "./TaskRunRollbackDrawer.vue";

const props = defineProps<{
  rollout: Rollout;
}>();

const { project } = useCurrentProjectV1();
const { taskRuns } = usePlanContextWithRollout();

const showRollbackDrawer = ref(false);

// Get rollbackable task runs
const rollbackableTaskRuns = computed(() => {
  const result: Array<{
    task: Task;
    taskRun: TaskRun;
    database: ReturnType<typeof databaseForTask>;
  }> = [];

  for (const stage of props.rollout.stages) {
    for (const task of stage.tasks) {
      // Find the latest successful task run with prior backup
      const rollbackableRun = taskRuns.value.find(
        (run) =>
          run.name.startsWith(`${task.name}/${taskRunNamePrefix}`) &&
          run.status === TaskRun_Status.DONE &&
          run.hasPriorBackup
      );

      if (rollbackableRun) {
        result.push({
          task,
          taskRun: rollbackableRun,
          database: databaseForTask(project.value, task),
        });
      }
    }
  }

  return result;
});
</script>
