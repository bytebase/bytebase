import { useRoleStore } from "@/store";
import { ComposedProject } from "@/types";
import { ProjectPermission, WorkspacePermission } from "@/types/iam";
import { User } from "@/types/proto/v1/auth_service";

export const hasWorkspacePermissionV2 = (
  user: User,
  permission: WorkspacePermission
): boolean => {
  const roleStore = useRoleStore();
  const permissions = user.roles
    .map((role) => roleStore.getRoleByName(role))
    .flatMap((role) => (role ? role.permissions : []));
  return permissions.includes(permission);
};

export const hasProjectPermissionV2 = (
  project: ComposedProject,
  user: User,
  permission: ProjectPermission
): boolean => {
  const roleStore = useRoleStore();
  const projectIAMPolicy = project.iamPolicy;
  const roles = [];
  for (const binding of projectIAMPolicy.bindings) {
    if (binding.members.includes(`user:${user.email}`)) {
      roles.push(binding.role);
    }
  }
  const permissions = roles
    .map((role) => roleStore.getRoleByName(role))
    .flatMap((role) => (role ? role.permissions : []));
  return permissions.includes(permission);
};
