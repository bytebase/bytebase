import { usePermissionStore } from "@/store";
import type { Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

export const hasWorkspacePermissionV2 = (permission: Permission): boolean => {
  return usePermissionStore().currentPermissions.has(permission);
};

// hasProjectPermissionV2 checks if the user has the given permission on the project.
export const hasProjectPermissionV2 = (
  project: Project,
  permission: Permission
): boolean => {
  // Check workspace-level permissions first.
  // For those users who have workspace-level project roles, they should have all project-level permissions.
  if (hasWorkspacePermissionV2(permission)) {
    return true;
  }

  // Check project-level permissions.
  const permissions =
    usePermissionStore().currentPermissionsInProjectV1(project);
  return permissions.has(permission);
};
