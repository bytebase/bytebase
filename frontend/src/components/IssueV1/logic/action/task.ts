import { head } from "lodash-es";
import type { ButtonProps } from "naive-ui";
import { t } from "@/plugins/i18n";
import { useCurrentUserV1 } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import type { ComposedIssue } from "@/types";
import { IssueStatus, Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  isUserIncludedInList,
} from "@/utils";
import { projectOfIssue } from "../utils";

export type TaskRolloutAction =
  | "ROLLOUT" // NOT_STARTED -> PENDING
  | "RETRY" // FAILED -> NOT_STARTED
  | "CANCEL" // ? -> CANCELLED
  | "SKIP" // ? -> SKIPPED
  | "RESTART"; // ? -> NOT_STARTED

// Primary actions will be displayed as big buttons.
export const PrimaryTaskRolloutActionList: TaskRolloutAction[] = [
  "ROLLOUT",
  "CANCEL",
  "RETRY",
  "RESTART",
];

// Secondary actions will be folded in the context menu.
export const SecondaryTaskRolloutActionList: TaskRolloutAction[] = ["SKIP"];

export const CancelableTaskTypeList: Task_Type[] = [
  Task_Type.DATABASE_DATA_UPDATE,
  Task_Type.DATABASE_SCHEMA_UPDATE,
  Task_Type.DATABASE_SCHEMA_UPDATE_SDL,
  Task_Type.DATABASE_SCHEMA_UPDATE_GHOST,
];

export const TaskRolloutActionMap: Record<Task_Status, TaskRolloutAction[]> = {
  [Task_Status.NOT_STARTED]: ["ROLLOUT", "SKIP"],
  [Task_Status.PENDING]: ["CANCEL"], // Only DDL/DML can be cancelled
  [Task_Status.RUNNING]: ["CANCEL"], // Only DDL/DML can be cancelled
  [Task_Status.SKIPPED]: [],
  [Task_Status.DONE]: [],
  [Task_Status.FAILED]: ["RETRY", "SKIP"],
  [Task_Status.CANCELED]: ["RESTART", "SKIP"],

  // Only to make TypeScript compiler happy
  [Task_Status.STATUS_UNSPECIFIED]: [],
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
      return CancelableTaskTypeList.includes(task.type);
    }
    if (action === "SKIP") {
      if (task.status !== Task_Status.FAILED && !allowSkipPendingTask) {
        return false;
      }
    }

    return true;
  });
};

export const taskRolloutActionDisplayName = (
  action: TaskRolloutAction,
  task?: Task
) => {
  switch (action) {
    case "ROLLOUT":
      return task?.type === Task_Type.DATABASE_EXPORT
        ? t("common.export")
        : t("common.rollout");
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

export const taskRolloutActionDialogButtonName = (
  action: TaskRolloutAction,
  tasks: Task[]
) => {
  if (action === "CANCEL") {
    // Avoiding [Cancel] [Cancel] button group scene
    return t("task.cancel-task", tasks.length);
  }
  return taskRolloutActionDisplayName(action, head(tasks));
};

export const taskRolloutActionButtonProps = (
  action: TaskRolloutAction
): ButtonProps => {
  switch (action) {
    case "ROLLOUT":
      return {
        type: "primary",
      };
    case "RESTART":
      return {
        type: "primary",
      };
    case "RETRY":
      return {
        type: "primary",
      };
  }
  return {
    type: "default",
  };
};

export const allowUserToApplyTaskRolloutAction = (
  issue: ComposedIssue,
  action: TaskRolloutAction,
  releaserCandidates: string[]
) => {
  const me = useCurrentUserV1();
  // For data export issues, only the creator can take actions.
  if (issue.type === Issue_Type.DATABASE_EXPORT) {
    return issue.creator === `${userNamePrefix}${me.value.email}`;
  }

  // Only for users with permission to create task runs.
  if (
    hasWorkspacePermissionV2("bb.taskRuns.create") ||
    hasProjectPermissionV2(projectOfIssue(issue), "bb.taskRuns.create")
  ) {
    return true;
  }

  return isUserIncludedInList(me.value.email, releaserCandidates);
};
