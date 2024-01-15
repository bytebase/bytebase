import { WorkspacePermission, ProjectPermission } from "./iam/permission";

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
export type DatabaseGroupQuickActionType =
  | "quickaction.bb.group.database-group.create"
  | "quickaction.bb.group.table-group.create";
export type IssueQuickActionType =
  | "quickaction.bb.issue.grant.request.querier"
  | "quickaction.bb.issue.grant.request.exporter";

export type QuickActionType =
  | EnvironmentQuickActionType
  | ProjectQuickActionType
  | InstanceQuickActionType
  | DatabaseQuickActionType
  | DatabaseGroupQuickActionType
  | IssueQuickActionType;

// Permission check for workspace level quick actions.
export const QuickActionPermissionMap: Map<
  QuickActionType,
  WorkspacePermission[]
> = new Map([
  ["quickaction.bb.environment.create", ["bb.environments.create"]],
  [
    "quickaction.bb.environment.reorder",
    ["bb.environments.list", "bb.environments.update"],
  ],
  ["quickaction.bb.project.create", ["bb.projects.create"]],
  ["quickaction.bb.instance.create", ["bb.instances.create"]],
  ["quickaction.bb.database.create", []],
  ["quickaction.bb.database.request", []],
  ["quickaction.bb.database.schema.update", []],
  ["quickaction.bb.database.data.update", []],
]);

// Permission check for project level quick actions.
export const QuickActionProjectPermissionMap: Map<
  QuickActionType,
  ProjectPermission[]
> = new Map([
  ["quickaction.bb.project.database.transfer", ["bb.projects.update"]],
  ["quickaction.bb.database.create", ["bb.issues.create"]],
  ["quickaction.bb.database.request", []],
  ["quickaction.bb.database.schema.update", ["bb.issues.create"]],
  ["quickaction.bb.database.data.update", ["bb.issues.create"]],
  ["quickaction.bb.group.database-group.create", ["bb.projects.update"]],
  ["quickaction.bb.group.table-group.create", ["bb.projects.update"]],
  ["quickaction.bb.issue.grant.request.querier", ["bb.issues.create"]],
  ["quickaction.bb.issue.grant.request.exporter", ["bb.issues.create"]],
]);
