import { t } from "@/plugins/i18n";
import { useRoleStore } from "@/store";
import { PRESET_ROLES, PresetRoleType } from "@/types";

export const extractRoleResourceName = (resourceId: string): string => {
  const pattern = /(?:^|\/)roles\/([^/]+)(?:$|\/)/;
  const matches = resourceId.match(pattern);
  return matches?.[1] ?? "";
};

export const displayRoleTitle = (role: string): string => {
  if (PRESET_ROLES.includes(role)) {
    switch (extractRoleResourceName(role)) {
      case PresetRoleType.WORKSPACE_ADMIN:
        return t("role.workspace-admin.self");
      case PresetRoleType.WORKSPACE_DBA:
        return t("role.workspace-dba.self");
      case PresetRoleType.WORKSPACE_MEMBER:
        return t("role.workspace-member.self");
      case PresetRoleType.PROJECT_OWNER:
        return t("role.project-owner.self");
      case PresetRoleType.PROJECT_DEVELOPER:
        return t("role.project-developer.self");
      case PresetRoleType.PROJECT_RELEASER:
        return t("role.project-releaser.self");
      case PresetRoleType.SQL_EDITOR_USER:
        return t("role.sql-editor-user.self");
      case PresetRoleType.PROJECT_EXPORTER:
        return t("role.project-exporter.self");
      case PresetRoleType.PROJECT_VIEWER:
        return t("role.project-viewer.self");
      default:
        return "UNKNOWN ROLE";
    }
  }
  // Use role.title if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.title || extractRoleResourceName(role);
};

export const displayRoleDescription = (role: string): string => {
  if (PRESET_ROLES.includes(role)) {
    switch (extractRoleResourceName(role)) {
      case PresetRoleType.WORKSPACE_ADMIN:
        return t("role.workspace-admin.description");
      case PresetRoleType.WORKSPACE_DBA:
        return t("role.workspace-dba.description");
      case PresetRoleType.WORKSPACE_MEMBER:
        return t("role.workspace-member.description");
      case PresetRoleType.PROJECT_OWNER:
        return t("role.project-owner.description");
      case PresetRoleType.PROJECT_DEVELOPER:
        return t("role.project-developer.description");
      case PresetRoleType.PROJECT_RELEASER:
        return t("role.project-releaser.description");
      case PresetRoleType.SQL_EDITOR_USER:
        return t("role.sql-editor-user.description");
      case PresetRoleType.PROJECT_EXPORTER:
        return t("role.project-exporter.description");
      case PresetRoleType.PROJECT_VIEWER:
        return t("role.project-viewer.description");
      default:
        return "UNKNOWN ROLE";
    }
  }
  // Use role.description if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.description || extractRoleResourceName(role);
};
