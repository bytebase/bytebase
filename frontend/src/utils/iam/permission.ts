import {
  useActuatorV1Store,
  useProjectV1ListByUser,
  useRoleStore,
} from "@/store";
import {
  ComposedProject,
  PresetRoleType,
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

export const hasProjectPermissionV2 = (
  project: ComposedProject,
  user: User,
  permission: ProjectPermission
): boolean => {
  const roleStore = useRoleStore();
  const actuatorStore = useActuatorV1Store();
  const isDevelopmentIAM = actuatorStore.serverInfo?.iamGuard;

  // Check workspace-level permissions first.
  if (isDevelopmentIAM) {
    // For those users who have workspace-level project roles, they should have all project-level permissions.
    const workspaceLevelPermissions = user.roles
      .map((role) => roleStore.getRoleByName(role))
      .flatMap((role) => (role ? role.permissions : []));
    if (workspaceLevelPermissions.includes(permission)) {
      return true;
    }
  } else {
    // TODO: Remove this block after fine-grained permissions are supported.
    const privilegedRoles = [
      PresetRoleType.WORKSPACE_ADMIN,
      PresetRoleType.WORKSPACE_DBA,
    ];
    if (user.roles.some((role) => privilegedRoles.includes(role))) {
      return true;
    }
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
  const { projectList } = useProjectV1ListByUser(user);
  return projectList.value.some((project) =>
    hasProjectPermissionV2(project, user, permission)
  );
};
