export type IssueReviewAction = "ISSUE_REVIEW";

export type IssueStatusAction =
  | "ISSUE_STATUS_CLOSE"
  | "ISSUE_STATUS_REOPEN"
  | "ISSUE_STATUS_RESOLVE";

export type IssueAction =
  | IssueReviewAction
  | IssueStatusAction
  | "ISSUE_CREATE";

export type PlanAction = "PLAN_CLOSE" | "PLAN_REOPEN";

export type RolloutAction =
  | "ROLLOUT_CREATE"
  | "ROLLOUT_START"
  | "ROLLOUT_CANCEL";

export type ExportAction = "EXPORT_DOWNLOAD";

// All unified actions
export type UnifiedAction =
  | IssueAction
  | PlanAction
  | RolloutAction
  | ExportAction;
