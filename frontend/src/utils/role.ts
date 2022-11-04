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
  | "bb.permission.workspace.manage-member"
  | "bb.permission.workspace.manage-im-integration"
  | "bb.permission.workspace.manage-vcs-provider"
  | "bb.permission.workspace.manage-general"
  // Can execute admininstrive queries such as "SHOW PROCESSLIST"
  | "bb.permission.workspace.admin-sql-editor";

// A map from a particular workspace permission to the respective enablement of a particular workspace role.
// The key is the workspace permission type and the value is the workspace [DEVELOPER, DBA, OWNER] triplet.
export const WORKSPACE_PERMISSION_MATRIX: Map<
  WorkspacePermissionType,
  boolean[]
> = new Map([
  ["bb.permission.workspace.debug", [false, true, true]],
  ["bb.permission.workspace.manage-environment", [false, true, true]],
  ["bb.permission.workspace.manage-instance", [false, true, true]],
  ["bb.permission.workspace.manage-issue", [false, true, true]],
  ["bb.permission.workspace.manage-label", [false, true, true]],
  ["bb.permission.workspace.manage-project", [false, true, true]],
  ["bb.permission.workspace.manage-sql-review-policy", [false, true, true]],
  ["bb.permission.workspace.manage-member", [false, false, true]],
  ["bb.permission.workspace.manage-vcs-provider", [false, false, true]],
  ["bb.permission.workspace.manage-general", [false, false, true]],
  ["bb.permission.workspace.manage-im-integration", [false, true, true]],
  ["bb.permission.workspace.admin-sql-editor", [false, true, true]],
]);

// Returns true if RBAC is not enabled or the particular role has the particular permission.
export function hasWorkspacePermission(
  permission: WorkspacePermissionType,
  role: RoleType
): boolean {
  if (!hasFeature("bb.feature.rbac")) {
    return true;
  }
  switch (role) {
    case "DEVELOPER":
      return WORKSPACE_PERMISSION_MATRIX.get(permission)![0];
    case "DBA":
      return WORKSPACE_PERMISSION_MATRIX.get(permission)![1];
    case "OWNER":
      return WORKSPACE_PERMISSION_MATRIX.get(permission)![2];
  }
}

export type ProjectPermissionType =
  | "bb.permission.project.manage-general"
  | "bb.permission.project.manage-member"
  | "bb.permission.project.manage-sheet"
  | "bb.permission.project.change-database"
  | "bb.permission.project.admin-database"
  | "bb.permission.project.create-database"
  | "bb.permission.project.transfer-database";

// Returns true if RBAC is not enabled or the particular project role has the particular project permission.
export function hasProjectPermission(
  permission: ProjectPermissionType,
  role: ProjectRoleType
): boolean {
  if (!hasFeature("bb.feature.rbac")) {
    return true;
  }

  // A map from a particular project permission to the respective enablement of a particular project role.
  // The key is the project permission type and the value is the project [DEVELOPER, OWNER] triplet.
  const PROJECT_PERMISSION_MATRIX: Map<ProjectPermissionType, boolean[]> =
    new Map([
      ["bb.permission.project.manage-general", [false, true]],
      ["bb.permission.project.manage-member", [false, true]],
      ["bb.permission.project.manage-sheet", [false, true]],
      ["bb.permission.project.change-database", [true, true]],
      ["bb.permission.project.admin-database", [false, true]],
      // If dba-workflow is disabled, then project developer can also create database.
      [
        "bb.permission.project.create-database",
        [!hasFeature("bb.feature.dba-workflow"), true],
      ],
      // If dba-workflow is disabled, then project developer can also transfer database.
      [
        "bb.permission.project.transfer-database",
        [!hasFeature("bb.feature.dba-workflow"), true],
      ],
    ]);

  switch (role) {
    case "DEVELOPER":
      return PROJECT_PERMISSION_MATRIX.get(permission)![0];
    case "OWNER":
      return PROJECT_PERMISSION_MATRIX.get(permission)![1];
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
export function projectRoleName(role: ProjectRoleType): string {
  switch (role) {
    case "OWNER":
      return "Owner";
    case "DEVELOPER":
      return "Developer";
  }
}
