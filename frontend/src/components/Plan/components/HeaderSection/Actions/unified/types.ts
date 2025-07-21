export type IssueReviewAction =
  | "ISSUE_REVIEW_APPROVE"
  | "ISSUE_REVIEW_REJECT"
  | "ISSUE_REVIEW_RE_REQUEST";

export type IssueStatusAction = "ISSUE_STATUS_CLOSE" | "ISSUE_STATUS_REOPEN";

export type IssueAction =
  | IssueReviewAction
  | IssueStatusAction
  | "ISSUE_CREATE";

export type PlanAction = "PLAN_CLOSE" | "PLAN_REOPEN";

// All unified actions
export type UnifiedAction = IssueAction | PlanAction;

export interface ActionConfig {
  action: UnifiedAction;
  disabled?: boolean;
  description?: string;
}
