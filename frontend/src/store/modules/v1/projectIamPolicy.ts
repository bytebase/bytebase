import { computed, onMounted, ref, unref, watch } from "vue";
import { defineStore } from "pinia";

import { IamPolicy } from "@/types/proto/v1/project_service";
import { projectServiceClient } from "@/grpcweb";
import { Database, MaybeRef } from "@/types";
import { useProjectStore } from "../project";
import { useProjectV1Store } from "./project";
import { useCurrentUserV1 } from "../auth";

export const useProjectIamPolicyStore = defineStore(
  "project-iam-policy",
  () => {
    const policyMap = ref(new Map<string, IamPolicy>());
    const requestCache = new Map<string, Promise<IamPolicy>>();

    const fetchProjectIamPolicy = async (project: string) => {
      const cache = requestCache.get(project);
      if (cache) {
        return cache;
      }
      const request = projectServiceClient
        .getIamPolicy({
          project,
        })
        .then((policy) => {
          policyMap.value.set(project, policy);
          return policy;
        });
      requestCache.set(project, request);
      return request;
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

    const getProjectIamPolicy = (project: string) => {
      return policyMap.value.get(project) ?? IamPolicy.fromJSON({});
    };

    const getOrFetchProjectIamPolicy = async (project: string) => {
      if (!policyMap.value.has(project)) {
        await fetchProjectIamPolicy(project);
      }
      return getProjectIamPolicy(project);
    };

    return {
      policyMap,
      getProjectIamPolicy,
      fetchProjectIamPolicy,
      getOrFetchProjectIamPolicy,
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

export const useCurrentUserIamPolicy = () => {
  const iamPolicyStore = useProjectIamPolicyStore();
  const projectStore = useProjectV1Store();
  const currentUser = useCurrentUserV1();

  onMounted(async () => {
    for (const project of projectStore.projectList) {
      await iamPolicyStore.getOrFetchProjectIamPolicy(project.name);
    }
  });

  const allowToChangeDatabaseOfProject = (projectName: string) => {
    const policy = iamPolicyStore.policyMap.get(projectName);
    if (!policy) {
      return false;
    }
    for (const binding of policy.bindings) {
      if (
        (binding.role === "roles/OWNER" ||
          binding.role === "roles/DEVELOPER") &&
        binding.members.find(
          (member) => member === `user:${currentUser.value.email}`
        )
      ) {
        return true;
      }
    }
    return false;
  };

  const allowToQueryDatabase = async (database: Database) => {
    const policy = await iamPolicyStore.getOrFetchProjectIamPolicy(
      `projects/${database.project.resourceId}`
    );
    console.log("!", policy, database.project.resourceId);
    if (!policy) {
      return false;
    }
    for (const binding of policy.bindings) {
      if (
        binding.role === "roles/OWNER" &&
        binding.members.find(
          (member) => member === `user:${currentUser.value.email}`
        )
      ) {
        return true;
      }
      if (
        binding.role === "roles/QUERIER" &&
        binding.members.find(
          (member) => member === `user:${currentUser.value.email}`
        )
      ) {
        const expressionList = binding.condition?.expression.split(" && ");
        if (expressionList && expressionList.length > 0) {
          for (const expression of expressionList) {
            const fields = expression.split(" ");
            if (fields[0] === "resource.database") {
              for (const url of JSON.parse(fields[2])) {
                const value = url.split("/");
                const instanceName = value[1] || "";
                const databaseName = value[3] || "";
                if (
                  database.instance.resourceId === instanceName &&
                  database.name === databaseName
                ) {
                  return true;
                }
              }
            }
          }
        } else {
          return true;
        }
      }
    }
    return false;
  };

  return {
    allowToChangeDatabaseOfProject,
    allowToQueryDatabase,
  };
};
