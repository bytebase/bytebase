import { taskRunNamePrefix } from "@/store";
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
 * Get first line preview of error detail, truncated to maxLength.
 */
export const getErrorPreview = (detail: string, maxLength = 50): string => {
  const firstLine = detail.split("\n")[0];
  return firstLine.length > maxLength
    ? `${firstLine.substring(0, maxLength)}...`
    : firstLine;
};
