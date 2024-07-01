import type { ButtonProps } from "naive-ui";
import { t } from "@/plugins/i18n";
import type { ComposedIssue } from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import {
  IssueStatus,
  Issue_Approver_Status,
} from "@/types/proto/v1/issue_service";
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
  reviewStatus?: Issue_Approver_Status
): IssueStatusAction[] => {
  const list = PossibleIssueStatusActionMap[issue.status];
  return list.filter((action) => {
    if (isGrantRequestIssue(issue) || isDatabaseDataExportIssue(issue)) {
      // Don't show RESOLVE or REOPEN for request granting issues.
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
    // For SQL review issues, only allow RESOLVE when the review is approved.
    if (!issue.rollout) {
      if (
        action === "RESOLVE" &&
        reviewStatus !== Issue_Approver_Status.APPROVED
      ) {
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
    case "CLOSE":
      return t("issue.status-transition.dropdown.close");
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
  user: User,
  action: IssueStatusAction
) => {
  // Allowed if the user has issues.update permission in the project
  if (hasProjectPermissionV2(issue.projectEntity, user, "bb.issues.update")) {
    return true;
  }

  // The creator is allowed.
  if (extractUserResourceName(issue.creator) === user.email) {
    return true;
  }

  return false;
};
