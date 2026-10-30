import { PRESET_ROLES, PresetRoleType } from "@/types";
import type { Role } from "@/types/proto-es/v1/role_service_pb";

const SQL_SELECT_PERMISSION = "bb.sql.select";

export const getDefaultQueryRole = (
  roles: readonly Pick<Role, "name" | "permissions">[],
  requiredPermissions: readonly string[],
  hasCustomRoleFeature: boolean
) => {
  const permissions =
    requiredPermissions.length > 0
      ? requiredPermissions
      : [SQL_SELECT_PERMISSION];
  const candidates = [...roles]
    .filter((role) => hasCustomRoleFeature || PRESET_ROLES.includes(role.name))
    .filter((role) =>
      permissions.every((permission) => role.permissions.includes(permission))
    )
    .sort((a, b) => {
      const permissionCountDelta = a.permissions.length - b.permissions.length;
      if (permissionCountDelta !== 0) return permissionCountDelta;
      if (a.name === PresetRoleType.SQL_EDITOR_READ_USER) return -1;
      if (b.name === PresetRoleType.SQL_EDITOR_READ_USER) return 1;
      return a.name.localeCompare(b.name);
    });

  return candidates[0]?.name ?? PresetRoleType.SQL_EDITOR_USER;
};
