import { TASK_STATUS_FILTERS } from "@/components/RolloutV1/constants/task";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

/**
 * Task statuses that are considered "actionable" - not yet completed successfully.
 * Used for filtering unfinished tasks and determining if bulk actions are available.
 */
const ACTIONABLE_STATUSES = new Set<Task_Status>([
  Task_Status.NOT_STARTED,
  Task_Status.PENDING,
  Task_Status.RUNNING,
  Task_Status.FAILED,
  Task_Status.CANCELED,
]);

/**
 * Priority map for task status sorting.
 * Built from TASK_STATUS_FILTERS to maintain single source of truth.
 */
const STATUS_PRIORITY = new Map<Task_Status, number>(
  TASK_STATUS_FILTERS.map((status, index) => [status, index])
);

/**
 * Check if a task status is actionable (not completed).
 */
const isActionableStatus = (status: Task_Status): boolean =>
  ACTIONABLE_STATUSES.has(status);

/**
 * Check if a task is in an unfinished/actionable state.
 */
export const isTaskUnfinished = (task: Task): boolean =>
  isActionableStatus(task.status);

/**
 * Check if a task is selectable for bulk actions.
 * Currently same as unfinished - tasks that aren't done can be acted upon.
 */
export const isTaskSelectable = (task: Task): boolean =>
  isActionableStatus(task.status);

/**
 * Check if a task is currently running.
 */
export const isTaskRunning = (task: Task): boolean =>
  task.status === Task_Status.RUNNING;

/**
 * Get the sort priority for a task status.
 * Lower values = higher priority in display order.
 */
export const getTaskStatusPriority = (status: Task_Status): number =>
  STATUS_PRIORITY.get(status) ?? Number.MAX_SAFE_INTEGER;

/**
 * Compare tasks for sorting: first by status priority, then by update time.
 * Tasks with same status are sorted most recently updated first.
 *
 * @example
 * const sorted = [...tasks].sort(compareTasksByStatus);
 */
export const compareTasksByStatus = (a: Task, b: Task): number => {
  // Primary: status priority (lower = first)
  const statusDiff =
    getTaskStatusPriority(a.status) - getTaskStatusPriority(b.status);
  if (statusDiff !== 0) return statusDiff;

  // Secondary: update time (newer = first)
  const timeA = getTimeForPbTimestampProtoEs(a.updateTime, 0);
  const timeB = getTimeForPbTimestampProtoEs(b.updateTime, 0);
  return timeB - timeA;
};
