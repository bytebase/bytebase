import { usePermissionStore, useProjectV1List } from "@/store";
import type { ComposedProject, Permission } from "@/types";

export const hasWorkspacePermissionV2 = (permission: Permission): boolean => {
  return usePermissionStore().currentPermissions.has(permission);
};

// hasProjectPermissionV2 checks if the user has the given permission on the project.
export const hasProjectPermissionV2 = (
  project: ComposedProject | undefined,
  permission: Permission
): boolean => {
  const permissionStore = usePermissionStore();

  // If the project is not provided, then check if the user has the given permission on any project in the workspace.
  if (!project) {
    const { projectList } = useProjectV1List();
    return projectList.value.some((project) =>
      hasProjectPermissionV2(project, permission)
    );
  }

  // Check workspace-level permissions first.
  // For those users who have workspace-level project roles, they should have all project-level permissions.
  if (permissionStore.currentPermissions.has(permission)) {
    return true;
  }

  // Check project-level permissions.
  const permissions = permissionStore.currentPermissionsInProjectV1(project);
  return permissions.has(permission);
};

// hasWorkspaceLevelProjectPermissionInAnyProject checks if the user has the given permission on ANY project in the workspace.
export const hasWorkspaceLevelProjectPermissionInAnyProject = (
  permission: Permission
): boolean => {
  const { projectList } = useProjectV1List();
  return projectList.value.some((project) =>
    hasProjectPermissionV2(project, permission)
  );
};
