import { computed, ref, unref, watch } from "vue";
import { defineStore } from "pinia";

import { IamPolicy } from "@/types/proto/v1/project_service";
import { projectServiceClient } from "@/grpcweb";
import { MaybeRef } from "@/types";
import { useProjectStore } from "../project";
import { useProjectV1Store } from "./project";

export const useProjectIamPolicyStore = defineStore(
  "project-iam-policy",
  () => {
    const policyMap = ref(new Map<string, IamPolicy>());

    const fetchProjectIamPolicy = async (project: string) => {
      const policy = await projectServiceClient.getIamPolicy({
        project,
      });
      policyMap.value.set(project, policy);
    };

    const updateProjectIamPolicy = async (
      project: string,
      policy: IamPolicy
    ) => {
      const updated = await projectServiceClient.setIamPolicy({
        project,
        policy,
      });
      policyMap.value.set(project, updated);

      // legacy project API support
      // re-fetch the legacy project entity to refresh its `memberList`
      const projectEntity = await useProjectV1Store().getOrFetchProjectByName(
        project
      );
      await useProjectStore().fetchProjectById(parseInt(projectEntity.uid, 10));
    };

    return {
      policyMap,
      fetchProjectIamPolicy,
      updateProjectIamPolicy,
    };
  }
);

export const useProjectIamPolicy = (project: MaybeRef<string>) => {
  const store = useProjectIamPolicyStore();
  const ready = ref(false);
  watch(
    () => unref(project),
    (project) => {
      ready.value = false;
      store.fetchProjectIamPolicy(project).finally(() => {
        ready.value = true;
      });
    },
    { immediate: true }
  );
  const policy = computed(() => {
    return store.policyMap.get(unref(project)) ?? IamPolicy.fromJSON({});
  });
  return { policy, ready };
};
