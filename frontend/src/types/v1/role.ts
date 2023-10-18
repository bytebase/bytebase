export const PresetRoleType = {
  OWNER: "roles/OWNER",
  DEVELOPER: "roles/DEVELOPER",
  QUERIER: "roles/QUERIER",
  EXPORTER: "roles/EXPORTER",
  RELEASER: "roles/RELEASER",
};

export const PresetRoleTypeList = [
  PresetRoleType.OWNER,
  PresetRoleType.DEVELOPER,
  PresetRoleType.QUERIER,
  PresetRoleType.EXPORTER,
  PresetRoleType.RELEASER,
];

export const isCustomRole = (role: string) => {
  return !PresetRoleTypeList.includes(role);
};
