export const PresetRoleType = {
  PROJECT_OWNER: "roles/projectOwner",
  PROJECT_DEVELOPER: "roles/projectDeveloper",
  PROJECT_QUERIER: "roles/projectQuerier",
  PROJECT_EXPORTER: "roles/projectExporter",
  PROJECT_RELEASER: "roles/projectReleaser",
  PROJECT_VIEWER: "roles/projectViewer",
};

export const PresetRoleTypeList = [
  PresetRoleType.PROJECT_OWNER,
  PresetRoleType.PROJECT_DEVELOPER,
  PresetRoleType.PROJECT_QUERIER,
  PresetRoleType.PROJECT_EXPORTER,
  PresetRoleType.PROJECT_RELEASER,
  PresetRoleType.PROJECT_VIEWER,
];

export const VirtualRoleType = {
  WORKSPACE_ADMIN: "roles/workspaceAdmin",
  WORKSPACE_DBA: "roles/workspaceDBA",
  LAST_APPROVER: "roles/LAST_APPROVER",
  CREATOR: "roles/CREATOR",
};

export const IssueReleaserRoleType = {
  WORKSPACE_ADMIN: "roles/workspaceAdmin",
  WORKSPACE_DBA: "roles/workspaceDBA",
  PROJECT_OWNER: "roles/projectOwner",
  PROJECT_RELEASER: "roles/projectReleaser",
};

export const isCustomRole = (role: string) => {
  return !PresetRoleTypeList.includes(role);
};
