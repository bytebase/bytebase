import { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Stage, Task_Status } from "@/types/proto/v1/rollout_service";
import { getApplicableTaskRolloutActionList } from "./task";

export type StageRolloutAction =
  | "ROLLOUT" // PENDING_APPROVAL -> PENDING
  | "SKIP"; // ? -> SKIPPED

export const getApplicableStageRolloutActionList = (
  issue: ComposedIssue,
  stage: Stage
) => {
  if (issue.status !== IssueStatus.OPEN) {
    return [];
  }

  const taskAndActions = stage.tasks.map((task) => ({
    task,
    actions: getApplicableTaskRolloutActionList(issue, task),
  }));

  const actions: StageRolloutAction[] = [];
  if (
    taskAndActions.filter(({ actions }) => {
      return actions.includes("ROLLOUT");
    }).length > 1
  ) {
    actions.push("ROLLOUT");
  }
  if (
    taskAndActions.filter(({ task, actions }) => {
      return task.status === Task_Status.FAILED && actions.includes("SKIP");
    }).length > 1
  ) {
    actions.push("SKIP");
  }
  return actions;
};
