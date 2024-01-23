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

export const sortRoles = (roles: string[]) => {
  return roles.sort((a, b) => {
    const priority = (role: string) => {
      if (WorkspaceLevelRoles.includes(role)) {
        return WorkspaceLevelRoles.indexOf(role);
      }
      if (ProjectLevelRoles.includes(role)) {
        return ProjectLevelRoles.indexOf(role) + WorkspaceLevelRoles.length;
      }
      return roles.length + roles.indexOf(role);
    };
    return priority(a) - priority(b);
  });
};
