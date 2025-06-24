export type IssueReviewAction = "APPROVE" | "REJECT" | "RE_REQUEST";

export type IssueStatusAction = "CLOSE" | "REOPEN";

export type UnifiedAction = IssueReviewAction | IssueStatusAction;

export interface ActionConfig {
  action: UnifiedAction;
  disabled?: boolean;
  description?: string;
}
