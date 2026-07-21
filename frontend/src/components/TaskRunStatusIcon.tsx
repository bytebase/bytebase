import { TaskStatusIcon } from "@/components/TaskStatusIcon";
import {
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";

// A task run is a single execution attempt, so it only ever reaches a subset of
// Task_Status. Map the run status onto the canonical TaskStatusIcon so run
// history shares the exact same status vocabulary as tasks and stages.
function toTaskStatus(status: TaskRun_Status): Task_Status {
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
}

/**
 * Canonical status icon for a task run. Renders the shared `TaskStatusIcon`
 * so run history stays visually consistent with task and stage status.
 */
export function TaskRunStatusIcon({
  status,
  size = "small",
}: {
  status: TaskRun_Status;
  size?: "tiny" | "small" | "medium" | "large";
}) {
  return <TaskStatusIcon size={size} status={toTaskStatus(status)} />;
}
