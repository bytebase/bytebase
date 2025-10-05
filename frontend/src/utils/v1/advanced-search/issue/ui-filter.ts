import { releaserCandidatesForIssue } from "@/components/IssueV1/logic";
import { candidatesOfApprovalStepV1, userNamePrefix } from "@/store";
import type { ComposedIssue } from "@/types";
import { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import { isUserIncludedInList } from "@/utils";
import type { SearchParams } from "../common";
import { getValueFromSearchParams } from "../common";

export const IssueApprovalStatusList = ["pending", "approved"] as const;
export type IssueApprovalStatus = (typeof IssueApprovalStatusList)[number];
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

  // Only show issues that are pending approval
  if (issue.approvalStatus !== Issue_ApprovalStatus.PENDING) {
    return false;
  }

  const { approvalTemplates, approvers } = issue;
  if (approvalTemplates.length === 0) return false;

  const currentStepIndex = approvers.length;
  const steps = approvalTemplates[0].flow?.steps || [];
  const step = steps[currentStepIndex];
  if (!step) return false;

  const candidates = candidatesOfApprovalStepV1(issue, step);

  // We support "approver:{email}" by now
  // Planning to support "approver:[{email_1}, {email_2}, ...]" and
  // "approver:roles/{role}" in the future
  if (approver.startsWith(userNamePrefix)) {
    return candidates.includes(approver);
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
  if (releaser.startsWith(userNamePrefix)) {
    const candidates = releaserCandidatesForIssue(issue);
    return isUserIncludedInList(releaser, candidates);
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

  const approvalStatus = issue.approvalStatus;
  if (status === "pending") {
    return approvalStatus === Issue_ApprovalStatus.PENDING;
  }
  if (status === "approved") {
    return (
      approvalStatus === Issue_ApprovalStatus.APPROVED ||
      approvalStatus === Issue_ApprovalStatus.SKIPPED
    );
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

export const getIssueApprovalStatusFromSearchParams = (
  params: SearchParams
) => {
  return getValueFromSearchParams(
    params,
    "approval",
    "" /* prefix='' */,
    IssueApprovalStatusList
  ) as IssueApprovalStatus | "";
};

export const buildUIIssueFilterBySearchParams = (params: SearchParams) => {
  const uiIssueFilter: UIIssueFilter = {
    approval: getIssueApprovalStatusFromSearchParams(params),
    approver: getValueFromSearchParams(params, "approver", userNamePrefix),
    releaser: getValueFromSearchParams(params, "releaser", userNamePrefix),
  };

  return uiIssueFilter;
};
