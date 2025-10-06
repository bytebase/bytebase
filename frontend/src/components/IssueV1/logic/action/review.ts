import { t } from "@/plugins/i18n";

export type IssueReviewAction = "APPROVE" | "SEND_BACK" | "RE_REQUEST";

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
