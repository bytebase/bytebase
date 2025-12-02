import type { ComputedRef } from "vue";
import { computed } from "vue";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { getTaskRunDuration } from "./useTaskRunUtils";

export type TimingType = "scheduled" | "running" | "completed" | "none";

export interface UseTaskTimingReturn {
  timingDisplay: ComputedRef<string>;
  timingType: ComputedRef<TimingType>;
  scheduledTime: ComputedRef<Date | undefined>;
}

export const useTaskTiming = (
  task: () => Task,
  latestTaskRun: () => TaskRun | undefined
): UseTaskTimingReturn => {
  const scheduledTime = computed(() => {
    const currentTask = task();
    if (currentTask.status === Task_Status.PENDING && currentTask.runTime) {
      const runTimeMs = Number(currentTask.runTime.seconds) * 1000;
      if (runTimeMs > Date.now()) {
        return new Date(runTimeMs);
      }
    }
    return undefined;
  });

  const timingType = computed((): TimingType => {
    const currentTask = task();
    const status = currentTask.status;

    if (scheduledTime.value) {
      return "scheduled";
    }

    if (status === Task_Status.RUNNING) {
      return "running";
    }

    if (
      status === Task_Status.DONE ||
      status === Task_Status.FAILED ||
      status === Task_Status.CANCELED
    ) {
      return "completed";
    }

    return "none";
  });

  const timingDisplay = computed(() => {
    const taskRun = latestTaskRun();
    const type = timingType.value;

    if (type === "scheduled" && scheduledTime.value) {
      return scheduledTime.value.toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      });
    }

    // Use getTaskRunDuration for running and completed tasks
    if ((type === "running" || type === "completed") && taskRun) {
      return getTaskRunDuration(taskRun);
    }

    return "";
  });

  return {
    timingDisplay,
    timingType,
    scheduledTime,
  };
};
