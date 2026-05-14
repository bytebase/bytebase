import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import {
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";

const TASK_RUN_STATUS_TO_TASK_STATUS: Partial<
  Record<TaskRun_Status, Task_Status>
> = {
  [TaskRun_Status.DONE]: Task_Status.DONE,
  [TaskRun_Status.FAILED]: Task_Status.FAILED,
  [TaskRun_Status.RUNNING]: Task_Status.RUNNING,
  [TaskRun_Status.PENDING]: Task_Status.PENDING,
  [TaskRun_Status.AVAILABLE]: Task_Status.PENDING,
};

export const taskRunStatusToTaskStatus = (
  status: TaskRun_Status
): Task_Status =>
  TASK_RUN_STATUS_TO_TASK_STATUS[status] ?? Task_Status.CANCELED;

const formatDuration = (durationMs: number): string => {
  const seconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  }
  if (seconds > 0) {
    return `${seconds}s`;
  }
  if (durationMs > 0) {
    return `${Math.round(durationMs)}ms`;
  }
  return "< 1s";
};

export const getTaskRunDuration = (taskRun: TaskRun): string => {
  if (taskRun.status === TaskRun_Status.PENDING || !taskRun.startTime) {
    return "";
  }
  const endMs =
    taskRun.status === TaskRun_Status.RUNNING
      ? Date.now()
      : taskRun.updateTime
        ? Number(taskRun.updateTime.seconds) * 1000
        : 0;
  if (endMs === 0) {
    return "";
  }
  return formatDuration(endMs - Number(taskRun.startTime.seconds) * 1000);
};
