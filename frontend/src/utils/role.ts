import i18n from "@/react/i18n";
import type { Permission } from "@/types/iam";
import { PresetRoleType } from "@/types/iam";
import { appStoreUtilBridge } from "@/utils/app-store-bridge";

export const checkRoleContainsAnyPermission = (
  roleName: string,
  ...permissions: Permission[]
): boolean => {
  const role = appStoreUtilBridge()?.getRoleByName(roleName);
  if (!role) {
    return false;
  }
  return permissions.some((p) => role.permissions.includes(p));
};

export const extractRoleResourceName = (resourceId: string): string => {
  const pattern = /(?:^|\/)roles\/([^/]+)(?:$|\/)/;
  const matches = resourceId.match(pattern);
  return matches?.[1] ?? "";
};

export const displayRoleTitle = (role: string): string => {
  switch (role) {
    case PresetRoleType.WORKSPACE_ADMIN:
      return i18n.t("role.workspace-admin.self");
    case PresetRoleType.WORKSPACE_DBA:
      return i18n.t("role.workspace-dba.self");
    case PresetRoleType.WORKSPACE_MEMBER:
      return i18n.t("role.workspace-member.self");
    case PresetRoleType.PROJECT_OWNER:
      return i18n.t("role.project-owner.self");
    case PresetRoleType.PROJECT_DEVELOPER:
      return i18n.t("role.project-developer.self");
    case PresetRoleType.PROJECT_RELEASER:
      return i18n.t("role.project-releaser.self");
    case PresetRoleType.SQL_EDITOR_USER:
      return i18n.t("role.sql-editor-user.self");
    case PresetRoleType.SQL_EDITOR_READ_USER:
      return i18n.t("role.sql-editor-read-user.self");
    case PresetRoleType.GITOPS_SERVICE_AGENT:
      return i18n.t("role.gitops-service-agent.self");
    case PresetRoleType.PROJECT_VIEWER:
      return i18n.t("role.project-viewer.self");
  }
  // Use role.title if possible
  const item = appStoreUtilBridge()?.getRoleByName(role);
  // Fallback to extracted resource name otherwise
  return item?.title || extractRoleResourceName(role);
};

export const displayRoleDescription = (role: string): string => {
  switch (role) {
    case PresetRoleType.WORKSPACE_ADMIN:
      return i18n.t("role.workspace-admin.description");
    case PresetRoleType.WORKSPACE_DBA:
      return i18n.t("role.workspace-dba.description");
    case PresetRoleType.WORKSPACE_MEMBER:
      return i18n.t("role.workspace-member.description");
    case PresetRoleType.PROJECT_OWNER:
      return i18n.t("role.project-owner.description");
    case PresetRoleType.PROJECT_DEVELOPER:
      return i18n.t("role.project-developer.description");
    case PresetRoleType.PROJECT_RELEASER:
      return i18n.t("role.project-releaser.description");
    case PresetRoleType.SQL_EDITOR_USER:
      return i18n.t("role.sql-editor-user.description");
    case PresetRoleType.SQL_EDITOR_READ_USER:
      return i18n.t("role.sql-editor-read-user.description");
    case PresetRoleType.GITOPS_SERVICE_AGENT:
      return i18n.t("role.gitops-service-agent.description");
    case PresetRoleType.PROJECT_VIEWER:
      return i18n.t("role.project-viewer.description");
  }
  // Use role.description if possible
  const item = appStoreUtilBridge()?.getRoleByName(role);
  // Fallback to extracted resource name otherwise
  return item?.description || extractRoleResourceName(role);
};
