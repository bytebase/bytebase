import { defineStore } from "pinia";
import { computed, reactive, watchEffect } from "vue";
import { projectServiceClient } from "@/grpcweb";
import { ResourceId } from "@/types";
import {
  ProtectionRule,
  ProtectionRule_Target,
  ProtectionRules,
} from "@/types/proto/v1/project_service";
import { protectionRulesSuffix } from "./common";

export const useProjectProtectionRulesStore = defineStore(
  "projec_protection_rules",
  () => {
    const protectionRulesMapByName = reactive(
      new Map<ResourceId, ProtectionRule[]>()
    );

    const reset = () => {
      protectionRulesMapByName.clear();
    };

    // Actions
    const fetchProjectProtectionRules = async (projectName: string) => {
      const data = await projectServiceClient.getProjectProtectionRules({
        name: projectName + protectionRulesSuffix,
      });
      protectionRulesMapByName.set(data.name, data.rules);
      return data.rules;
    };
    const getProjectProtectionRules = (projectName: string) => {
      return (
        protectionRulesMapByName.get(projectName + protectionRulesSuffix) ?? []
      );
    };
    const getOrFetchProjectProtectionRules = async (projectName: string) => {
      if (protectionRulesMapByName.has(projectName + protectionRulesSuffix)) {
        return getProjectProtectionRules(projectName);
      }
      return await fetchProjectProtectionRules(projectName);
    };
    const updateProjectProtectionRules = async (
      protectionRules: ProtectionRules
    ) => {
      const data = await projectServiceClient.updateProjectProtectionRules({
        protectionRules,
      });
      protectionRulesMapByName.set(data.name, data.rules);
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
