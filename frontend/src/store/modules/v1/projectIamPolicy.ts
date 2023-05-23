import { computed, ref, unref, watch, watchEffect } from "vue";
import { defineStore } from "pinia";

import { IamPolicy } from "@/types/proto/v1/project_service";
import { projectServiceClient } from "@/grpcweb";
import { Database, MaybeRef, PresetRoleType } from "@/types";
import { useLegacyProjectStore } from "../project";
import { useProjectV1Store } from "./project";
import { useCurrentUserV1 } from "../auth";
import {
  getDatabaseNameById,
  hasWorkspacePermissionV1,
  isDeveloperOfProjectV1,
  isMemberOfProjectV1,
  isOwnerOfProjectV1,
  parseConditionExpressionString,
} from "@/utils";

export const useProjectIamPolicyStore = defineStore(
  "project-iam-policy",
  () => {
    const policyMap = ref(new Map<string, IamPolicy>());
    const requestCache = new Map<string, Promise<IamPolicy>>();

    const fetchProjectIamPolicy = async (
      project: string,
      skipCache = false
    ) => {
      if (!skipCache) {
        const cache = requestCache.get(project);
        if (cache) {
          return cache;
        }
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

    const batchFetchIamPolicy = async (projectList: string[]) => {
      const response = await projectServiceClient.batchGetIamPolicy({
        scope: "projects/-",
        names: projectList,
      });
      response.policyResults.forEach(({ policy, project }) => {
        if (policy) {
          policyMap.value.set(project, policy);
        }
      });
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

      const projectEntity = await useProjectV1Store().getOrFetchProjectByName(
        project
      );
      // legacy project API support
      // re-fetch the legacy project entity to refresh its `memberList`
      await useLegacyProjectStore().fetchProjectById(
        parseInt(projectEntity.uid, 10)
      );
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

    const batchGetOrFetchProjectIamPolicy = async (projectList: string[]) => {
      // BatchFetch policies that missing in the local map.
      const missingProjectList = projectList.filter(
        (project) => !policyMap.value.has(project)
      );
      if (missingProjectList.length > 0) {
        await batchFetchIamPolicy(missingProjectList);
      }
      return projectList.map(getProjectIamPolicy);
    };

    return {
      policyMap,
      getProjectIamPolicy,
      fetchProjectIamPolicy,
      getOrFetchProjectIamPolicy,
      batchGetOrFetchProjectIamPolicy,
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

  watchEffect(() => {
    // Fetch all project iam policies.
    Promise.all(
      projectStore.projectList.map((project) =>
        iamPolicyStore.getOrFetchProjectIamPolicy(project.name)
      )
    );
  });

  // hasWorkspaceSuperPrivilege checks whether the current user has the super privilege to access all databases. AKA. Owners and DBAs
  const hasWorkspaceSuperPrivilege = hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-access-control",
    currentUser.value.userRole
  );

  const isMemberOfProject = (projectName: string) => {
    if (hasWorkspaceSuperPrivilege) {
      return true;
    }

    const policy = iamPolicyStore.policyMap.get(projectName);
    if (!policy) {
      return false;
    }
    return isMemberOfProjectV1(policy, currentUser.value);
  };

  const isProjectOwnerOrDeveloper = (projectName: string) => {
    if (hasWorkspaceSuperPrivilege) {
      return true;
    }

    const policy = iamPolicyStore.policyMap.get(projectName);
    if (!policy) {
      return false;
    }
    return (
      isOwnerOfProjectV1(policy, currentUser.value) ||
      isDeveloperOfProjectV1(policy, currentUser.value)
    );
  };

  const allowToChangeDatabaseOfProject = (projectName: string) => {
    return isProjectOwnerOrDeveloper(projectName);
  };

  const allowToQueryDatabase = (database: Database) => {
    if (hasWorkspaceSuperPrivilege) {
      return true;
    }

    const policy = iamPolicyStore.getProjectIamPolicy(
      `projects/${database.project.resourceId}`
    );
    if (!policy) {
      return false;
    }
    const iamPolicyCheckResult = policy.bindings.map((binding) => {
      if (
        binding.role === PresetRoleType.OWNER &&
        binding.members.find(
          (member) => member === `user:${currentUser.value.email}`
        )
      ) {
        return true;
      }
      if (
        binding.role === PresetRoleType.QUERIER &&
        binding.members.find(
          (member) => member === `user:${currentUser.value.email}`
        )
      ) {
        const conditionExpression = parseConditionExpressionString(
          binding.condition?.expression || ""
        );
        if (conditionExpression.databases) {
          const databaseResourceName = getDatabaseNameById(database.id);
          return conditionExpression.databases.includes(databaseResourceName);
        } else {
          return true;
        }
      }
      return false;
    });
    // If one of the binding is true, then the user is allowed to query the database.
    if (iamPolicyCheckResult.includes(true)) {
      return true;
    }

    return false;
  };

  const allowToExportDatabase = (database: Database) => {
    if (hasWorkspaceSuperPrivilege) {
      return true;
    }

    const policy = iamPolicyStore.getProjectIamPolicy(
      `projects/${database.project.resourceId}`
    );
    if (!policy) {
      return false;
    }
    const iamPolicyCheckResult = policy.bindings.map((binding) => {
      if (
        binding.role === PresetRoleType.OWNER &&
        binding.members.find(
          (member) => member === `user:${currentUser.value.email}`
        )
      ) {
        return true;
      }
      if (
        binding.role === PresetRoleType.EXPORTER &&
        binding.members.find(
          (member) => member === `user:${currentUser.value.email}`
        )
      ) {
        const conditionExpression = parseConditionExpressionString(
          binding.condition?.expression || ""
        );
        if (conditionExpression.databases) {
          const databaseResourceName = getDatabaseNameById(database.id);
          return conditionExpression.databases.includes(databaseResourceName);
        } else {
          return true;
        }
      }
    });
    // If one of the binding is true, then the user is allowed to export the database.
    if (iamPolicyCheckResult.includes(true)) {
      return true;
    }

    return false;
  };

  return {
    isMemberOfProject,
    isProjectOwnerOrDeveloper,
    allowToChangeDatabaseOfProject,
    allowToQueryDatabase,
    allowToExportDatabase,
  };
};
