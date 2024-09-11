import { PRESET_ROLES } from "@/types";

export const isCustomRole = (role: string) => {
  return !PRESET_ROLES.includes(role);
};

export const sortRoles = (roles: string[]) => {
  return roles.sort((a, b) => {
    const priority = (role: string) => {
      const presetRoleIndex = PRESET_ROLES.indexOf(role);
      if (presetRoleIndex !== -1) {
        return presetRoleIndex;
      }
      return PRESET_ROLES.length + roles.indexOf(role);
    };
    return priority(a) - priority(b);
  });
};
