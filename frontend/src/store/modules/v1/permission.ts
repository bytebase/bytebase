import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, shallowReactive } from "vue";
import { type Permission, PRESET_WORKSPACE_ROLES } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { type User } from "@/types/proto-es/v1/user_service_pb";
import { bindingListInIAM } from "@/utils";
import { useRoleStore } from "../role";
import { useCurrentUserV1 } from "./auth";
import { useProjectIamPolicyStore } from "./projectIamPolicy";
import { useWorkspaceV1Store } from "./workspace";

export const usePermissionStore = defineStore("permission", () => {
  const projectRoleListCache = shallowReactive(new Map<string, string[]>());
  const projectPermissionsCache = shallowReactive(
    new Map<string, Set<Permission>>()
  );
  const roleStore = useRoleStore();
  const currentUser = useCurrentUserV1();
  const workspaceStore = useWorkspaceV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();

  const currentRolesInWorkspace = computed(() => {
    return workspaceStore.getWorkspaceRolesByEmail(currentUser.value.email);
  });

  const currentPermissions = computed(() => {
    return new Set(
      [...currentRolesInWorkspace.value]
        .map((role) => roleStore.getRoleByName(role))
        .flatMap((role) => (role ? role.permissions : []) as Permission[])
    );
  });

  const currentRoleListInProjectV1 = (project: Project) => {
    const key = `${currentUser.value.name}@@${project.name}`;
    const cached = projectRoleListCache.get(key);
    if (cached) {
      return cached;
    }

    const workspaceLevelProjectRoles = [
      ...currentRolesInWorkspace.value,
    ].filter((role) => !PRESET_WORKSPACE_ROLES.includes(role));

    const iamPolicy = projectIamPolicyStore.getProjectIamPolicy(project.name);
    const projectBindings = bindingListInIAM({
      policy: iamPolicy,
      email: currentUser.value.email,
      ignoreGroup: false,
    });

    const result = uniq([
      ...projectBindings.map((binding) => binding.role),
      ...workspaceLevelProjectRoles,
    ]);
    if (projectBindings.length > 0) {
      // Note: do not set cache if no project IAM policy.
      // The project IAM policy may not initialized at this time.
      projectRoleListCache.set(key, result);
    }
    return result;
  };

  const currentPermissionsInProjectV1 = (project: Project): Set<Permission> => {
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
    if (permissions.size > 0) {
      // Note: do not set cache if no project IAM policy.
      // The project IAM policy may not initialized at this time.
      projectPermissionsCache.set(key, permissions);
    }
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
    currentRoleListInProjectV1,
    currentPermissionsInProjectV1,
    invalidCacheByProject,
    invalidCacheByUser,
  };
});
