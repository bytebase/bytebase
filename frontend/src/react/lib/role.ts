import { t } from "@/plugins/i18n";
import type { Role } from "@/types/proto-es/v1/role_service_pb";

const PresetRoleType = {
  WORKSPACE_ADMIN: "roles/workspaceAdmin",
  WORKSPACE_DBA: "roles/workspaceDBA",
  WORKSPACE_MEMBER: "roles/workspaceMember",
  PROJECT_OWNER: "roles/projectOwner",
  PROJECT_DEVELOPER: "roles/projectDeveloper",
  SQL_EDITOR_USER: "roles/sqlEditorUser",
  SQL_EDITOR_READ_USER: "roles/sqlEditorReadUser",
  PROJECT_RELEASER: "roles/projectReleaser",
  GITOPS_SERVICE_AGENT: "roles/gitopsServiceAgent",
  PROJECT_VIEWER: "roles/projectViewer",
} as const;

const PRESET_ROLES: string[] = Object.values(PresetRoleType);

const PRESET_ROLE_TITLE_KEYS: Record<string, string> = {
  [PresetRoleType.WORKSPACE_ADMIN]: "role.workspace-admin.self",
  [PresetRoleType.WORKSPACE_DBA]: "role.workspace-dba.self",
  [PresetRoleType.WORKSPACE_MEMBER]: "role.workspace-member.self",
  [PresetRoleType.PROJECT_OWNER]: "role.project-owner.self",
  [PresetRoleType.PROJECT_DEVELOPER]: "role.project-developer.self",
  [PresetRoleType.PROJECT_RELEASER]: "role.project-releaser.self",
  [PresetRoleType.SQL_EDITOR_USER]: "role.sql-editor-user.self",
  [PresetRoleType.SQL_EDITOR_READ_USER]: "role.sql-editor-read-user.self",
  [PresetRoleType.GITOPS_SERVICE_AGENT]: "role.gitops-service-agent.self",
  [PresetRoleType.PROJECT_VIEWER]: "role.project-viewer.self",
};

const PRESET_ROLE_DESCRIPTION_KEYS: Record<string, string> = {
  [PresetRoleType.WORKSPACE_ADMIN]: "role.workspace-admin.description",
  [PresetRoleType.WORKSPACE_DBA]: "role.workspace-dba.description",
  [PresetRoleType.WORKSPACE_MEMBER]: "role.workspace-member.description",
  [PresetRoleType.PROJECT_OWNER]: "role.project-owner.description",
  [PresetRoleType.PROJECT_DEVELOPER]: "role.project-developer.description",
  [PresetRoleType.PROJECT_RELEASER]: "role.project-releaser.description",
  [PresetRoleType.SQL_EDITOR_USER]: "role.sql-editor-user.description",
  [PresetRoleType.SQL_EDITOR_READ_USER]:
    "role.sql-editor-read-user.description",
  [PresetRoleType.GITOPS_SERVICE_AGENT]:
    "role.gitops-service-agent.description",
  [PresetRoleType.PROJECT_VIEWER]: "role.project-viewer.description",
};

const extractRoleResourceName = (resourceId: string): string => {
  const pattern = /(?:^|\/)roles\/([^/]+)(?:$|\/)/;
  const matches = resourceId.match(pattern);
  return matches?.[1] ?? "";
};

const displayPresetRoleTitle = (roleName: string): string => {
  const key = PRESET_ROLE_TITLE_KEYS[roleName];
  return key ? t(key) : roleName;
};

const displayPresetRoleDescription = (roleName: string): string => {
  const key = PRESET_ROLE_DESCRIPTION_KEYS[roleName];
  return key ? t(key) : roleName;
};

export const displayRoleTitleFromList = (
  roleName: string,
  roleList: Role[]
) => {
  if (PRESET_ROLES.includes(roleName)) {
    return displayPresetRoleTitle(roleName);
  }
  const role = roleList.find((role) => role.name === roleName);
  return role?.title || extractRoleResourceName(roleName) || roleName;
};

export const displayRoleDescriptionFromList = (
  roleName: string,
  roleList: Role[]
) => {
  if (PRESET_ROLES.includes(roleName)) {
    return displayPresetRoleDescription(roleName);
  }
  return roleList.find((role) => role.name === roleName)?.description;
};
