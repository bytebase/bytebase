import { usePermissionStore } from "@/store";
import type { ComposedProject, Permission } from "@/types";

export const hasWorkspacePermissionV2 = (permission: Permission): boolean => {
  return usePermissionStore().currentPermissions.has(permission);
};

// hasProjectPermissionV2 checks if the user has the given permission on the project.
export const hasProjectPermissionV2 = (
  project: ComposedProject,
  permission: Permission
): boolean => {
  const permissionStore = usePermissionStore();

  // Check workspace-level permissions first.
  // For those users who have workspace-level project roles, they should have all project-level permissions.
  if (permissionStore.currentPermissions.has(permission)) {
    return true;
  }

  // Check project-level permissions.
  const permissions = permissionStore.currentPermissionsInProjectV1(project);
  return permissions.has(permission);
};
