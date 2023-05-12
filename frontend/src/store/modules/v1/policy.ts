import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { policyServiceClient } from "@/grpcweb";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
  policyTypeToJSON,
} from "@/types/proto/v1/org_policy_service";
import { MaybeRef, UNKNOWN_ID } from "@/types";
import { useCurrentUser } from "../auth";
import { State } from "@/types/proto/v1/common";
import { policyNamePrefix } from "@/store/modules/v1/common";

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
      showDeleted?: boolean;
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
    getPolicies({
      resourceType,
      resourceUID,
      policyType,
      showDeleted,
    }: {
      resourceType: PolicyResourceType;
      policyType: PolicyType;
      showDeleted?: boolean;
      resourceUID?: string;
    }) {
      const response: Policy[] = [];
      for (const [_, policy] of this.policyMapByName) {
        if (policy.resourceType != resourceType || policy.type != policyType) {
          continue;
        }
        if (!showDeleted && policy.state == State.DELETED) {
          continue;
        }
        if (resourceUID && policy.resourceUid != resourceUID) {
          continue;
        }
        response.push(policy);
      }
      return response;
    },
    async getOrFetchPolicyByParentAndType({
      parentPath,
      policyType,
    }: {
      parentPath: string;
      policyType: PolicyType;
    }) {
      const name = `${parentPath}/${policyNamePrefix}${policyTypeToJSON(
        policyType
      )}`;
      return this.getOrFetchPolicyByName(name);
    },
    async getOrFetchPolicyByName(name: string) {
      const cachedData = this.getPolicyByName(name);
      if (cachedData) {
        return cachedData;
      }
      try {
        const policy = await policyServiceClient.getPolicy({
          name: name.toLowerCase(),
        });
        this.policyMapByName.set(policy.name, policy);
        return policy;
      } catch {
        return;
      }
    },
    getPolicyByName(name: string) {
      return this.policyMapByName.get(name.toLowerCase());
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
    async upsertPolicy({
      parentPath,
      policy,
      updateMask,
    }: {
      parentPath: string;
      policy: Partial<Policy>;
      updateMask?: string[];
    }) {
      if (!policy.type) {
        throw new Error("policy type is required");
      }
      policy.name = `${parentPath}/${policyNamePrefix}${policyTypeToJSON(
        policy.type
      ).toLowerCase()}`;
      const updatedPolicy = await policyServiceClient.updatePolicy({
        policy,
        updateMask,
        allowMissing: true,
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

export const usePolicyListByResourceTypeAndPolicyType = (
  params: MaybeRef<{
    resourceType: PolicyResourceType;
    policyType: PolicyType;
    showDeleted: false;
  }>
) => {
  const store = usePolicyV1Store();
  const currentUser = useCurrentUser();
  watchEffect(() => {
    if (currentUser.value.id === UNKNOWN_ID) return;
    const { resourceType, policyType, showDeleted } = unref(params);

    store.fetchPolicies({ resourceType, policyType, showDeleted });
  });

  return computed(() => {
    const { resourceType, policyType, showDeleted } = unref(params);
    return store.getPolicies({ resourceType, policyType, showDeleted });
  });
};

export const usePolicyByParentAndType = (
  params: MaybeRef<{
    parentPath: string;
    policyType: PolicyType;
  }>
) => {
  const store = usePolicyV1Store();
  const currentUser = useCurrentUser();
  watchEffect(() => {
    if (currentUser.value.id === UNKNOWN_ID) return;
    const { policyType, parentPath } = unref(params);
    store.getOrFetchPolicyByParentAndType({
      parentPath,
      policyType,
    });
  });

  return computed(() => {
    const { parentPath, policyType } = unref(params);
    const res = store.getPolicyByName(
      `${parentPath}/${policyNamePrefix}${policyTypeToJSON(policyType)}`
    );
    return res;
  });
};
