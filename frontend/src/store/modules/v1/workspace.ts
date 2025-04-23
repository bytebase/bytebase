import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { workspaceServiceClient } from "@/grpcweb";
import { userNamePrefix } from "@/store/modules/v1/common";
import {
  groupBindingPrefix,
  PresetRoleType,
  ALL_USERS_USER_EMAIL,
} from "@/types";
import { IamPolicy, Binding } from "@/types/proto/v1/iam_policy";
import { roleListInIAM, getUserEmailListInBinding } from "@/utils";
import { extractUserId } from "./common";
import { extractGroupEmail } from "./group";

export const useWorkspaceV1Store = defineStore("workspace_v1", () => {
  const workspaceIamPolicy = ref<IamPolicy>(IamPolicy.fromPartial({}));

  // roleMapToUsers returns Map<roles/{role}, Set<users/{email}>>
  // the user could includes users/ALL_USERS_USER_EMAIL
  const roleMapToUsers = computed(() => {
    const map = new Map<string, Set<string>>();
    for (const binding of workspaceIamPolicy.value.bindings) {
      if (!map.has(binding.role)) {
        map.set(binding.role, new Set());
      }
      for (const email of getUserEmailListInBinding({
        binding,
        ignoreGroup: false,
      })) {
        map.get(binding.role)?.add(`${userNamePrefix}${email}`);
      }
    }
    return map;
  });

  // userMapToRoles returns Map<users/{email}, Set<roles/{role}>>
  // the user could includes users/ALL_USERS_USER_EMAIL
  const userMapToRoles = computed(() => {
    const map = new Map<string, Set<string>>();
    for (const binding of workspaceIamPolicy.value.bindings) {
      for (const email of getUserEmailListInBinding({
        binding,
        ignoreGroup: false,
      })) {
        const key = `${userNamePrefix}${email}`;
        if (!map.has(key)) {
          map.set(key, new Set());
        }
        map.get(key)?.add(binding.role);
      }
    }
    return map;
  });

  const fetchIamPolicy = async () => {
    const policy = await workspaceServiceClient.getIamPolicy({
      resource: "workspaces/-",
    });
    workspaceIamPolicy.value = policy;
  };

  const mergeBinding = ({
    member,
    roles,
    policy,
  }: {
    member: string;
    roles: string[];
    policy: IamPolicy;
  }) => {
    const newRolesSet = new Set(roles);
    const workspacePolicy = cloneDeep(policy);

    for (const binding of workspacePolicy.bindings) {
      const index = binding.members.findIndex((m) => m === member);
      if (!newRolesSet.has(binding.role)) {
        if (index >= 0) {
          binding.members.splice(index, 1);
        }
      } else {
        if (index < 0) {
          binding.members.push(member);
        }
      }

      newRolesSet.delete(binding.role);
    }

    for (const role of newRolesSet) {
      workspacePolicy.bindings.push(
        Binding.fromPartial({
          role,
          members: [member],
        })
      );
    }

    return workspacePolicy;
  };

  const patchIamPolicy = async (
    batchPatch: {
      member: string;
      roles: string[];
    }[]
  ) => {
    if (batchPatch.length === 0) {
      return;
    }
    let workspacePolicy = cloneDeep(workspaceIamPolicy.value);
    for (const patch of batchPatch) {
      workspacePolicy = mergeBinding({
        ...patch,
        policy: workspacePolicy,
      });
    }

    const policy = await workspaceServiceClient.setIamPolicy({
      resource: "workspaces/-",
      policy: workspacePolicy,
      etag: workspacePolicy.etag,
    });
    workspaceIamPolicy.value = policy;
  };

  const findRolesByMember = ({
    member,
    ignoreGroup,
  }: {
    member: string;
    ignoreGroup: boolean;
  }): string[] => {
    let email = member;
    if (member.startsWith(groupBindingPrefix)) {
      email = extractGroupEmail(member);
    } else {
      email = extractUserId(member);
    }
    return roleListInIAM({
      policy: workspaceIamPolicy.value,
      email,
      ignoreGroup,
    });
  };

  const getWorkspaceRolesByEmail = (email: string) => {
    const specificRoles =
      userMapToRoles.value.get(`${userNamePrefix}${email}`) ??
      // TODO(ed): not default member role
      new Set<string>([PresetRoleType.WORKSPACE_MEMBER]);
    if (userMapToRoles.value.has(`${userNamePrefix}${ALL_USERS_USER_EMAIL}`)) {
      for (const role of userMapToRoles.value.get(
        `${userNamePrefix}${ALL_USERS_USER_EMAIL}`
      )!) {
        specificRoles.add(role);
      }
    }
    return specificRoles;
  };

  return {
    workspaceIamPolicy,
    fetchIamPolicy,
    patchIamPolicy,
    findRolesByMember,
    roleMapToUsers,
    getWorkspaceRolesByEmail,
  };
});
