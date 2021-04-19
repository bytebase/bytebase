import {
  Issue,
  StageId,
  Stage,
  EnvironmentId,
  DatabaseId,
  Step,
  FINAL_STAGE,
  ZERO_ID,
} from "../types";

export function stageName(issue: Issue, stageId: StageId): string {
  for (const stage of issue.stageList) {
    if (stage.id == stageId) {
      return stage.name;
    }
  }
  return "<<Unknown stage>>";
}

// Returns true if the active step is the last step in the entire issue
export function pendingResolve(issue: Issue): boolean {
  return activeStage(issue).type == "bytebase.stage.final";
}

export function activeStage(issue: Issue): Stage {
  for (const stage of issue.stageList) {
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

export function activeStageIsRunning(issue: Issue): boolean {
  return activeStage(issue).status === "RUNNING";
}

export function activeEnvironmentId(issue: Issue): EnvironmentId {
  const stage: Stage = activeStage(issue);
  if (stage.id == ZERO_ID) {
    return ZERO_ID;
  }
  return stage.database.instance.environment.id;
}

export function activeDatabaseId(issue: Issue): DatabaseId {
  const stage = activeStage(issue);
  if (stage.id == ZERO_ID) {
    return ZERO_ID;
  }
  return stage.database.id;
}
