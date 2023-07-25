import { t } from "@/plugins/i18n";
import { ComposedIssue } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Task, Task_Status, Task_Type } from "@/types/proto/v1/rollout_service";
import { extractUserResourceName } from "@/utils";
import {
  allowUserToBeAssignee,
  getCurrentRolloutPolicyForTask,
} from "../assignee";

export type TaskRolloutAction =
  | "ROLLOUT" // PENDING_APPROVAL -> PENDING
  | "RETRY" // FAILED -> PENDING_APPROVAL
  | "CANCEL" // ? -> CANCELLED
  | "SKIP" // ? -> SKIPPED
  | "RESTART"; // ? -> PENDING_APPROVAL

// Primary actions will be displayed as big buttons.
export const PrimaryTaskRolloutActionList: TaskRolloutAction[] = [
  "ROLLOUT",
  "CANCEL",
  "RETRY",
  "RESTART",
];

// Secondary actions will be folded in the context menu.
export const SecondaryTaskRolloutActionList: TaskRolloutAction[] = ["SKIP"];

export const TaskRolloutActionMap: Record<Task_Status, TaskRolloutAction[]> = {
  [Task_Status.PENDING_APPROVAL]: ["ROLLOUT", "SKIP"],
  [Task_Status.PENDING]: ["CANCEL"],
  [Task_Status.RUNNING]: ["CANCEL"], // Only gh-ost sync task can be cancelled
  [Task_Status.SKIPPED]: [],
  [Task_Status.DONE]: [],
  [Task_Status.FAILED]: ["RETRY", "SKIP"],
  [Task_Status.CANCELED]: ["RESTART"],

  // Only to make TypeScript compiler happy
  [Task_Status.STATUS_UNSPECIFIED]: [],
  [Task_Status.UNRECOGNIZED]: [],
};

export const getApplicableTaskRolloutActionList = (
  issue: ComposedIssue,
  task: Task,
  allowSkipPendingTask = false // If set to true, only FAILED tasks can be skipped
): TaskRolloutAction[] => {
  if (issue.status !== IssueStatus.OPEN) {
    return [];
  }
  const list = TaskRolloutActionMap[task.status];
  return list.filter((action) => {
    if (action === "CANCEL") {
      // Now, only gh-ost sync task is cancelable
      return task.type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC;
    }
    if (action === "RETRY") {
      // RETRYing gh-ost cut-over task is not allowed (yet).
      if (task.type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER) {
        return false;
      }
    }
    if (action === "SKIP") {
      if (task.status !== Task_Status.FAILED && !allowSkipPendingTask) {
        return false;
      }
    }

    return true;
  });
};

export const taskRolloutActionDisplayName = (action: TaskRolloutAction) => {
  switch (action) {
    case "ROLLOUT":
      return t("common.rollout");
    case "CANCEL":
      return t("common.cancel");
    case "RETRY":
      return t("common.retry");
    case "RESTART":
      return t("common.restart");
    case "SKIP":
      return t("common.skip");
  }
};

export const allowUserToApplyTaskRolloutAction = async (
  issue: ComposedIssue,
  task: Task,
  user: User,
  action: TaskRolloutAction
) => {
  if (extractUserResourceName(issue.assignee) === user.email) {
    return true;
  }

  const project = issue.projectEntity;
  const rolloutPolicy = await getCurrentRolloutPolicyForTask(issue, task);
  if (
    allowUserToBeAssignee(
      user,
      project,
      project.iamPolicy,
      rolloutPolicy.policy,
      rolloutPolicy.assigneeGroup
    )
  ) {
    return true;
  }

  return false;
};
