export const PresetRoleType = {
  OWNER: "roles/OWNER",
  DEVELOPER: "roles/DEVELOPER",
  QUERIER: "roles/QUERIER",
  EXPORTER: "roles/EXPORTER",
};

export const PresetRoleTypeList = [
  PresetRoleType.OWNER,
  PresetRoleType.DEVELOPER,
  PresetRoleType.QUERIER,
  PresetRoleType.EXPORTER,
];

export const isCustomRole = (role: string) => {
  return !PresetRoleTypeList.includes(role);
};
