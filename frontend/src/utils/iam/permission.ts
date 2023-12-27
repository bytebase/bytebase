import { useRoleStore } from "@/store";
import { WorkspacePermission } from "@/types/iam";
import { User } from "@/types/proto/v1/auth_service";

export function hasWorkspacePermissionV2(
  user: User,
  permission: WorkspacePermission
): boolean {
  const roleStore = useRoleStore();
  const permissions = user.roles
    .map((role) => roleStore.getRoleByName(role))
    .flatMap((role) => role.permissions);
  return permissions.includes(permission);
}
