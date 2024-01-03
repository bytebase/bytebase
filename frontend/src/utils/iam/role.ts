import { useRoleStore } from "@/store";
import {
  WorkspaceLevelRoles,
  ProjectLevelRoles,
  WORKSPACE_PERMISSIONS,
  WorkspacePermission,
} from "@/types";

export const isWorkspaceLevelRole = (role: string) => {
  const roleStore = useRoleStore();
  return (
    WorkspaceLevelRoles.includes(role) &&
    roleStore
      .getRoleByName(role)
      ?.permissions.every((permission) =>
        WORKSPACE_PERMISSIONS.includes(permission as WorkspacePermission)
      )
  );
};

export const isProjectLevelRole = (role: string) => {
  return ProjectLevelRoles.includes(role) || !isWorkspaceLevelRole(role);
};

export const isCustomRole = (role: string) => {
  return (
    !WorkspaceLevelRoles.includes(role) && !ProjectLevelRoles.includes(role)
  );
};
