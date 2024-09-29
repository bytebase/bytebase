import type { Permission } from "./iam/permission";

export type EnvironmentQuickActionType =
  | "quickaction.bb.environment.create"
  | "quickaction.bb.environment.reorder";
export type ProjectQuickActionType =
  | "quickaction.bb.project.create"
  | "quickaction.bb.project.database.transfer";
export type InstanceQuickActionType = "quickaction.bb.instance.create";
export type DatabaseQuickActionType = "quickaction.bb.database.create";
export type DatabaseGroupQuickActionType =
  "quickaction.bb.group.database-group.create";
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
export const QuickActionPermissionMap: Map<QuickActionType, Permission[]> =
  new Map([
    ["quickaction.bb.environment.create", ["bb.environments.create"]],
    [
      "quickaction.bb.environment.reorder",
      ["bb.environments.list", "bb.environments.update"],
    ],
    ["quickaction.bb.project.create", ["bb.projects.create"]],
    ["quickaction.bb.instance.create", ["bb.instances.create"]],
    [
      "quickaction.bb.database.create",
      ["bb.instances.list", "bb.issues.create"],
    ],
  ]);
