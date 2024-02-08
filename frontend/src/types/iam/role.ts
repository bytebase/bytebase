export const PresetRoleType = {
  WORKSPACE_ADMIN: "roles/workspaceAdmin",
  WORKSPACE_DBA: "roles/workspaceDBA",
  WORKSPACE_MEMBER: "roles/workspaceMember",
  PROJECT_OWNER: "roles/projectOwner",
  PROJECT_DEVELOPER: "roles/projectDeveloper",
  PROJECT_QUERIER: "roles/projectQuerier",
  PROJECT_EXPORTER: "roles/projectExporter",
  PROJECT_RELEASER: "roles/projectReleaser",
  PROJECT_VIEWER: "roles/projectViewer",
};

export const PRESET_WORKSPACE_ROLES = [
  PresetRoleType.WORKSPACE_ADMIN,
  PresetRoleType.WORKSPACE_DBA,
  PresetRoleType.WORKSPACE_MEMBER,
];

export const PRESET_ROLES = Object.values(PresetRoleType);
