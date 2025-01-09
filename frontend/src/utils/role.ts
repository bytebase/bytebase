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
      case "workspaceAdmin":
        return t("role.workspace-admin.self");
      case "workspaceDBA":
        return t("role.workspace-dba.self");
      case "workspaceMember":
        return t("role.workspace-member.self");
      case "projectOwner":
        return t("role.project-owner.self");
      case "projectDeveloper":
        return t("role.project-developer.self");
      case "projectReleaser":
        return t("role.project-releaser.self");
      case "sqlEditorUser":
        return t("role.sql-editor-user.self");
      case "projectExporter":
        return t("role.project-exporter.self");
      case "projectViewer":
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
      case "workspaceAdmin":
        return t("role.workspace-admin.description");
      case "workspaceDBA":
        return t("role.workspace-dba.description");
      case "workspaceMember":
        return t("role.workspace-member.description");
      case "projectOwner":
        return t("role.project-owner.description");
      case "projectDeveloper":
        return t("role.project-developer.description");
      case "projectReleaser":
        return t("role.project-releaser.description");
      case "sqlEditorUser":
        return t("role.sql-editor-user.description");
      case "projectExporter":
        return t("role.project-exporter.description");
      case "projectViewer":
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
