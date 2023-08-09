import { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Stage, Task } from "@/types/proto/v1/rollout_service";
import { getApplicableTaskRolloutActionList, TaskRolloutAction } from "./task";

export type StageRolloutAction = Extract<
  TaskRolloutAction,
  "ROLLOUT" | "CANCEL" | "RETRY" | "SKIP" | "RESTART"
>;
export const StageRolloutActionList: StageRolloutAction[] = [
  "ROLLOUT",
  "CANCEL",
  "RETRY",
  "SKIP",
  "RESTART",
];

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
    CANCEL: [],
    RETRY: [],
    SKIP: [],
    RESTART: [],
  };
  stage.tasks.forEach((task) => {
    const actions = getApplicableTaskRolloutActionList(
      issue,
      task,
      allowSkipPendingTasks
    );
    StageRolloutActionList.forEach((action) => {
      if (actions.includes(action)) {
        applicableActionsMap[action].push(task);
      }
    });
  });

  const actions: { action: StageRolloutAction; tasks: Task[] }[] = [];
  StageRolloutActionList.forEach((action) => {
    const tasks = applicableActionsMap[action];
    if (tasks.length > 1) {
      actions.push({
        action,
        tasks,
      });
    }
  });
  return actions;
};
