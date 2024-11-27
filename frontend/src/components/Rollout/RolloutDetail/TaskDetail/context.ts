import { isEqual, sortBy } from "lodash-es";
import type { ComputedRef, InjectionKey } from "vue";
import { computed, inject, provide, ref, watchEffect } from "vue";
import { rolloutServiceClient } from "@/grpcweb";
import { getDateForPbTimestamp, unknownStage, unknownTask } from "@/types";
import type { Stage, Task, TaskRun } from "@/types/proto/v1/rollout_service";
import { isValidTaskName } from "@/utils";
import { useRolloutDetailContext } from "../context";
import { stageForTask } from "./utils";

export type TaskDetailContext = {
  stage: ComputedRef<Stage>;
  task: ComputedRef<Task>;
  taskRuns: ComputedRef<TaskRun[]>;
};

export const KEY = Symbol(
  "bb.rollout.task.detail"
) as InjectionKey<TaskDetailContext>;

export const useTaskDetailContext = () => {
  return inject(KEY)!;
};

export const provideTaskDetailContext = (stageId: string, taskId: string) => {
  const { rollout, tasks } = useRolloutDetailContext();
  const taskRunsRef = ref<TaskRun[]>([]);

  const task = computed(() => {
    return (
      tasks.value.find((task) => task.name.endsWith(`/${taskId}`)) ||
      unknownTask()
    );
  });

  const stage = computed(
    () => stageForTask(rollout.value, task.value) || unknownStage()
  );

  watchEffect(async () => {
    if (!isValidTaskName(task.value.name)) {
      return;
    }

    // Prepare task runs.
    const { taskRuns } = await rolloutServiceClient.listTaskRuns({
      parent: task.value.name,
    });
    const sorted = sortBy(taskRuns, (t) =>
      getDateForPbTimestamp(t.createTime)
    ).reverse();
    if (!isEqual(sorted, taskRunsRef.value)) {
      taskRunsRef.value = sorted;
    }
  });

  const context: TaskDetailContext = {
    stage,
    task,
    taskRuns: computed(() => taskRunsRef.value),
  };

  provide(KEY, context);

  return context;
};
