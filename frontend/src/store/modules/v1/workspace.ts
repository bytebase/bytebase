import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";
import { workspaceServiceClient } from "@/grpcweb";
import { IamPolicy, Binding } from "@/types/proto/v1/iam_policy";

export const useWorkspaceV1Store = defineStore("workspace_v1", () => {
  const workspaceIamPolicy = ref<IamPolicy>(IamPolicy.fromPartial({}));

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

  const findRolesByMember = (member: string): string[] => {
    const roles: string[] = [];

    for (const binding of workspaceIamPolicy.value.bindings) {
      for (const m of binding.members) {
        if (m === member) {
          roles.push(binding.role);
          break;
        }
      }
    }

    return roles;
  };

  return {
    workspaceIamPolicy,
    fetchIamPolicy,
    patchIamPolicy,
    findRolesByMember,
  };
});
