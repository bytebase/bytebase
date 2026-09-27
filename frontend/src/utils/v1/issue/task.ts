import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

/**
 * Task statuses that allow run/skip actions.
 * Tasks in these states can be started, restarted, or skipped.
 */
export const RUNNABLE_TASK_STATUSES: Task_Status[] = [
  Task_Status.NOT_STARTED,
  Task_Status.CANCELED,
  Task_Status.FAILED,
];

/**
 * Task statuses that allow cancel action.
 * Tasks in these states are currently executing or queued.
 */
export const CANCELABLE_TASK_STATUSES: Task_Status[] = [
  Task_Status.PENDING,
  Task_Status.RUNNING,
];
