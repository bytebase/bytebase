import { useProjectV1List, useRoleStore } from "@/store";
import {
  ComposedProject,
  ProjectPermission,
  WorkspacePermission,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { roleListInProjectV1 } from "../v1/project";

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

// hasProjectPermissionV2 checks if the user has the given permission on the project.
export const hasProjectPermissionV2 = (
  project: ComposedProject | undefined,
  user: User,
  permission: ProjectPermission
): boolean => {
  // If the project is not provided, then check if the user has the given permission on any project in the workspace.
  if (!project) {
    const { projectList } = useProjectV1List();
    return projectList.value.some((project) =>
      hasProjectPermissionV2(project, user, permission)
    );
  }

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
  const roles = roleListInProjectV1(project.iamPolicy, user);
  const permissions = roles
    .map((role) => roleStore.getRoleByName(role))
    .flatMap((role) => (role ? role.permissions : []));
  return permissions.includes(permission);
};

// hasWorkspaceLevelProjectPermission checks if the user has the given permission on any project in the workspace.
export const hasWorkspaceLevelProjectPermission = (
  user: User,
  permission: ProjectPermission
): boolean => {
  const { projectList } = useProjectV1List();
  return projectList.value.some((project) =>
    hasProjectPermissionV2(project, user, permission)
  );
};
