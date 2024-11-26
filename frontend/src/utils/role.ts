import { kebabCase } from "lodash-es";
import { t } from "@/plugins/i18n";
import { useRoleStore } from "@/store";
import { PRESET_ROLES } from "@/types";

export const extractRoleResourceName = (resourceId: string): string => {
  const pattern = /(?:^|\/)roles\/([^/]+)(?:$|\/)/;
  const matches = resourceId.match(pattern);
  return matches?.[1] ?? "";
};

export const displayRoleTitle = (role: string): string => {
  if (PRESET_ROLES.includes(role)) {
    return t(`role.${kebabCase(extractRoleResourceName(role))}.self`);
  }
  // Use role.title if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.title || extractRoleResourceName(role);
};

export const displayRoleDescription = (role: string): string => {
  if (PRESET_ROLES.includes(role)) {
    return t(`role.${kebabCase(extractRoleResourceName(role))}.description`);
  }
  // Use role.description if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.description || extractRoleResourceName(role);
};
