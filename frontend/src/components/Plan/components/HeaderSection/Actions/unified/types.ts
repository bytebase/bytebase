export type IssueReviewAction = "APPROVE" | "REJECT" | "RE_REQUEST";

export type IssueStatusAction = "CLOSE" | "REOPEN";

export type RolloutAction = "CREATE_ROLLOUT";

export type IssueCreationAction = "CREATE_ISSUE";

export type UnifiedAction = IssueReviewAction | IssueStatusAction | RolloutAction | IssueCreationAction;

export interface ActionConfig {
  action: UnifiedAction;
  disabled?: boolean;
  description?: string;
}
