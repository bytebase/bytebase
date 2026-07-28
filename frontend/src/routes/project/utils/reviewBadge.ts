import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";

export interface ReviewBadgeInput {
  hasIssue: boolean;
  issueStatus: IssueStatus | undefined;
  hasRollout: boolean;
  approvalStatus: ApprovalStatus | undefined;
}

export interface ReviewBadge {
  labelKey: string;
  variant: "default" | "secondary" | "destructive" | "warning" | "success";
}

export type PlanDraftState = "draft" | "incomplete" | undefined;

export function getPlanDraftState(input: {
  approvalStatus: ApprovalStatus;
  hasRollout: boolean;
  isGitOpsPlan: boolean;
  issueName: string;
}): PlanDraftState {
  if (
    input.isGitOpsPlan ||
    input.hasRollout ||
    input.approvalStatus !== ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED
  ) {
    return undefined;
  }
  return input.issueName ? "draft" : "incomplete";
}

export function getReviewBadge(
  input: ReviewBadgeInput
): ReviewBadge | undefined {
  const { hasIssue, issueStatus, hasRollout, approvalStatus } = input;
  if (!hasIssue) return undefined;
  if (issueStatus === IssueStatus.CANCELED) {
    return { labelKey: "common.closed", variant: "default" };
  }
  const completed = hasRollout || issueStatus === IssueStatus.DONE;
  if (completed && approvalStatus === ApprovalStatus.PENDING) {
    return { labelKey: "common.bypassed", variant: "default" };
  }
  switch (approvalStatus) {
    case ApprovalStatus.APPROVED:
      return { labelKey: "issue.table.approved", variant: "success" };
    case ApprovalStatus.SKIPPED:
      return { labelKey: "common.skipped", variant: "default" };
    case ApprovalStatus.REJECTED:
      return { labelKey: "common.rejected", variant: "warning" };
    case ApprovalStatus.PENDING:
      return { labelKey: "common.under-review", variant: "secondary" };
    default:
      // CHECKING and APPROVAL_STATUS_UNSPECIFIED render no badge. This
      // aligns Plan Detail (current behavior) and Plan List (which
      // previously showed a transient "Checking" pill during plan checks).
      return undefined;
  }
}
