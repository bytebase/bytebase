import { usePermissionStore, useProjectV1List } from "@/store";
import type {
  ComposedProject,
  ProjectPermission,
  WorkspacePermission,
} from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { TinyTimer } from "../TinyTimer";

const timer = new TinyTimer<
  | "hasWorkspacePermissionV2"
  | "hasProjectPermissionV2.1"
  | "hasProjectPermissionV2.2"
  | "hasProjectPermissionV2.3"
  | "hasProjectPermissionV2.3.1"
  | "hasProjectPermissionV2.3.2"
  | "hasWorkspaceLevelProjectPermission"
  | "hasWorkspaceLevelProjectPermissionInAnyProject"
>("permission");
(window as any).__permissionTimer = timer;

export const hasWorkspacePermissionV2 = (
  user: User,
  permission: WorkspacePermission
): boolean => {
  timer.begin("hasWorkspacePermissionV2");
  const permissions =
    usePermissionStore().workspaceLevelPermissionsByUser(user);
  const ok = permissions.has(permission);
  timer.end("hasWorkspacePermissionV2");
  return ok;
};

// hasProjectPermissionV2 checks if the user has the given permission on the project.
export const hasProjectPermissionV2 = (
  project: ComposedProject | undefined,
  user: User,
  permission: ProjectPermission
): boolean => {
  const permissionStore = usePermissionStore();

  timer.begin("hasProjectPermissionV2.1");
  // If the project is not provided, then check if the user has the given permission on any project in the workspace.
  if (!project) {
    const { projectList } = useProjectV1List();
    const ok = projectList.value.some((project) =>
      hasProjectPermissionV2(project, user, permission)
    );
    timer.end("hasProjectPermissionV2.1", projectList.value.length);
    return ok;
  } else {
    timer.end("hasProjectPermissionV2.1");
  }

  timer.begin("hasProjectPermissionV2.2");

  // Check workspace-level permissions first.
  // For those users who have workspace-level project roles, they should have all project-level permissions.
  const workspaceLevelPermissions =
    permissionStore.workspaceLevelPermissionsByUser(user);

  if (workspaceLevelPermissions.has(permission)) {
    timer.end("hasProjectPermissionV2.2", workspaceLevelPermissions.size);
    return true;
  } else {
    timer.end("hasProjectPermissionV2.2", workspaceLevelPermissions.size);
  }

  timer.begin("hasProjectPermissionV2.3");
  // Check project-level permissions.

  timer.begin("hasProjectPermissionV2.3.2");
  const permissions = permissionStore.permissionsInProjectV1(project, user);
  timer.end("hasProjectPermissionV2.3.2");
  const ok = permissions.has(permission);
  timer.end("hasProjectPermissionV2.3", permissions.size);

  return ok;
};

// hasWorkspaceLevelProjectPermission checks if the user has the given permission on workspace-level-assigned project roles
export const hasWorkspaceLevelProjectPermission = (
  user: User,
  permission: ProjectPermission
): boolean => {
  timer.begin("hasWorkspaceLevelProjectPermission");
  const permissions =
    usePermissionStore().workspaceLevelPermissionsByUser(user);
  const ok = permissions.has(permission);
  timer.end("hasWorkspaceLevelProjectPermission");
  return ok;
};

// hasWorkspaceLevelProjectPermission checks if the user has the given permission on ANY project in the workspace.
export const hasWorkspaceLevelProjectPermissionInAnyProject = (
  user: User,
  permission: ProjectPermission
): boolean => {
  timer.begin("hasWorkspaceLevelProjectPermissionInAnyProject");
  const { projectList } = useProjectV1List();
  const ok = projectList.value.some((project) =>
    hasProjectPermissionV2(project, user, permission)
  );
  timer.end("hasWorkspaceLevelProjectPermissionInAnyProject");
  return ok;
};
