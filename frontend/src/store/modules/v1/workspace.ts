import { defineStore } from "pinia";
import { computed, ref, watchEffect } from "vue";
import { workspaceServiceClient } from "@/grpcweb";
import { IamPolicy } from "@/types/proto/v1/iam_policy";

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
    const policy = await workspaceServiceClient.patchIamPolicy({
      resource: "workspaces/-",
      member,
      roles,
    });
    workspaceIamPolicy.value = policy;
  };

  return {
    workspaceIamPolicy,
    fetchIamPolicy,
    patchIamPolicy,
  };
});

export const useWorkspaceIamPolicy = () => {
  const store = useWorkspaceV1Store();

  watchEffect(() => {
    store.fetchIamPolicy();
  });

  return computed(() => store.workspaceIamPolicy);
};
