// Frontier-stage helpers shared by the deploy section and the lifecycle header.
// The "frontier" is the first stage that is not yet complete — the single stage
// the header's lifecycle advance points at. A stage is complete when every task
// is done or skipped.
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { RUNNABLE_TASK_STATUSES } from "../../../issue-detail/utils/rollout";

export function isStageComplete(stage: Stage): boolean {
  return stage.tasks.every(
    (task) =>
      task.status === Task_Status.DONE || task.status === Task_Status.SKIPPED
  );
}

// The first non-complete stage, or undefined when every stage is complete.
export function getFrontierStage(
  rollout: Rollout | undefined
): Stage | undefined {
  return rollout?.stages.find((stage) => !isStageComplete(stage));
}

export function stageHasRunnableTasks(stage: Stage): boolean {
  return stage.tasks.some((task) =>
    RUNNABLE_TASK_STATUSES.includes(task.status)
  );
}

// A task that already executed once (failed or canceled) — running it again is
// a re-run ("Rerun") rather than a first run ("Run"). The single source for the
// verb choice at both the task and stage level.
export function isRerunnableTaskStatus(status: Task_Status): boolean {
  return status === Task_Status.FAILED || status === Task_Status.CANCELED;
}

// Named in full so it isn't mistaken for stageHasRunningTasks.
export function stageHasFailedOrCanceledTasks(stage: Stage): boolean {
  return stage.tasks.some((task) => isRerunnableTaskStatus(task.status));
}

export function stageHasRunningTasks(stage: Stage): boolean {
  return stage.tasks.some(
    (task) =>
      task.status === Task_Status.RUNNING || task.status === Task_Status.PENDING
  );
}
