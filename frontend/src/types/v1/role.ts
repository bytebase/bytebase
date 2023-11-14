export const PresetRoleType = {
  OWNER: "roles/OWNER",
  DEVELOPER: "roles/DEVELOPER",
  QUERIER: "roles/QUERIER",
  EXPORTER: "roles/EXPORTER",
  RELEASER: "roles/RELEASER",
  VIEWER: "roles/VIEWER",
};

export const PresetRoleTypeList = [
  PresetRoleType.OWNER,
  PresetRoleType.DEVELOPER,
  PresetRoleType.QUERIER,
  PresetRoleType.EXPORTER,
  PresetRoleType.RELEASER,
  PresetRoleType.VIEWER,
];

export const VirtualRoleType = {
  OWNER: "roles/OWNER",
  DBA: "roles/DBA",
  LAST_APPROVER: "roles/LAST_APPROVER",
  CREATOR: "roles/CREATOR",
};

export const IssueReleaserRoleType = {
  WORKSPACE_OWNER: "roles/workspaceOwner",
  WORKSPACE_DBA: "roles/workspaceDBA",
  PROJECT_OWNER: "roles/projectOwner",
  PROJECT_RELEASER: "roles/projectReleaser",
};

export const isCustomRole = (role: string) => {
  return !PresetRoleTypeList.includes(role);
};
