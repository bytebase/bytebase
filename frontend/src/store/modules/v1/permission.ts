import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { shallowReactive } from "vue";
import {
  PRESET_WORKSPACE_ROLES,
  type ComposedProject,
  type ProjectPermission,
  type ComposedUser,
  type Permission,
} from "@/types";
import { roleListInIAM } from "@/utils";
import { useRoleStore } from "../role";

export const usePermissionStore = defineStore("permission", () => {
  const projectRoleListCache = shallowReactive(new Map<string, string[]>());
  const projectPermissionsCache = shallowReactive(
    new Map<string, Set<ProjectPermission>>()
  );
  const workspaceLevelPermissionsMapByUserName = shallowReactive(
    new Map<string, Set<Permission>>()
  );
  const roleStore = useRoleStore();

  const workspaceLevelPermissionsByUser = (
    user: ComposedUser
  ): Set<Permission> => {
    const cached = workspaceLevelPermissionsMapByUserName.get(user.name);
    if (cached) {
      return cached;
    }
    const roleStore = useRoleStore();
    const permissions = new Set(
      user.roles
        .map((role) => roleStore.getRoleByName(role))
        .flatMap((role) => (role ? role.permissions : []) as Permission[])
    );
    workspaceLevelPermissionsMapByUserName.set(user.name, permissions);
    return permissions;
  };

  const roleListInProjectV1 = (
    project: ComposedProject,
    user: ComposedUser
  ) => {
    const key = `${user.name}@@${project.name}`;
    const cached = projectRoleListCache.get(key);
    if (cached) {
      return cached;
    }

    const workspaceLevelProjectRoles = user.roles.filter(
      (role) => !PRESET_WORKSPACE_ROLES.includes(role)
    );

    const { iamPolicy } = project;
    const projectBindingRoles = roleListInIAM(iamPolicy, user.email);

    const result = uniq([
      ...projectBindingRoles,
      ...workspaceLevelProjectRoles,
    ]);
    projectRoleListCache.set(key, result);
    return result;
  };

  const permissionsInProjectV1 = (
    project: ComposedProject,
    user: ComposedUser
  ): Set<Permission> => {
    const key = `${user.name}@@${project.name}`;
    const cached = projectPermissionsCache.get(key);
    if (cached) {
      return cached;
    }

    const roles = roleListInProjectV1(project, user);
    const permissions = new Set(
      roles
        .map((role) => roleStore.getRoleByName(role))
        .flatMap(
          (role) => (role ? role.permissions : []) as ProjectPermission[]
        )
    );
    projectPermissionsCache.set(key, permissions);
    return permissions;
  };

  const invalidCacheByProject = (project: string) => {
    const roleListKeys = Array.from(projectRoleListCache.keys()).filter((key) =>
      key.endsWith(`@@${project}`)
    );
    roleListKeys.forEach((key) => projectRoleListCache.delete(key));

    const permissionsKeys = Array.from(projectPermissionsCache.keys()).filter(
      (key) => key.endsWith(`@@${project}`)
    );
    permissionsKeys.forEach((key) => projectPermissionsCache.delete(key));
  };
  const invalidCacheByUser = (user: ComposedUser) => {
    workspaceLevelPermissionsMapByUserName.delete(user.name);

    const roleListKeys = Array.from(projectRoleListCache.keys()).filter((key) =>
      key.startsWith(`${user.name}@@`)
    );
    roleListKeys.forEach((key) => projectRoleListCache.delete(key));

    const permissionsKeys = Array.from(projectPermissionsCache.keys()).filter(
      (key) => key.startsWith(`${user.name}@@`)
    );
    permissionsKeys.forEach((key) => projectPermissionsCache.delete(key));
  };

  return {
    workspaceLevelPermissionsByUser,
    roleListInProjectV1,
    permissionsInProjectV1,
    invalidCacheByProject,
    invalidCacheByUser,
  };
});
