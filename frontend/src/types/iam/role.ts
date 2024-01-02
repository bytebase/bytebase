import { Role } from "../proto/v1/role_service";
import { ProjectPermission, WorkspacePermission } from "./permission";

export interface ComposedRole extends Role {
  permissions: WorkspacePermission[] | ProjectPermission[];
}

export const PresetRoleType = {
  // Workspace level roles.
  WORKSPACE_ADMIN: "roles/workspaceAdmin",
  WORKSPACE_DBA: "roles/workspaceDBA",
  WORKSPACE_MEMBER: "roles/workspaceMember",
  // Project level roles.
  PROJECT_OWNER: "roles/projectOwner",
  PROJECT_DEVELOPER: "roles/projectDeveloper",
  PROJECT_QUERIER: "roles/projectQuerier",
  PROJECT_EXPORTER: "roles/projectExporter",
  PROJECT_RELEASER: "roles/projectReleaser",
  PROJECT_VIEWER: "roles/projectViewer",
};

export const WorkspaceLevelRoles = [
  PresetRoleType.WORKSPACE_ADMIN,
  PresetRoleType.WORKSPACE_DBA,
  PresetRoleType.WORKSPACE_MEMBER,
];

export const ProjectLevelRoles = [
  PresetRoleType.PROJECT_OWNER,
  PresetRoleType.PROJECT_DEVELOPER,
  PresetRoleType.PROJECT_QUERIER,
  PresetRoleType.PROJECT_EXPORTER,
  PresetRoleType.PROJECT_RELEASER,
  PresetRoleType.PROJECT_VIEWER,
];
