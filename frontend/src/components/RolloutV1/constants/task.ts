import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

export const TASK_STATUS_FILTERS: Task_Status[] = [
  Task_Status.RUNNING,
  Task_Status.FAILED,
  Task_Status.PENDING,
  Task_Status.NOT_STARTED,
  Task_Status.CANCELED,
  Task_Status.DONE,
  Task_Status.SKIPPED,
];
