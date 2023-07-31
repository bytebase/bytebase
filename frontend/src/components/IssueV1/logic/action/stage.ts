import { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Stage, Task } from "@/types/proto/v1/rollout_service";
import { getApplicableTaskRolloutActionList } from "./task";

export type StageRolloutAction =
  | "ROLLOUT" // PENDING_APPROVAL -> PENDING
  | "RETRY" // FAILED -> PENDING_APPROVAL
  | "SKIP"; // ? -> SKIPPED

export const getApplicableStageRolloutActionList = (
  issue: ComposedIssue,
  stage: Stage,
  allowSkipPendingTasks = false // If set to true, only FAILED tasks can be skipped
) => {
  if (issue.status !== IssueStatus.OPEN) {
    return [];
  }

  const applicableActionsMap: Record<StageRolloutAction, Task[]> = {
    ROLLOUT: [],
    RETRY: [],
    SKIP: [],
  };
  stage.tasks.forEach((task) => {
    const actions = getApplicableTaskRolloutActionList(
      issue,
      task,
      allowSkipPendingTasks
    );
    if (actions.includes("ROLLOUT")) {
      applicableActionsMap.ROLLOUT.push(task);
    }
    if (actions.includes("RETRY")) {
      applicableActionsMap.RETRY.push(task);
    }
    if (actions.includes("SKIP")) {
      applicableActionsMap.SKIP.push(task);
    }
  });

  const actions: { action: StageRolloutAction; tasks: Task[] }[] = [];
  if (applicableActionsMap.ROLLOUT.length > 1) {
    actions.push({
      action: "ROLLOUT",
      tasks: applicableActionsMap.ROLLOUT,
    });
  }
  if (applicableActionsMap.RETRY.length > 1) {
    actions.push({
      action: "RETRY",
      tasks: applicableActionsMap.RETRY,
    });
  }
  if (applicableActionsMap.SKIP.length > 1) {
    actions.push({
      action: "SKIP",
      tasks: applicableActionsMap.SKIP,
    });
  }
  return actions;
};
