import { TASK_STATUS_FILTERS } from "@/components/Plan/constants/task";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

/**
 * Task statuses that are considered "unfinished"
 */
export const UNFINISHED_TASK_STATUSES: Task_Status[] = [
  Task_Status.NOT_STARTED,
  Task_Status.PENDING,
  Task_Status.RUNNING,
  Task_Status.FAILED,
  Task_Status.CANCELED,
];

/**
 * Task statuses that allow task selection for bulk actions
 */
export const SELECTABLE_TASK_STATUSES: Task_Status[] = [
  Task_Status.NOT_STARTED,
  Task_Status.PENDING,
  Task_Status.RUNNING,
  Task_Status.FAILED,
  Task_Status.CANCELED,
];

/**
 * Check if a task is in an unfinished state
 */
export const isTaskUnfinished = (task: Task): boolean => {
  return UNFINISHED_TASK_STATUSES.includes(task.status);
};

/**
 * Check if a task is selectable for bulk actions
 */
export const isTaskSelectable = (task: Task): boolean => {
  return SELECTABLE_TASK_STATUSES.includes(task.status);
};

/**
 * Check if a task is running
 */
export const isTaskRunning = (task: Task): boolean => {
  return task.status === Task_Status.RUNNING;
};

/**
 * Get border and background CSS classes for a task based on its status
 */
export const getTaskBorderClass = (status: Task_Status): string => {
  switch (status) {
    case Task_Status.FAILED:
      return "border-red-300";
    default:
      return "border-gray-300";
  }
};

/**
 * Create a Map of task status to sort priority based on TASK_STATUS_FILTERS order.
 * This ensures a single source of truth for task ordering.
 *
 * @returns Map where keys are Task_Status and values are their sort priority (lower = higher priority)
 */
const createTaskStatusPriorityMap = (): Map<Task_Status, number> => {
  const priorityMap = new Map<Task_Status, number>();
  TASK_STATUS_FILTERS.forEach((status, index) => {
    priorityMap.set(status, index);
  });
  return priorityMap;
};

/**
 * Cached task status priority map
 */
const taskStatusPriorityMap = createTaskStatusPriorityMap();

/**
 * Get the sort priority for a task status.
 * Lower values indicate higher priority in the display order.
 *
 * @param status - The task status
 * @returns The sort priority (0-based index from TASK_STATUS_FILTERS)
 */
export const getTaskStatusPriority = (status: Task_Status): number => {
  return taskStatusPriorityMap.get(status) ?? Number.MAX_SAFE_INTEGER;
};

/**
 * Comparator function for sorting tasks by status priority.
 * Use with Array.sort() to order tasks according to TASK_STATUS_FILTERS.
 *
 * @example
 * const sortedTasks = tasks.slice().sort(compareTasksByStatus);
 *
 * @param a - First task to compare
 * @param b - Second task to compare
 * @returns Negative if a should come before b, positive if b should come before a, 0 if equal
 */
export const compareTasksByStatus = (a: Task, b: Task): number => {
  const priorityA = getTaskStatusPriority(a.status);
  const priorityB = getTaskStatusPriority(b.status);
  return priorityA - priorityB;
};
