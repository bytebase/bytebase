export type IssueReviewAction = "APPROVE" | "REJECT" | "RE_REQUEST";

export type IssueStatusAction = "CLOSE" | "REOPEN";

export type IssueCreationAction = "CREATE_ISSUE";

export type UnifiedAction =
  | IssueReviewAction
  | IssueStatusAction
  | IssueCreationAction;

export interface ActionConfig {
  action: UnifiedAction;
  disabled?: boolean;
  description?: string;
}
