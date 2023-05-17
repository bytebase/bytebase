export const PresetRoleType = {
  Owner: "roles/OWNER",
  Developer: "roles/DEVELOPER",
  Exporter: "roles/EXPORTER",
  Querier: "roles/QUERIER",
};

export const PresetRoleTypeList = [
  PresetRoleType.Owner,
  PresetRoleType.Developer,
  PresetRoleType.Exporter,
  PresetRoleType.Querier,
];

export const isCustomRole = (role: string) => {
  return !PresetRoleTypeList.includes(role);
};
