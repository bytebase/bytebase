import { t } from "@/plugins/i18n";
import { UserRole, userRoleToJSON } from "@/types/proto/v1/auth_service";

export function roleNameV1(role: UserRole): string {
  switch (role) {
    case UserRole.OWNER:
      return t("role.workspace-admin.self");
    case UserRole.DBA:
      return t("role.workspace-dba.self");
    case UserRole.DEVELOPER:
      return t("role.workspace-member.self");
  }
  return userRoleToJSON(role);
}
