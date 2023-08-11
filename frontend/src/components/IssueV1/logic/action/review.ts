import { ButtonProps } from "naive-ui";
import { t } from "@/plugins/i18n";
import { candidatesOfApprovalStepV1 } from "@/store";
import { ComposedIssue } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import {
  IssueStatus,
  Issue_Approver_Status,
} from "@/types/proto/v1/issue_service";
import { extractUserResourceName } from "@/utils";
import { ReviewContext } from "../context";

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
  context: ReviewContext,
  user: User,
  action: IssueReviewAction
) => {
  const { ready, done, flow } = context;

  if (
    issue.status === IssueStatus.CANCELED ||
    issue.status === IssueStatus.DONE
  ) {
    return false;
  }

  if (!ready.value) return false;
  if (done.value) return false;

  if (action === "APPROVE" || action === "SEND_BACK") {
    const index = flow.value.currentStepIndex;
    const steps = flow.value.template.flow?.steps ?? [];
    const step = steps[index];
    if (!step) return false;
    const candidates = candidatesOfApprovalStepV1(issue, step);
    return candidates.includes(user.name);
  }

  // action === 'RE_REQUEST'
  return user.email === extractUserResourceName(issue.creator);
};
