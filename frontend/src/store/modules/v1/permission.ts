import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { shallowReactive } from "vue";
import {
  ALL_USERS_USER_EMAIL,
  getUserEmailInBinding,
  groupBindingPrefix,
  PRESET_WORKSPACE_ROLES,
  type ComposedProject,
  type ProjectPermission,
  type WorkspacePermission,
} from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { isBindingPolicyExpired } from "@/utils";
import { useRoleStore } from "../role";
import { userNamePrefix } from "./common";
import { useUserGroupStore } from "./userGroup";

export const usePermissionStore = defineStore("permission", () => {
  const projectRoleListCache = shallowReactive(new Map<string, string[]>());
  const projectPermissionsCache = shallowReactive(
    new Map<string, Set<ProjectPermission>>()
  );
  const workspaceLevelPermissionsMapByUserName = shallowReactive(
    new Map<string, Set<WorkspacePermission | ProjectPermission>>()
  );
  const roleStore = useRoleStore();

  const workspaceLevelPermissionsByUser = (
    user: User
  ): Set<WorkspacePermission | ProjectPermission> => {
    const cached = workspaceLevelPermissionsMapByUserName.get(user.name);
    if (cached) {
      return cached;
    }
    const roleStore = useRoleStore();
    const permissions = new Set(
      user.roles
        .map((role) => roleStore.getRoleByName(role))
        .flatMap(
          (role) =>
            (role ? role.permissions : []) as (
              | WorkspacePermission
              | ProjectPermission
            )[]
        )
    );
    workspaceLevelPermissionsMapByUserName.set(user.name, permissions);
    return permissions;
  };

  const roleListInProjectV1 = (project: ComposedProject, user: User) => {
    const key = `${user.name}@@${project.name}`;
    const cached = projectRoleListCache.get(key);
    if (cached) {
      return cached;
    }

    const groupStore = useUserGroupStore();
    const userInBinding = getUserEmailInBinding(user.email);

    const workspaceLevelProjectRoles = user.roles.filter(
      (role) => !PRESET_WORKSPACE_ROLES.includes(role)
    );

    const { iamPolicy } = project;
    const projectBindingRoles = iamPolicy.bindings
      .filter((binding) => {
        if (isBindingPolicyExpired(binding)) {
          return false;
        }
        for (const member of binding.members) {
          if (member === ALL_USERS_USER_EMAIL) {
            return true;
          }
          if (member === userInBinding) {
            return true;
          }

          if (member.startsWith(groupBindingPrefix)) {
            const group = groupStore.getGroupByIdentifier(member);
            if (!group) {
              continue;
            }

            return group.members.some(
              (m) => m.member === `${userNamePrefix}${user.email}`
            );
          }
        }
        return false;
      })
      .map((binding) => binding.role);

    const result = uniq([
      ...projectBindingRoles,
      ...workspaceLevelProjectRoles,
    ]);
    projectRoleListCache.set(key, result);
    return result;
  };

  const permissionsInProjectV1 = (project: ComposedProject, user: User) => {
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
  const invalidCacheByUser = (user: User) => {
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
