export type EnvironmentQuickActionType =
  | "quickaction.bb.environment.create"
  | "quickaction.bb.environment.reorder";
export type ProjectQuickActionType =
  | "quickaction.bb.project.create"
  | "quickaction.bb.project.database.transfer"
  | "quickaction.bb.project.database.transfer-out";
export type InstanceQuickActionType = "quickaction.bb.instance.create";
export type UserQuickActionType = "quickaction.bb.user.manage";
export type DatabaseQuickActionType =
  | "quickaction.bb.database.create" // Used by DBA and Owner
  | "quickaction.bb.database.request" // Used by Developer (not yet)
  | "quickaction.bb.database.schema.update"
  | "quickaction.bb.database.data.update"
  // Schema designer quick action. (Maybe will be removed after changelist is implemented)
  | "quickaction.bb.database.branching"
  | "quickaction.bb.database.troubleshoot";
export type IssueQuickActionType =
  | "quickaction.bb.issue.grant.request.querier"
  | "quickaction.bb.issue.grant.request.exporter";
type SubscriptionQuickActionType =
  "quickaction.bb.subscription.license-assignment";

export type QuickActionType =
  | EnvironmentQuickActionType
  | ProjectQuickActionType
  | InstanceQuickActionType
  | UserQuickActionType
  | DatabaseQuickActionType
  | IssueQuickActionType
  | SubscriptionQuickActionType;
