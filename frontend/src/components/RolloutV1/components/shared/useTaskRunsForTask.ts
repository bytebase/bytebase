import type { ComputedRef } from "vue";
import { computed } from "vue";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";

export const useTaskRunsForTask = (
  taskName: () => string,
  allTaskRuns: () => TaskRun[]
): ComputedRef<TaskRun[]> => {
  return computed(() => {
    const name = taskName();
    const prefix = `${name}/taskRuns/`;
    return allTaskRuns().filter((run) => run.name.startsWith(prefix));
  });
};
