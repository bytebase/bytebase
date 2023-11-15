import {
  extractReviewContext,
  releaserCandidatesForIssue,
  useWrappedReviewStepsV1,
} from "@/components/IssueV1/logic";
import { useUserStore } from "@/store";
import { ComposedIssue } from "@/types";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import { extractUserResourceName } from "../../user";
import { SearchParams, getValueFromSearchParams } from "./common";

export const IssueApprovalStatusList = ["pending", "approved"] as const;
export type IssueApprovalStatus = typeof IssueApprovalStatusList[number];
export const isValidIssueApprovalStatus = (
  s: string
): s is IssueApprovalStatus => {
  return IssueApprovalStatusList.includes(s as IssueApprovalStatus);
};

// Use snake_case to keep consistent with the advanced search query string
export interface UIIssueFilter {
  approver?: string;
  approval?: IssueApprovalStatus | "";
  releaser?: string;
}

export const filterIssueByApprover = (
  issue: ComposedIssue,
  approver: string | undefined
) => {
  if (!approver) return true;

  const reviewContext = extractReviewContext(issue);
  const steps = useWrappedReviewStepsV1(issue, reviewContext);

  const currentStep = steps.value?.find((step) => step.status === "CURRENT");
  if (!currentStep) return false;
  // We support "approver:{email}" by now
  // Planning to support "approver:[{email_1}, {email_2}, ...]" and
  // "approver:roles/{role}" in the future
  if (approver.startsWith("users/")) {
    return (
      currentStep.candidates.findIndex(
        (user) => `users/${user.email}` === approver
      ) >= 0
    );
  }

  console.error(
    "[filterIssueByApprover] should never reach this line",
    approver
  );
  return false;
};

export const filterIssueByReleaser = (
  issue: ComposedIssue,
  releaser: string | undefined
) => {
  if (!releaser) return true;

  // We support "release:{email}" by now
  // Planning to support "release:[{email_1}, {email_2}, ...]" and
  // "release:roles/{role}" in the future
  if (releaser.startsWith("users/")) {
    const user = useUserStore().getUserByEmail(
      extractUserResourceName(releaser)
    );
    if (!user) return false;

    if (
      hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-issue",
        user.userRole
      )
    ) {
      // Super users are always allowed to rollout issues.
      return true;
    }

    const candidates = releaserCandidatesForIssue(issue);
    // Otherwise any releasers can rollout the issue.
    if (candidates.findIndex((c) => c.name === user.name) >= 0) {
      return true;
    }

    return false;
  }

  console.error(
    "[filterIssueByReleaser] should never reach this line",
    releaser
  );
  return false;
};

export const filterIssueByApprovalStatus = (
  issue: ComposedIssue,
  status: IssueApprovalStatus | "" | undefined
) => {
  if (!status) return true;

  const reviewContext = extractReviewContext(issue);
  if (status === "pending") {
    return reviewContext.status.value === Issue_Approver_Status.PENDING;
  }
  if (status === "approved") {
    return reviewContext.status.value === Issue_Approver_Status.APPROVED;
  }

  console.error(
    "[filterIssueByApprovalStatus] should never reach this line",
    status
  );
  return false;
};

export const applyUIIssueFilter = (
  list: ComposedIssue[],
  filter: UIIssueFilter | undefined
) => {
  if (!filter) return list;
  return list
    .filter((issue) => filterIssueByApprover(issue, filter.approver))
    .filter((issue) => filterIssueByApprovalStatus(issue, filter.approval))
    .filter((issue) => filterIssueByReleaser(issue, filter.releaser));
};

export const buildUIIssueFilterBySearchParams = (params: SearchParams) => {
  const uiIssueFilter: UIIssueFilter = {
    approval: getValueFromSearchParams(
      params,
      "approval",
      "" /* prefix='' */,
      IssueApprovalStatusList
    ) as IssueApprovalStatus | "",
    approver: getValueFromSearchParams(params, "approver", "users/"),
    releaser: getValueFromSearchParams(params, "releaser", "users/"),
  };

  return uiIssueFilter;
};
