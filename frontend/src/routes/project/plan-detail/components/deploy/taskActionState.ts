import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

const RUNNABLE_TASK_STATUSES: Task_Status[] = [
  Task_Status.NOT_STARTED,
  Task_Status.CANCELED,
  Task_Status.FAILED,
];

const CANCELABLE_TASK_STATUSES: Task_Status[] = [
  Task_Status.PENDING,
  Task_Status.RUNNING,
];

export interface DeployTaskActionState {
  canCancel: boolean;
  canRun: boolean;
  canSkip: boolean;
  hasActions: boolean;
}

export const getDeployTaskActionState = ({
  canPerformActions,
  status,
}: {
  canPerformActions: boolean;
  status: Task_Status;
}): DeployTaskActionState => {
  const canRun = canPerformActions && RUNNABLE_TASK_STATUSES.includes(status);
  const canSkip = canPerformActions && RUNNABLE_TASK_STATUSES.includes(status);
  const canCancel =
    canPerformActions && CANCELABLE_TASK_STATUSES.includes(status);

  return {
    canCancel,
    canRun,
    canSkip,
    hasActions: canRun || canSkip || canCancel,
  };
};

const ACTIONABLE_TASK_STATUSES = new Set<Task_Status>([
  Task_Status.NOT_STARTED,
  Task_Status.PENDING,
  Task_Status.RUNNING,
  Task_Status.FAILED,
  Task_Status.CANCELED,
]);

export const isDeployTaskSelectable = (task: { status: Task_Status }) =>
  ACTIONABLE_TASK_STATUSES.has(task.status);
