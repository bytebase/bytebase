import type { ButtonProps } from "naive-ui";
import { t } from "@/plugins/i18n";
import { useCurrentUserV1 } from "@/store";
import type { ComposedIssue } from "@/types";
import {
  IssueStatus,
  Issue_Approver_Status,
} from "@/types/proto/v1/issue_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import {
  extractUserResourceName,
  flattenTaskV1List,
  hasProjectPermissionV2,
  isDatabaseChangeRelatedIssue,
  isDatabaseDataExportIssue,
  isGrantRequestIssue,
} from "@/utils";
import { isTaskFinished } from "..";

export type IssueStatusAction = "RESOLVE" | "CLOSE" | "REOPEN";

export const IssueStatusActionToIssueStatusMap: Record<
  IssueStatusAction,
  IssueStatus
> = {
  CLOSE: IssueStatus.CANCELED,
  REOPEN: IssueStatus.OPEN,
  RESOLVE: IssueStatus.DONE,
};

export const PossibleIssueStatusActionMap: Record<
  IssueStatus,
  IssueStatusAction[]
> = {
  [IssueStatus.OPEN]: ["RESOLVE", "CLOSE"],
  [IssueStatus.DONE]: ["REOPEN"],
  [IssueStatus.CANCELED]: ["REOPEN"],

  // Only to make TypeScript compiler happy
  [IssueStatus.ISSUE_STATUS_UNSPECIFIED]: [],
  [IssueStatus.UNRECOGNIZED]: [],
};

export const getApplicableIssueStatusActionList = (
  issue: ComposedIssue,
  _reviewStatus?: Issue_Approver_Status
): IssueStatusAction[] => {
  const list = PossibleIssueStatusActionMap[issue.status];
  return list.filter((action) => {
    if (isGrantRequestIssue(issue) || isDatabaseDataExportIssue(issue)) {
      // Don't show RESOLVE or REOPEN for grantRequest/dataExport issues.
      if (action === "RESOLVE" || action === "REOPEN") {
        return false;
      }
    }
    if (isDatabaseChangeRelatedIssue(issue) && action === "RESOLVE") {
      const tasks = flattenTaskV1List(issue.rolloutEntity);
      // Ths issue cannot be resolved if some tasks are not finished yet.
      if (tasks.some((task) => !isTaskFinished(task))) {
        return false;
      }
    }
    return true;
  });
};

export const issueStatusActionDisplayName = (
  action: IssueStatusAction,
  count: number = 1
) => {
  let actionText = "";
  switch (action) {
    case "RESOLVE":
      actionText = t("issue.batch-transition.resolve");
      break;
    case "CLOSE":
      actionText = t("issue.batch-transition.close");
      break;
    case "REOPEN":
      actionText = t("issue.batch-transition.reopen");
      break;
    default:
      return "";
  }
  if (count <= 1) {
    return actionText;
  }
  return t("issue.batch-transition.action-n-issues", {
    action: actionText,
    n: count,
  });
};

export const issueStatusActionButtonProps = (
  action: IssueStatusAction
): ButtonProps => {
  switch (action) {
    case "RESOLVE":
      return {
        type: "success",
      };
    case "CLOSE":
      return {
        type: "default",
      };
    case "REOPEN":
      return {
        type: "default",
      };
  }
};

export const allowUserToApplyIssueStatusAction = (
  issue: ComposedIssue,
  action: IssueStatusAction
): [boolean /** ok */, string /** reason */] => {
  const user = useCurrentUserV1();
  // User does not have permission to update the issue and is not the creator of the issue.
  if (
    !hasProjectPermissionV2(issue.projectEntity, "bb.issues.update") &&
    extractUserResourceName(issue.creator) !== user.value.email
  ) {
    return [false, t("issue.error.you-don-have-privilege-to-edit-this-issue")];
  }
  if (action === "CLOSE") {
    const tasks = flattenTaskV1List(issue.rolloutEntity);
    // The issue cannot be closed if some tasks are running.
    if (tasks.some((task) => task.status === Task_Status.RUNNING)) {
      return [false, t("issue.error.cannot-close-issue-with-running-tasks")];
    }
  }
  return [true, ""];
};
