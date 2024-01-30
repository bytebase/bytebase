import { kebabCase } from "lodash-es";
import { computed, unref } from "vue";
import { t } from "@/plugins/i18n";
import { hasFeature, useCurrentUserV1, useRoleStore } from "@/store";
import { MaybeRef, RoleType, PresetRoleType } from "@/types";
import { UserRole } from "@/types/proto/v1/auth_service";

export type WorkspacePermissionType =
  | "bb.permission.workspace.debug"
  | "bb.permission.workspace.manage-instance"
  // Visible to and manage databases even if not in the project the database
  // belongs to, and unassigned databases
  | "bb.permission.workspace.manage-database"
  // Change issue assignee, change issue status, view all issues
  | "bb.permission.workspace.manage-issue"
  | "bb.permission.workspace.manage-label"
  | "bb.permission.workspace.manage-project"
  | "bb.permission.workspace.manage-sql-review-policy"
  | "bb.permission.workspace.manage-member"
  | "bb.permission.workspace.manage-im-integration"
  | "bb.permission.workspace.manage-sso"
  | "bb.permission.workspace.manage-vcs-provider"
  | "bb.permission.workspace.manage-general"
  | "bb.permission.workspace.manage-sensitive-data"
  | "bb.permission.workspace.manage-access-control"
  | "bb.permission.workspace.manage-custom-approval"
  | "bb.permission.workspace.manage-slow-query"
  | "bb.permission.workspace.manage-subscription"
  // Can execute admininstrive queries such as "SHOW PROCESSLIST"
  | "bb.permission.workspace.admin-sql-editor"
  // Can view sensitive information such as audit logs and debug logs
  | "bb.permission.workspace.audit-log"
  | "bb.permission.workspace.debug-log"
  | "bb.permission.workspace.manage-mail-delivery"
  | "bb.permission.workspace.manage-database-secrets"
  | "bb.permission.workspace.manage-announcement";

// A map from a particular workspace permission to the respective enablement of a particular workspace role.
// The key is the workspace permission type and the value is the workspace [DEVELOPER, DBA, OWNER] triplet.
export const WORKSPACE_PERMISSION_MATRIX: Map<
  WorkspacePermissionType,
  boolean[]
> = new Map([
  ["bb.permission.workspace.debug", [false, true, true]],
  ["bb.permission.workspace.manage-instance", [false, true, true]],
  ["bb.permission.workspace.manage-database", [false, true, true]],
  ["bb.permission.workspace.manage-issue", [false, true, true]],
  ["bb.permission.workspace.manage-label", [false, true, true]],
  ["bb.permission.workspace.manage-project", [false, true, true]],
  ["bb.permission.workspace.manage-sql-review-policy", [false, true, true]],
  ["bb.permission.workspace.manage-member", [false, false, true]],
  ["bb.permission.workspace.manage-vcs-provider", [false, false, true]],
  ["bb.permission.workspace.manage-general", [false, false, true]],
  ["bb.permission.workspace.manage-im-integration", [false, false, true]],
  ["bb.permission.workspace.manage-sso", [false, false, true]],
  ["bb.permission.workspace.manage-sensitive-data", [false, true, true]],
  ["bb.permission.workspace.manage-access-control", [false, true, true]],
  ["bb.permission.workspace.manage-custom-approval", [false, true, true]],
  ["bb.permission.workspace.manage-slow-query", [false, true, true]],
  ["bb.permission.workspace.manage-subscription", [false, true, true]],
  ["bb.permission.workspace.admin-sql-editor", [false, true, true]],
  ["bb.permission.workspace.audit-log", [false, true, true]],
  ["bb.permission.workspace.debug-log", [false, true, true]],
  ["bb.permission.workspace.manage-mail-delivery", [false, false, true]],
  ["bb.permission.workspace.manage-database-secrets", [false, true, true]],
  ["bb.permission.workspace.manage-announcement", [false, true, true]],
]);

// Returns true if RBAC is not enabled or the particular role has the particular permission.
export function hasWorkspacePermissionV1(
  permission: WorkspacePermissionType,
  role: UserRole
): boolean {
  if (!hasFeature("bb.feature.rbac")) {
    return true;
  }
  switch (role) {
    case UserRole.DEVELOPER:
      return WORKSPACE_PERMISSION_MATRIX.get(permission)![0];
    case UserRole.DBA:
      return WORKSPACE_PERMISSION_MATRIX.get(permission)![1];
    case UserRole.OWNER:
      return WORKSPACE_PERMISSION_MATRIX.get(permission)![2];
  }
  return false;
}

export const useWorkspacePermissionV1 = (
  permission: MaybeRef<WorkspacePermissionType>
) => {
  const user = useCurrentUserV1();
  return computed(() => {
    return hasWorkspacePermissionV1(unref(permission), user.value.userRole);
  });
};

export type ProjectPermissionType =
  | "bb.permission.project.manage-general"
  | "bb.permission.project.manage-member"
  | "bb.permission.project.manage-sheet"
  | "bb.permission.project.change-database"
  | "bb.permission.project.admin-database"
  | "bb.permission.project.create-database"
  | "bb.permission.project.transfer-database"
  | "bb.permission.project.manage-database-secrets";

// Returns true if RBAC is not enabled or the particular project role has the particular project permission.
export function hasProjectPermission(
  permission: ProjectPermissionType,
  role: string
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
      ["bb.permission.project.manage-database-secrets", [false, true]],
    ]);

  switch (role) {
    case "projectDeveloper":
      return PROJECT_PERMISSION_MATRIX.get(permission)![0];
    case "projectOwner":
      return PROJECT_PERMISSION_MATRIX.get(permission)![1];
  }
  return false;
}

// Returns true if admin feature is NOT supported or the principal is OWNER
export function isOwner(role: UserRole): boolean {
  return !hasFeature("bb.feature.rbac") || role === UserRole.OWNER;
}

// Returns true if admin feature is NOT supported or the principal is DBA
export function isDBA(role: UserRole): boolean {
  return !hasFeature("bb.feature.rbac") || role === UserRole.DBA;
}

// Returns true if admin feature is NOT supported or the principal is DEVELOPER
export function isDeveloper(role: UserRole): boolean {
  return !hasFeature("bb.feature.rbac") || role === UserRole.DEVELOPER;
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
export function projectRoleName(role: string): string {
  switch (role) {
    case "OWNER":
      return "Owner";
    case "DEVELOPER":
      return "Developer";
    case "QUERIER":
      return "Querier";
    case "EXPORTER":
      return "Exporter";
    case "VIEWER":
      return "Viewer";
  }
  return role;
}

export const extractRoleResourceName = (resourceId: string): string => {
  const pattern = /(?:^|\/)roles\/([^/]+)(?:$|\/)/;
  const matches = resourceId.match(pattern);
  return matches?.[1] ?? "";
};

export const displayRoleTitle = (role: string): string => {
  if (Object.values(PresetRoleType).includes(role)) {
    return t(`role.${kebabCase(extractRoleResourceName(role))}.self`);
  }
  // Use role.title if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.title || extractRoleResourceName(role);
};

export const displayRoleDescription = (role: string): string => {
  if (Object.values(PresetRoleType).includes(role)) {
    return t(`role.${kebabCase(extractRoleResourceName(role))}.description`);
  }
  // Use role.description if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.description || extractRoleResourceName(role);
};

export const hasSettingPagePermission = (
  routeName: string,
  role: UserRole
): boolean => {
  switch (routeName) {
    case "setting.workspace.member":
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-member",
        role
      );
    case "setting.workspace.sensitive-data":
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-sensitive-data",
        role
      );
    case "setting.workspace.access-control":
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-access-control",
        role
      );
    case "setting.workspace.sso":
    case "setting.workspace.sso.create":
    case "setting.workspace.sso.detail":
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-sso",
        role
      );
    case "setting.workspace.gitops":
    case "setting.workspace.gitops.create":
    case "setting.workspace.gitops.detail":
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-vcs-provider",
        role
      );
    case "setting.workspace.debug-log":
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.debug-log",
        role
      );
    case "setting.workspace.audit-log":
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.audit-log",
        role
      );
    case "setting.workspace.mail-delivery":
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-mail-delivery",
        role
      );
  }
  return true;
};
