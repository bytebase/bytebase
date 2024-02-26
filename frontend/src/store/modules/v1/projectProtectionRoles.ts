import { defineStore } from "pinia";
import { computed, watchEffect } from "vue";
import { projectServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import {
  ProtectionRule,
  ProtectionRule_Target,
  ProtectionRules,
} from "@/types/proto/v1/project_service";
import { protectionRulesSuffix } from "./common";

type ProjectProtectionRuleCacheKey = [
  string /* project protection rules resource name */
];

export const useProjectProtectionRulesStore = defineStore(
  "project_protection_rules",
  () => {
    const cacheByName = useCache<
      ProjectProtectionRuleCacheKey,
      ProtectionRule[]
    >("bb.project-protection-rules.by-name");

    const reset = () => {
      cacheByName.clear();
    };

    // Actions
    const fetchProjectProtectionRules = async (projectName: string) => {
      const name = `${projectName}${protectionRulesSuffix}`;
      const data = await projectServiceClient.getProjectProtectionRules({
        name,
      });
      return data.rules;
    };
    const getProjectProtectionRules = (projectName: string) => {
      const name = projectName + protectionRulesSuffix;
      return cacheByName.getEntity([name]) ?? [];
    };
    const getOrFetchProjectProtectionRules = async (projectName: string) => {
      const name = `${projectName}${protectionRulesSuffix}`;
      const cachedRequest = cacheByName.getRequest([name]);
      if (cachedRequest) {
        return cachedRequest;
      }
      const cachedEntity = cacheByName.getEntity([name]);
      if (cachedEntity) {
        return cachedEntity;
      }
      const request = fetchProjectProtectionRules(projectName);
      cacheByName.setRequest([name], request);
      return request;
    };
    const updateProjectProtectionRules = async (
      protectionRules: ProtectionRules
    ) => {
      const data = await projectServiceClient.updateProjectProtectionRules({
        protectionRules,
      });
      cacheByName.setEntity([data.name], data.rules);
      return data.rules;
    };

    return {
      reset,
      fetchProjectProtectionRules,
      getProjectProtectionRules,
      getOrFetchProjectProtectionRules,
      updateProjectProtectionRules,
    };
  }
);

export const useProjectBranchProtectionRules = (projectName: string) => {
  const store = useProjectProtectionRulesStore();

  const branchProtectionRules = computed(() => {
    return store
      .getProjectProtectionRules(projectName)
      .filter((rule) => rule.target === ProtectionRule_Target.BRANCH);
  });

  watchEffect(() => {
    store.getOrFetchProjectProtectionRules(projectName);
  });

  return branchProtectionRules;
};
