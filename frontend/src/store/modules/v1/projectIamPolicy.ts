import { isUndefined, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref, unref, watch } from "vue";
import { projectServiceClient } from "@/grpcweb";
import type {
  ComposedDatabase,
  ComposedProject,
  MaybeRef,
  Permission,
} from "@/types";
import type { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import type { User } from "@/types/proto/v1/auth_service";
import { IamPolicy } from "@/types/proto/v1/iam_policy";
import { getUserEmailListInBinding } from "@/utils";
import { convertFromExpr } from "@/utils/issue/cel";
import { useCurrentUserV1 } from "../auth";
import { useRoleStore } from "../role";
import { usePermissionStore } from "./permission";

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
          resource: project,
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
      policy.bindings.forEach((binding) => {
        if (binding.members) {
          binding.members = uniq(binding.members);
        }
      });
      const updated = await projectServiceClient.setIamPolicy({
        resource: project,
        policy,
        etag: policy.etag,
      });
      policyMap.value.set(project, updated);

      usePermissionStore().invalidCacheByProject(project);
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

    const batchGetOrFetchProjectIamPolicy = async (
      projectList: string[],
      skipCache = false
    ) => {
      if (skipCache) {
        await batchFetchIamPolicy(projectList);
      } else {
        // BatchFetch policies that missing in the local map.
        const missingProjectList = projectList.filter(
          (project) => !policyMap.value.has(project)
        );
        if (missingProjectList.length > 0) {
          await batchFetchIamPolicy(missingProjectList);
        }
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

const checkProjectIAMPolicyWithExpr = (
  user: User,
  project: ComposedProject,
  permission: Permission,
  bindingExprCheck: (expr?: Expr) => boolean
): boolean => {
  const roleStore = useRoleStore();
  // Check if the user has the permission.
  for (const binding of project.iamPolicy.bindings) {
    // If the user is not in the binding, then skip.
    const userEmailList = getUserEmailListInBinding({
      binding,
      ignoreGroup: false,
    });
    if (!userEmailList.includes(user.email)) {
      continue;
    }
    // If the role does not have the permission, then skip.
    const permissions =
      roleStore.getRoleByName(binding.role)?.permissions || [];
    if (!permissions.includes(permission)) {
      continue;
    }
    // If binding expr check passes, then return true.
    if (bindingExprCheck(binding.parsedExpr?.expr)) {
      return true;
    }
  }

  return false;
};

export const checkQuerierPermission = (
  database: ComposedDatabase,
  schema?: string,
  table?: string
) => {
  return checkProjectIAMPolicyWithExpr(
    useCurrentUserV1().value,
    database.projectEntity,
    "bb.databases.query",
    (expr?: Expr): boolean => {
      // If no condition is set, then return true.
      if (!expr) {
        return true;
      }

      const conditionExpr = convertFromExpr(expr);
      // Check if the condition is expired.
      if (
        conditionExpr.expiredTime &&
        new Date(conditionExpr.expiredTime).getTime() < Date.now()
      ) {
        return false;
      }
      // Check if the condition is valid for the database.
      if (
        conditionExpr.databaseResources &&
        conditionExpr.databaseResources.length > 0
      ) {
        for (const databaseResource of conditionExpr.databaseResources) {
          if (databaseResource.databaseName === database.name) {
            if (isUndefined(schema) && isUndefined(table)) {
              return true;
            } else {
              if (
                isUndefined(databaseResource.schema) ||
                (isUndefined(databaseResource.schema) &&
                  isUndefined(databaseResource.table)) ||
                (databaseResource.schema === schema &&
                  databaseResource.table === table)
              ) {
                return true;
              }
            }
          }
        }
        return false;
      } else {
        return true;
      }
    }
  );
};
