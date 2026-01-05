import { computed, type Ref } from "vue";
import { useCurrentProjectV1 } from "@/store";
import { taskRunNamePrefix } from "@/store/modules/v1/common";
import type {
  Stage,
  Task,
  TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";

export type RollbackableTaskRun = {
  task: Task;
  taskRun: TaskRun;
  database: ReturnType<typeof databaseForTask>;
};

export const useRollbackableTasks = (
  stage: Ref<Stage | null | undefined>,
  taskRuns: Ref<TaskRun[]>
) => {
  const { project } = useCurrentProjectV1();

  const rollbackableTaskRuns = computed<RollbackableTaskRun[]>(() => {
    if (!stage.value) {
      return [];
    }

    const result: RollbackableTaskRun[] = [];

    for (const task of stage.value.tasks) {
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

    return result;
  });

  return {
    rollbackableTaskRuns,
  };
};
