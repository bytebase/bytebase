import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { workspaceServiceClient } from "@/grpcweb";
import { groupBindingPrefix, PresetRoleType } from "@/types";
import { IamPolicy, Binding } from "@/types/proto/v1/iam_policy";
import { roleListInIAM, getUserEmailListInBinding } from "@/utils";
import { extractUserEmail } from "../user";
import { extractGroupEmail } from "./group";

export const useWorkspaceV1Store = defineStore("workspace_v1", () => {
  const workspaceIamPolicy = ref<IamPolicy>(IamPolicy.fromPartial({}));

  const emailMapToRoles = computed(() => {
    const map = new Map<string, Set<string>>(); // Map<email, role list>
    for (const binding of workspaceIamPolicy.value.bindings) {
      for (const email of getUserEmailListInBinding({
        binding,
        ignoreGroup: false,
      })) {
        if (!map.has(email)) {
          map.set(email, new Set());
        }
        map.get(email)?.add(binding.role);
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
      email = extractUserEmail(member);
    }
    return roleListInIAM({
      policy: workspaceIamPolicy.value,
      email,
      ignoreGroup,
    });
  };

  const getWorkspaceRolesByEmail = (email: string) => {
    return (
      emailMapToRoles.value.get(email) ??
      new Set<string>([PresetRoleType.WORKSPACE_MEMBER])
    );
  };

  return {
    workspaceIamPolicy,
    fetchIamPolicy,
    patchIamPolicy,
    findRolesByMember,
    emailMapToRoles,
    getWorkspaceRolesByEmail,
  };
});
