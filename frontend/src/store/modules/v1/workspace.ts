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

  const patchIamPolicy = async ({
    member,
    roles,
  }: {
    member: string;
    roles: string[];
  }) => {
    const newRolesSet = new Set(roles);
    const workspacePolicy = cloneDeep(workspaceIamPolicy.value);

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

    const policy = await workspaceServiceClient.setIamPolicy({
      resource: "workspaces/-",
      policy: workspacePolicy,
      etag: workspacePolicy.etag,
    });
    workspaceIamPolicy.value = policy;
  };

  return {
    workspaceIamPolicy,
    fetchIamPolicy,
    patchIamPolicy,
  };
});
