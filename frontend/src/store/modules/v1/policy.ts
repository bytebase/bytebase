import { defineStore } from "pinia";
import { policyServiceClient } from "@/grpcweb";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";

interface PolicyState {
  policyMapByName: Map<string, Policy>;
}

const getPolicyParentByResourceType = (
  resourceType: PolicyResourceType
): string => {
  switch (resourceType) {
    case PolicyResourceType.PROJECT:
      return "projects/-";
    case PolicyResourceType.ENVIRONMENT:
      return "environments/-";
    case PolicyResourceType.INSTANCE:
      return "environments/-/instances/-";
    case PolicyResourceType.DATABASE:
      return "environments/-/instances/-/databases/-";
    default:
      return "";
  }
};

export const usePolicyV1Store = defineStore("policy_v1", {
  state: (): PolicyState => ({
    policyMapByName: new Map(),
  }),
  getters: {
    policyList(state) {
      return Array.from(state.policyMapByName.values());
    },
  },
  actions: {
    async fetchPolicies({
      resourceType,
      policyType,
      showDeleted = false,
    }: {
      resourceType: PolicyResourceType;
      policyType: PolicyType;
      showDeleted: boolean;
    }) {
      const { policies } = await policyServiceClient.listPolicies({
        parent: getPolicyParentByResourceType(resourceType),
        policyType,
        showDeleted,
      });
      for (const policy of policies) {
        this.policyMapByName.set(policy.name, policy);
      }
      return policies;
    },
    async getOrFetchPolicyByName(name: string) {
      const cachedData = this.getPolicyByName(name);
      if (cachedData) {
        return cachedData;
      }
      const policy = await policyServiceClient.getPolicy({
        name,
      });
      this.policyMapByName.set(policy.name, policy);
      return policy;
    },
    getPolicyByName(name: string) {
      return this.policyMapByName.get(name);
    },
    async createPolicy(parent: string, policy: Partial<Policy>) {
      const createdPolicy = await policyServiceClient.createPolicy({
        parent,
        policy,
      });
      this.policyMapByName.set(createdPolicy.name, createdPolicy);
      return createdPolicy;
    },
    async updatePolicy(updateMask: string[], policy: Partial<Policy>) {
      const updatedPolicy = await policyServiceClient.updatePolicy({
        policy,
        updateMask,
      });
      this.policyMapByName.set(updatedPolicy.name, updatedPolicy);
      return updatedPolicy;
    },
    async deletePolicy(name: string) {
      await policyServiceClient.deletePolicy({ name });
      this.policyMapByName.delete(name);
    },
  },
});
