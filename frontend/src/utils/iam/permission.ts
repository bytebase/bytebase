import { useRoleStore } from "@/store";
import {
  ComposedProject,
  ProjectPermission,
  WorkspacePermission,
} from "@/types";
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
  // Check workspace-level permissions first.
  // For those users who have workspace-level project roles, they should have all project-level permissions.
  const workspaceLevelPermissions = user.roles
    .map((role) => roleStore.getRoleByName(role))
    .flatMap((role) => (role ? role.permissions : []));
  if (workspaceLevelPermissions.includes(permission)) {
    return true;
  }

  // Check project-level permissions.
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
