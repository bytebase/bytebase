import type { ComputedRef } from "vue";
import { computed } from "vue";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

export type TimingType = "scheduled" | "running" | "completed" | "none";

export interface UseTaskTimingReturn {
  timingDisplay: ComputedRef<string>;
  timingType: ComputedRef<TimingType>;
  duration: ComputedRef<number | undefined>;
  scheduledTime: ComputedRef<Date | undefined>;
}

export const useTaskTiming = (
  task: () => Task,
  latestTaskRun: () => TaskRun | undefined
): UseTaskTimingReturn => {
  const duration = computed(() => {
    const taskRun = latestTaskRun();
    if (!taskRun) return undefined;

    const startTime = taskRun.startTime;
    const updateTime = taskRun.updateTime;

    if (!startTime || !updateTime) {
      return undefined;
    }

    const startMs = Number(startTime.seconds) * 1000;
    const updateMs = Number(updateTime.seconds) * 1000;
    return updateMs - startMs;
  });

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
      (status === Task_Status.DONE ||
        status === Task_Status.FAILED ||
        status === Task_Status.CANCELED) &&
      duration.value !== undefined
    ) {
      return "completed";
    }

    return "none";
  });

  const timingDisplay = computed(() => {
    const type = timingType.value;

    if (type === "completed" && duration.value !== undefined) {
      return formatDuration(duration.value);
    }

    if (type === "running" && duration.value !== undefined) {
      return formatDuration(duration.value);
    }

    if (type === "scheduled" && scheduledTime.value) {
      return scheduledTime.value.toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      });
    }

    return "";
  });

  return {
    timingDisplay,
    timingType,
    duration,
    scheduledTime,
  };
};

export const formatDuration = (durationMs: number): string => {
  const seconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    const remainingMinutes = minutes % 60;
    return `${hours}h ${remainingMinutes}m`;
  }

  if (minutes > 0) {
    const remainingSeconds = seconds % 60;
    return `${minutes}m ${remainingSeconds}s`;
  }

  return `${seconds}s`;
};
