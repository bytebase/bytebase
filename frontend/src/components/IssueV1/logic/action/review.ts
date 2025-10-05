import type { ButtonProps } from "naive-ui";
import type { ComputedRef } from "vue";
import { t } from "@/plugins/i18n";
import {
  candidatesOfApprovalStepV1,
  useCurrentUserV1,
  extractUserId,
} from "@/store";
import type { ComposedIssue, ReviewFlow } from "@/types";
import {
  IssueStatus,
  Issue_Approver_Status,
  Issue_ApprovalStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { isUserIncludedInList } from "@/utils";

export type IssueReviewAction = "APPROVE" | "SEND_BACK" | "RE_REQUEST";

export const targetReviewStatusForReviewAction = (
  action: IssueReviewAction
) => {
  switch (action) {
    case "APPROVE":
      return Issue_Approver_Status.APPROVED;
    case "SEND_BACK":
      return Issue_Approver_Status.REJECTED;
    case "RE_REQUEST":
      return Issue_Approver_Status.PENDING;
  }
};

export const issueReviewActionDisplayName = (action: IssueReviewAction) => {
  switch (action) {
    case "APPROVE":
      return t("common.approve");
    case "SEND_BACK":
      return t("custom-approval.issue-review.send-back");
    case "RE_REQUEST":
      return t("custom-approval.issue-review.re-request-review");
  }
};

export const issueReviewActionButtonProps = (
  action: IssueReviewAction
): ButtonProps => {
  switch (action) {
    case "APPROVE":
      return {
        type: "primary",
      };
    case "SEND_BACK":
      return {
        type: "default",
      };
    case "RE_REQUEST":
      return {
        type: "primary",
      };
  }
};

export const allowUserToApplyReviewAction = (
  issue: ComposedIssue,
  flow: ComputedRef<ReviewFlow>,
  action: IssueReviewAction
) => {
  const approvalFlowReady =
    issue.approvalStatus !== Issue_ApprovalStatus.CHECKING;
  const rolloutReady =
    issue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
    issue.approvalStatus === Issue_ApprovalStatus.SKIPPED;

  if (
    issue.status === IssueStatus.CANCELED ||
    issue.status === IssueStatus.DONE
  ) {
    return false;
  }

  if (!approvalFlowReady) return false;
  if (rolloutReady) return false;

  const me = useCurrentUserV1();

  if (action === "APPROVE" || action === "SEND_BACK") {
    const index = flow.value.currentStepIndex;
    const steps = flow.value.template.flow?.steps ?? [];
    const step = steps[index];
    if (!step) return false;
    const candidates = candidatesOfApprovalStepV1(issue, step);
    return isUserIncludedInList(me.value.email, candidates);
  }

  // action === 'RE_REQUEST'
  return me.value.email === extractUserId(issue.creator);
};
