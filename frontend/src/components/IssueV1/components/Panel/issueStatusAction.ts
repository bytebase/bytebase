import type { ButtonProps } from "naive-ui";
import { t } from "@/plugins/i18n";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  isAccessGrantIssue,
  isDatabaseDataExportIssue,
  isGrantRequestIssue,
} from "@/utils";

export type IssueStatusAction = "CLOSE" | "REOPEN";

export const IssueStatusActionToIssueStatusMap: Record<
  IssueStatusAction,
  IssueStatus
> = {
  CLOSE: IssueStatus.CANCELED,
  REOPEN: IssueStatus.OPEN,
};

const PossibleIssueStatusActionMap: Record<IssueStatus, IssueStatusAction[]> = {
  [IssueStatus.OPEN]: ["CLOSE"],
  // Done/resolved issues cannot be reopened
  [IssueStatus.DONE]: [],
  [IssueStatus.CANCELED]: ["REOPEN"],

  // Only to make TypeScript compiler happy
  [IssueStatus.ISSUE_STATUS_UNSPECIFIED]: [],
};

export const getApplicableIssueStatusActionList = (
  issue: Issue
): IssueStatusAction[] => {
  const list = PossibleIssueStatusActionMap[issue.status];
  return list.filter((action) => {
    if (
      isGrantRequestIssue(issue) ||
      isAccessGrantIssue(issue) ||
      isDatabaseDataExportIssue(issue)
    ) {
      // Don't show REOPEN for grantRequest/accessGrant/dataExport issues.
      if (action === "REOPEN") {
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
