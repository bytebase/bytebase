import {
  Task,
  StageId,
  Stage,
  EnvironmentId,
  DatabaseId,
  Step,
  FINAL_STAGE,
} from "../types";

export function stageName(task: Task, stageId: StageId): string {
  for (const stage of task.stageList) {
    if (stage.id == stageId) {
      return stage.name;
    }
  }
  return "<<Unknown stage>>";
}

// Returns true if the active step is the last step in the entire task
export function pendingResolve(task: Task): boolean {
  return activeStage(task).type == "bytebase.stage.final";
}

export function activeStage(task: Task): Stage {
  for (const stage of task.stageList) {
    if (
      stage.status === "PENDING" ||
      stage.status === "RUNNING" ||
      // "FAILED" is also a transient stage status, which requires user
      // to take further action (e.g. Cancel, Skip, Retry)
      stage.status === "FAILED"
    ) {
      return stage;
    }
  }
  return FINAL_STAGE;
}

export function activeStageIsRunning(task: Task): boolean {
  return activeStage(task).status === "RUNNING";
}

export function activeEnvironmentId(task: Task): EnvironmentId {
  const stage: Stage = activeStage(task);
  return stage.database.instance.environment.id;
}

export function activeDatabaseId(task: Task): DatabaseId {
  const stage = activeStage(task);
  return stage.database.id;
}
