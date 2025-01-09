import { t } from "@/plugins/i18n";
import { useRoleStore } from "@/store";
import { PRESET_ROLES } from "@/types";

const roleTranslations = {
  "workspace-admin": "role.workspace-admin",
  "workspace-dba": "role.workspace-dba",
  "workspace-member": "role.workspace-member",
  "project-owner": "role.project-owner",
  "project-developer": "role.project-developer",
  "project-releaser": "role.project-releaser",
  "project-querier": "role.project-querier",
  "sql-editor-user": "role.sql-editor-user",
  "project-exporter": "role.project-exporter",
  "project-viewer": "role.project-viewer"
};

export const extractRoleResourceName = (resourceId: string): string => {
  const pattern = /(?:^|\/)roles\/([^/]+)(?:$|\/)/;
  const matches = resourceId.match(pattern);
  return matches?.[1] ?? "";
};

export const displayRoleTitle = (role: string): string => {
  if (PRESET_ROLES.includes(role)) {
    return t(roleTranslations[extractRoleResourceName(role) as keyof typeof roleTranslations] + ".self");
  }
  // Use role.title if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.title || extractRoleResourceName(role);
};

export const displayRoleDescription = (role: string): string => {
  if (PRESET_ROLES.includes(role)) {
    return t(roleTranslations[extractRoleResourceName(role) as keyof typeof roleTranslations] + ".description");
  }
  // Use role.description if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.description || extractRoleResourceName(role);
};
