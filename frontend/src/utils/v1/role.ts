import { t } from "@/plugins/i18n";
import { UserRole, userRoleToJSON } from "@/types/proto/v1/auth_service";

export function roleNameV1(role: UserRole): string {
  switch (role) {
    case UserRole.OWNER:
      return t("common.role.owner");
    case UserRole.DBA:
      return t("common.role.dba");
    case UserRole.DEVELOPER:
      return t("common.role.developer");
  }
  return userRoleToJSON(role);
}
