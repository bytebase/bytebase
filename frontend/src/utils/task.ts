import { Task, StageProgress, EnvironmentId } from "../types";
import { templateForType } from "../plugins";

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

export function activeStageIsRunning(task: Task): boolean {
  return activeStage(task).status === "RUNNING";
}

export function activeEnvironmentId(task: Task): EnvironmentId | null {
  const stageProgress = activeStage(task);
  if (stageProgress.type === "ENVIRONMENT") {
    return stageProgress.environmentId!;
  }
  const taskTemplate = templateForType(task.attributes.type);
  if (taskTemplate) {
    const inputFieldList =
      taskTemplate.fieldList?.filter((item) => item.category === "INPUT") || [];
    for (const field of inputFieldList) {
      if (field.type === "Environment") {
        return task.attributes.payload[field.id];
      }
    }
  }
  return null;
}
