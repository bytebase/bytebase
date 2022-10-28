import { ProjectRoleType, RoleType } from "../types";
import { hasFeature } from "@/store";

export type WorkspacePermissionType = "bb.permission.workspace.manage-user";

// A map from a particular workspace permission to the respective enablement of a particular workspace role.
// The key is the workspace permission type and the value is the [DEVELOPER, DBA, OWNER] triplet.
export const WORKSPACE_FEATURE_MATRIX: Map<WorkspacePermissionType, boolean[]> =
  new Map([["bb.permission.workspace.manage-user", [false, false, true]]]);

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

export function isDBAOrOwner(role: RoleType): boolean {
  return isDBA(role) || isOwner(role);
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
