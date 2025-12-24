import { taskRunNamePrefix } from "@/store";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import {
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";

/**
 * Extract task name from task run name.
 * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun}
 */
export const getTaskNameFromTaskRun = (taskRunName: string): string => {
  return taskRunName.replace(
    `/${taskRunNamePrefix}${taskRunName.split("/").pop()}`,
    ""
  );
};

/**
 * Map TaskRun_Status to Task_Status for TaskStatus component.
 */
export const mapTaskRunStatusToTaskStatus = (
  status: TaskRun_Status
): Task_Status => {
  switch (status) {
    case TaskRun_Status.PENDING:
      return Task_Status.PENDING;
    case TaskRun_Status.RUNNING:
      return Task_Status.RUNNING;
    case TaskRun_Status.DONE:
      return Task_Status.DONE;
    case TaskRun_Status.FAILED:
      return Task_Status.FAILED;
    case TaskRun_Status.CANCELED:
      return Task_Status.CANCELED;
    default:
      return Task_Status.STATUS_UNSPECIFIED;
  }
};

export type TimelineType = "success" | "error" | "warning" | "info" | "default";

/**
 * Get NTimeline type from TaskRun_Status.
 */
export const getTimelineType = (status: TaskRun_Status): TimelineType => {
  switch (status) {
    case TaskRun_Status.DONE:
      return "success";
    case TaskRun_Status.FAILED:
      return "error";
    case TaskRun_Status.RUNNING:
      return "info";
    case TaskRun_Status.PENDING:
      return "warning";
    case TaskRun_Status.CANCELED:
      return "default";
    default:
      return "default";
  }
};

/**
 * Format duration in milliseconds to human-readable string.
 * - >= 1h: "Xh Ym"
 * - >= 1m: "Xm Ys"
 * - >= 1s: "Xs"
 * - > 0ms: "Xms"
 * - <= 0: "< 1s"
 */
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

  if (seconds > 0) {
    return `${seconds}s`;
  }

  if (durationMs > 0) {
    return `${Math.round(durationMs)}ms`;
  }

  return "< 1s";
};

/**
 * Get duration display string for a task run.
 * - PENDING: empty (not started yet)
 * - RUNNING: elapsed time from startTime to now
 * - Completed (DONE/FAILED/CANCELED): total duration from startTime to updateTime
 */
export const getTaskRunDuration = (taskRun: TaskRun): string => {
  const { startTime, status, updateTime } = taskRun;

  // PENDING tasks are scheduled/queued, not running - no duration to show
  if (status === TaskRun_Status.PENDING) {
    return "";
  }

  if (!startTime) {
    return "";
  }

  if (status === TaskRun_Status.RUNNING) {
    // Show elapsed time for running tasks
    const elapsedMs = Date.now() - Number(startTime.seconds) * 1000;
    return formatDuration(elapsedMs);
  }

  // For completed tasks (DONE, FAILED, CANCELED), show total duration
  if (!updateTime) {
    return "";
  }

  const durationMs =
    (Number(updateTime.seconds) - Number(startTime.seconds)) * 1000;
  return formatDuration(durationMs);
};
