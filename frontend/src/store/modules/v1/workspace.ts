import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { create } from "@bufbuild/protobuf";
import { workspaceServiceClientConnect } from "@/grpcweb";
import { userNamePrefix } from "@/store/modules/v1/common";
import { groupBindingPrefix, ALL_USERS_USER_EMAIL } from "@/types";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import {
  IamPolicySchema,
  BindingSchema,
  GetIamPolicyRequestSchema,
  SetIamPolicyRequestSchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import { bindingListInIAM, getUserEmailListInBinding } from "@/utils";
import { extractUserId } from "./common";
import { extractGroupEmail } from "./group";
import { convertNewIamPolicyToOld } from "@/utils/v1/iam-conversions";

export const useWorkspaceV1Store = defineStore("workspace_v1", () => {
  // Internal state uses proto-es types
  const _workspaceIamPolicy = ref<IamPolicy>(create(IamPolicySchema, {}));
  
  // Computed property that provides old proto type for compatibility
  const workspaceIamPolicy = computed(() => {
    return convertNewIamPolicyToOld(_workspaceIamPolicy.value);
  });

  // roleMapToUsers returns Map<roles/{role}, Set<users/{email}>>
  // the user could includes users/ALL_USERS_USER_EMAIL
  const roleMapToUsers = computed(() => {
    const map = new Map<string, Set<string>>();
    for (const binding of _workspaceIamPolicy.value.bindings) {
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
    for (const binding of _workspaceIamPolicy.value.bindings) {
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
    const request = create(GetIamPolicyRequestSchema, {
      resource: "workspaces/-",
    });
    const policy = await workspaceServiceClientConnect.getIamPolicy(request);
    _workspaceIamPolicy.value = policy;
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
        create(BindingSchema, {
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
    let workspacePolicy = cloneDeep(_workspaceIamPolicy.value);
    for (const patch of batchPatch) {
      workspacePolicy = mergeBinding({
        ...patch,
        policy: workspacePolicy,
      });
    }

    const request = create(SetIamPolicyRequestSchema, {
      resource: "workspaces/-",
      policy: workspacePolicy,
      etag: workspacePolicy.etag,
    });
    const policy = await workspaceServiceClientConnect.setIamPolicy(request);
    _workspaceIamPolicy.value = policy;
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
    return bindingListInIAM({
      policy: _workspaceIamPolicy.value,
      email,
      ignoreGroup,
    }).map((binding) => binding.role);
  };

  const getWorkspaceRolesByEmail = (email: string) => {
    const specificRoles =
      userMapToRoles.value.get(`${userNamePrefix}${email}`) ??
      new Set<string>([]);
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
