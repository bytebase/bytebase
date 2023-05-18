export const PresetRoleType = {
  OWNER: "roles/OWNER",
  DEVELOPER: "roles/DEVELOPER",
  EXPORTER: "roles/EXPORTER",
  QUERIER: "roles/QUERIER",
};

export const PresetRoleTypeList = [
  PresetRoleType.OWNER,
  PresetRoleType.DEVELOPER,
  PresetRoleType.EXPORTER,
  PresetRoleType.QUERIER,
];

export const isCustomRole = (role: string) => {
  return !PresetRoleTypeList.includes(role);
};
