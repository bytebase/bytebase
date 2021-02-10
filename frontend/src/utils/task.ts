import { Task, StageProgress, EnvironmentId } from "../types";

export function activeStage(task: Task): StageProgress {
  for (const stageProgress of task.attributes.stageProgressList) {
    if (
      stageProgress.status === "PENDING" ||
      stageProgress.status === "RUNNING" ||
      // "FAILED" is also a transient stage status, which requires user
      // to take further action (e.g. Cancel, Skip, Retry)
      stageProgress.status === "FAILED"
    ) {
      return stageProgress;
    }
  }
  return task.attributes.stageProgressList[
    task.attributes.stageProgressList.length - 1
  ];
}

export function activeEnvironmentId(task: Task): EnvironmentId | null {
  const stageProgress = activeStage(task);
  if (stageProgress.type === "ENVIRONMENT") {
    return stageProgress.environmentId!;
  }
  return null;
}
