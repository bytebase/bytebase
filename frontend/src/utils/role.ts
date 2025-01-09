import { t } from "@/plugins/i18n";
import { useRoleStore } from "@/store";
import { PRESET_ROLES } from "@/types";

export const extractRoleResourceName = (resourceId: string): string => {
  const pattern = /(?:^|\/)roles\/([^/]+)(?:$|\/)/;
  const matches = resourceId.match(pattern);
  return matches?.[1] ?? "";
};

export const displayRoleTitle = (role: string): string => {
  if (PRESET_ROLES.includes(role)) {
    switch (extractRoleResourceName(role)) {
      case "workspace-admin":
        return t("role.workspace-admin.self");
      case "workspace-dba":
        return t("role.workspace-dba.self");
      case "workspace-member":
        return t("role.workspace-member.self");
      case "project-owner":
        return t("role.project-owner.self");
      case "project-developer":
        return t("role.project-developer.self");
      case "project-releaser":
        return t("role.project-releaser.self");
      case "project-querier":
        return t("role.project-querier.self");
      case "sql-editor-user":
        return t("role.sql-editor-user.self");
      case "project-exporter":
        return t("role.project-exporter.self");
      case "project-viewer":
        return t("role.project-viewer.self");
      default:
        return "";
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
      case "workspace-admin":
        return t("role.workspace-admin.description");
      case "workspace-dba":
        return t("role.workspace-dba.description");
      case "workspace-member":
        return t("role.workspace-member.description");
      case "project-owner":
        return t("role.project-owner.description");
      case "project-developer":
        return t("role.project-developer.description");
      case "project-releaser":
        return t("role.project-releaser.description");
      case "project-querier":
        return t("role.project-querier.description");
      case "sql-editor-user":
        return t("role.sql-editor-user.description");
      case "project-exporter":
        return t("role.project-exporter.description");
      case "project-viewer":
        return t("role.project-viewer.description");
      default:
        return "";
    }
  }
  // Use role.description if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.description || extractRoleResourceName(role);
};
