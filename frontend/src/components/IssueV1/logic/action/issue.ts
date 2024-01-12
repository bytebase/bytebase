import { ButtonProps } from "naive-ui";
import { t } from "@/plugins/i18n";
import { ComposedIssue } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  extractUserResourceName,
  flattenTaskV1List,
  hasProjectPermissionV2,
  isDatabaseRelatedIssue,
  isGrantRequestIssue,
} from "@/utils";
import { isTaskFinished } from "..";

export type IssueStatusAction = "RESOLVE" | "CANCEL" | "REOPEN";

export const IssueStatusActionToIssueStatusMap: Record<
  IssueStatusAction,
  IssueStatus
> = {
  CANCEL: IssueStatus.CANCELED,
  REOPEN: IssueStatus.OPEN,
  RESOLVE: IssueStatus.DONE,
};

export const PossibleIssueStatusActionMap: Record<
  IssueStatus,
  IssueStatusAction[]
> = {
  [IssueStatus.OPEN]: ["RESOLVE", "CANCEL"],
  [IssueStatus.DONE]: ["REOPEN"],
  [IssueStatus.CANCELED]: ["REOPEN"],

  // Only to make TypeScript compiler happy
  [IssueStatus.ISSUE_STATUS_UNSPECIFIED]: [],
  [IssueStatus.UNRECOGNIZED]: [],
};

export const getApplicableIssueStatusActionList = (
  issue: ComposedIssue
): IssueStatusAction[] => {
  const list = PossibleIssueStatusActionMap[issue.status];
  return list.filter((action) => {
    if (isGrantRequestIssue(issue)) {
      // Don't show RESOLVE or REOPEN for request granting issues.
      if (action === "RESOLVE" || action === "REOPEN") {
        return false;
      }
    }
    if (isDatabaseRelatedIssue(issue) && action === "RESOLVE") {
      const tasks = flattenTaskV1List(issue.rolloutEntity);
      // Ths issue cannot be resolved if some tasks are not finished yet.
      if (tasks.some((task) => !isTaskFinished(task))) {
        return false;
      }
    }
    return true;
  });
};

export const issueStatusActionDisplayName = (action: IssueStatusAction) => {
  switch (action) {
    case "RESOLVE":
      return t("issue.status-transition.dropdown.resolve");
    case "CANCEL":
      return t("issue.status-transition.dropdown.cancel");
    case "REOPEN":
      return t("issue.status-transition.dropdown.reopen");
  }
};

export const issueStatusActionButtonProps = (
  action: IssueStatusAction
): ButtonProps => {
  switch (action) {
    case "RESOLVE":
      return {
        type: "success",
      };
    case "CANCEL":
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
  user: User,
  action: IssueStatusAction
) => {
  // Allowed if the user has issues.update permission in the project
  if (hasProjectPermissionV2(issue.projectEntity, user, "bb.issues.update")) {
    return true;
  }

  // The creator and the assignee are allowed.
  if (extractUserResourceName(issue.creator) === user.email) {
    return true;
  }
  if (extractUserResourceName(issue.assignee) === user.email) {
    return true;
  }

  return false;
};
