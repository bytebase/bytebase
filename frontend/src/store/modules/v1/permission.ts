import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { shallowReactive, computed } from "vue";
import {
  PresetRoleType,
  PRESET_WORKSPACE_ROLES,
  type ComposedProject,
  type Permission,
} from "@/types";
import { type User } from "@/types/proto/v1/auth_service";
import { roleListInIAM } from "@/utils";
import { useCurrentUserV1 } from "../auth";
import { useRoleStore } from "../role";
import { useWorkspaceV1Store } from "./workspace";

export const usePermissionStore = defineStore("permission", () => {
  const projectRoleListCache = shallowReactive(new Map<string, string[]>());
  const projectPermissionsCache = shallowReactive(
    new Map<string, Set<Permission>>()
  );
  const workspaceLevelPermissionsMapByUserName = shallowReactive(
    new Map<string, Set<Permission>>()
  );
  const roleStore = useRoleStore();
  const currentUser = useCurrentUserV1();
  const workspaceStore = useWorkspaceV1Store();

  const currentRolesInWorkspace = computed(() => {
    return workspaceStore.getWorkspaceRolesByEmail(currentUser.value.email);
  });

  const onlyWorkspaceMember = computed(
    () =>
      currentRolesInWorkspace.value.size === 0 ||
      (currentRolesInWorkspace.value.size === 1 &&
        currentRolesInWorkspace.value.has(PresetRoleType.WORKSPACE_MEMBER))
  );

  const currentPermissions = computed(() => {
    return new Set(
      [...currentRolesInWorkspace.value]
        .map((role) => roleStore.getRoleByName(role))
        .flatMap((role) => (role ? role.permissions : []) as Permission[])
    );
  });

  const currentRoleListInProjectV1 = (project: ComposedProject) => {
    const key = `${currentUser.value.name}@@${project.name}`;
    const cached = projectRoleListCache.get(key);
    if (cached) {
      return cached;
    }

    const workspaceLevelProjectRoles = [
      ...currentRolesInWorkspace.value,
    ].filter((role) => !PRESET_WORKSPACE_ROLES.includes(role));

    const { iamPolicy } = project;
    const projectBindingRoles = roleListInIAM({
      policy: iamPolicy,
      email: currentUser.value.email,
      ignoreGroup: false,
    });

    const result = uniq([
      ...projectBindingRoles,
      ...workspaceLevelProjectRoles,
    ]);
    projectRoleListCache.set(key, result);
    return result;
  };

  const currentPermissionsInProjectV1 = (
    project: ComposedProject
  ): Set<Permission> => {
    const key = `${currentUser.value.name}@@${project.name}`;
    const cached = projectPermissionsCache.get(key);
    if (cached) {
      return cached;
    }

    const roles = currentRoleListInProjectV1(project);
    const permissions = new Set(
      roles
        .map((role) => roleStore.getRoleByName(role))
        .flatMap((role) => (role ? role.permissions : []) as Permission[])
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
    currentPermissions,
    currentRolesInWorkspace,
    onlyWorkspaceMember,
    currentRoleListInProjectV1,
    currentPermissionsInProjectV1,
    invalidCacheByProject,
    invalidCacheByUser,
  };
});
