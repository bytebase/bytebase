import { ProjectRoleType, RoleType } from "../types";
import { hasFeature } from "@/store";

export type WorkspacePermissionType =
  | "bb.permission.workspace.debug"
  | "bb.permission.workspace.manage-environment"
  | "bb.permission.workspace.manage-instance"
  // Change issue assignee, approve issue, view all issues
  | "bb.permission.workspace.manage-issue"
  | "bb.permission.workspace.manage-label"
  | "bb.permission.workspace.manage-project"
  | "bb.permission.workspace.manage-sql-review-policy"
  | "bb.permission.workspace.manage-user"
  | "bb.permission.workspace.manage-vcs-provider"
  | "bb.permission.workspace.manage-workspace"
  // Can execute admininstrive queries such as "SHOW PROCESSLIST"
  | "bb.permission.workspace.admin-sql-editor";

// A map from a particular workspace permission to the respective enablement of a particular workspace role.
// The key is the workspace permission type and the value is the [DEVELOPER, DBA, OWNER] triplet.
export const WORKSPACE_FEATURE_MATRIX: Map<WorkspacePermissionType, boolean[]> =
  new Map([
    ["bb.permission.workspace.debug", [false, true, true]],
    ["bb.permission.workspace.manage-environment", [false, true, true]],
    ["bb.permission.workspace.manage-instance", [false, true, true]],
    ["bb.permission.workspace.manage-issue", [false, true, true]],
    ["bb.permission.workspace.manage-label", [false, true, true]],
    ["bb.permission.workspace.manage-project", [false, true, true]],
    ["bb.permission.workspace.manage-sql-review-policy", [false, true, true]],
    ["bb.permission.workspace.manage-user", [false, false, true]],
    ["bb.permission.workspace.manage-vcs-provider", [false, false, true]],
    ["bb.permission.workspace.manage-workspace", [false, false, true]],
    ["bb.permission.workspace.admin-sql-editor", [false, true, true]],
  ]);

// Returns true if the particular role has the particular permission.
export function hasWorkspacePermission(
  permission: WorkspacePermissionType,
  role: RoleType
): boolean {
  switch (role) {
    case "DEVELOPER":
      return WORKSPACE_FEATURE_MATRIX.get(permission)![0];
    case "DBA":
      return WORKSPACE_FEATURE_MATRIX.get(permission)![1];
    case "OWNER":
      return WORKSPACE_FEATURE_MATRIX.get(permission)![2];
  }
}

// Returns true if admin feature is NOT supported or the principal is OWNER
export function isOwner(role: RoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "OWNER";
}

// Returns true if admin feature is NOT supported or the principal is DBA
export function isDBA(role: RoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "DBA";
}

// Returns true if admin feature is NOT supported or the principal is DEVELOPER
export function isDeveloper(role: RoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "DEVELOPER";
}

export function roleName(role: RoleType): string {
  switch (role) {
    case "OWNER":
      return "Owner";
    case "DBA":
      return "DBA";
    case "DEVELOPER":
      return "Developer";
  }
}

// Project Role
export function isProjectOwner(role: ProjectRoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "OWNER";
}

export function isProjectDeveloper(role: ProjectRoleType): boolean {
  return !hasFeature("bb.feature.rbac") || role == "DEVELOPER";
}

export function projectRoleName(role: ProjectRoleType): string {
  switch (role) {
    case "OWNER":
      return "Owner";
    case "DEVELOPER":
      return "Developer";
  }
}
