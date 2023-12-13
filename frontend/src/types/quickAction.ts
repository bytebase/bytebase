export type EnvironmentQuickActionType =
  | "quickaction.bb.environment.create"
  | "quickaction.bb.environment.reorder";
export type ProjectQuickActionType =
  | "quickaction.bb.project.create"
  | "quickaction.bb.project.database.transfer";
export type InstanceQuickActionType = "quickaction.bb.instance.create";
export type DatabaseQuickActionType =
  | "quickaction.bb.database.create" // Used by DBA and Owner
  | "quickaction.bb.database.request" // Used by Developer (not yet)
  | "quickaction.bb.database.schema.update"
  | "quickaction.bb.database.data.update";
export type IssueQuickActionType =
  | "quickaction.bb.issue.grant.request.querier"
  | "quickaction.bb.issue.grant.request.exporter";

export type QuickActionType =
  | EnvironmentQuickActionType
  | ProjectQuickActionType
  | InstanceQuickActionType
  | DatabaseQuickActionType
  | IssueQuickActionType;
