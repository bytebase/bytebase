import { roleNamePrefix } from "@/store/modules/v1/common";

export enum PresetRoleType {
  WORKSPACE_ADMIN = `${roleNamePrefix}workspaceAdmin`,
  WORKSPACE_DBA = `${roleNamePrefix}workspaceDBA`,
  WORKSPACE_MEMBER = `${roleNamePrefix}workspaceMember`,
  PROJECT_OWNER = `${roleNamePrefix}projectOwner`,
  PROJECT_DEVELOPER = `${roleNamePrefix}projectDeveloper`,
  SQL_EDITOR_USER = `${roleNamePrefix}sqlEditorUser`,
  PROJECT_EXPORTER = `${roleNamePrefix}projectExporter`,
  PROJECT_RELEASER = `${roleNamePrefix}projectReleaser`,
  PROJECT_VIEWER = `${roleNamePrefix}projectViewer`,
}

export const PRESET_ROLES: string[] = Object.values(PresetRoleType);

export const PRESET_WORKSPACE_ROLES: string[] = [
  PresetRoleType.WORKSPACE_ADMIN,
  PresetRoleType.WORKSPACE_DBA,
  PresetRoleType.WORKSPACE_MEMBER,
];

export const PRESET_PROJECT_ROLES: string[] = [
  PresetRoleType.PROJECT_OWNER,
  PresetRoleType.PROJECT_DEVELOPER,
  PresetRoleType.SQL_EDITOR_USER,
  PresetRoleType.PROJECT_EXPORTER,
  PresetRoleType.PROJECT_RELEASER,
  PresetRoleType.PROJECT_VIEWER,
];
