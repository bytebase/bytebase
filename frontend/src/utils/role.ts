import { t } from "@/plugins/i18n";
import { useRoleStore } from "@/store";
import { PRESET_ROLES } from "@/types";

const roleTranslationsKey = {
  "workspace-admin": {
    self: "role.workspace-admin.self",
    description: "role.workspace-admin.description"
  },
  "workspace-dba": {
    self: "role.workspace-dba.self",
    description: "role.workspace-dba.description"
  },
  "workspace-member": {
    self: "role.workspace-member.self",
    description: "role.workspace-member.description"
  },
  "project-owner": {
    self: "role.project-owner.self",
    description: "role.project-owner.description"
  },
  "project-developer": {
    self: "role.project-developer.self",
    description: "role.project-developer.description"
  },
  "project-releaser": {
    self: "role.project-releaser.self",
    description: "role.project-releaser.description"
  },
  "project-querier": {
    self: "role.project-querier.self",
    description: "role.project-querier.description"
  },
  "sql-editor-user": {
    self: "role.sql-editor-user.self",
    description: "role.sql-editor-user.description"
  },
  "project-exporter": {
    self: "role.project-exporter.self",
    description: "role.project-exporter.description"
  },
  "project-viewer": {
    self: "role.project-viewer.self",
    description: "role.project-viewer.description"
  }
};

export const extractRoleResourceName = (resourceId: string): string => {
  const pattern = /(?:^|\/)roles\/([^/]+)(?:$|\/)/;
  const matches = resourceId.match(pattern);
  return matches?.[1] ?? "";
};

export const displayRoleTitle = (role: string): string => {
  if (PRESET_ROLES.includes(role)) {
    return t(roleTranslationsKey[extractRoleResourceName(role) as keyof typeof roleTranslationsKey].self);
  }
  // Use role.title if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.title || extractRoleResourceName(role);
};

export const displayRoleDescription = (role: string): string => {
  if (PRESET_ROLES.includes(role)) {
    return t(roleTranslationsKey[extractRoleResourceName(role) as keyof typeof roleTranslationsKey].description);
  }
  // Use role.description if possible
  const item = useRoleStore().roleList.find((r) => r.name === role);
  // Fallback to extracted resource name otherwise
  return item?.description || extractRoleResourceName(role);
};
