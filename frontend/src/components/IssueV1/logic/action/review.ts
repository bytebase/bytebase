import type { ButtonProps } from "naive-ui";
import { t } from "@/plugins/i18n";
import { Issue_Approver_Status } from "@/types/proto-es/v1/issue_service_pb";

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
