import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { orgPolicyServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { policyNamePrefix } from "@/store/modules/v1/common";
import type { MaybeRef } from "@/types";
import { UNKNOWN_USER_NAME } from "@/types";
import { Policy, PolicyResourceType, PolicyType, policyTypeToJSON } from "@/types/proto/v1/org_policy_service";
import type { Policy as NewPolicy } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  GetPolicyRequestSchema,
  ListPoliciesRequestSchema,
  PolicySchema,
  UpdatePolicyRequestSchema,
  DeletePolicyRequestSchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  convertNewPolicyToOld,
  convertOldPolicyToNew,
  convertOldPolicyTypeToNew,
  convertOldPolicyResourceTypeToNew,
} from "@/utils/v1/org-policy-conversions";
import { defineStore } from "pinia";
import { computed, ref, unref, watchEffect } from "vue";
import { useCurrentUserV1 } from "./auth";

interface PolicyState {
  policyMapByName: Map<string, NewPolicy>;
}

const replacePolicyTypeNameToLowerCase = (name: string) => {
  const pattern = /(^|\/)policies\/([^/]+)($|\/)/;
  const replaced = name.replace(
    pattern,
    (_: string, left: string, policyType: string, right: string) => {
      return `${left}policies/${policyType.toLowerCase()}${right}`;
    }
  );
  if (replaced.startsWith("/")) {
    return replaced.slice(1);
  }
  return replaced;
};

const getPolicyParentByResourceType = (
  resourceType: PolicyResourceType
): string => {
  switch (resourceType) {
    case PolicyResourceType.PROJECT:
      return "projects/-";
    case PolicyResourceType.ENVIRONMENT:
      return "environments/-";
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
      return Array.from(state.policyMapByName.values()).map(convertNewPolicyToOld);
    },
  },
  actions: {
    async fetchPolicies({
      resourceType,
      policyType,
      parent,
      showDeleted = false,
    }: {
      resourceType: PolicyResourceType;
      policyType?: PolicyType;
      parent?: string;
      showDeleted?: boolean;
    }) {
      const request = create(ListPoliciesRequestSchema, {
        parent: parent ?? getPolicyParentByResourceType(resourceType),
        policyType: policyType ? convertOldPolicyTypeToNew(policyType) : undefined,
        showDeleted,
      });
      const response = await orgPolicyServiceClientConnect.listPolicies(request);
      const policies = response.policies;
      for (const policy of policies) {
        this.policyMapByName.set(policy.name, policy);
      }
      return policies.map(convertNewPolicyToOld);
    },
    getPolicies({
      resourceType,
      policyType,
      showDeleted,
    }: {
      resourceType: PolicyResourceType;
      policyType: PolicyType;
      showDeleted?: boolean;
    }) {
      const response: Policy[] = [];
      const newResourceType = convertOldPolicyResourceTypeToNew(resourceType);
      const newPolicyType = convertOldPolicyTypeToNew(policyType);
      for (const [_, policy] of this.policyMapByName) {
        if (policy.resourceType != newResourceType || policy.type != newPolicyType) {
          continue;
        }
        if (!showDeleted && !policy.enforce) {
          continue;
        }
        response.push(convertNewPolicyToOld(policy));
      }
      return response;
    },
    async getOrFetchPolicyByParentAndType({
      parentPath,
      policyType,
      refresh,
    }: {
      parentPath: string;
      policyType: PolicyType;
      refresh?: boolean;
    }) {
      const name = replacePolicyTypeNameToLowerCase(
        `${parentPath}/${policyNamePrefix}${policyTypeToJSON(policyType)}`
      );
      return this.getOrFetchPolicyByName(name, refresh);
    },
    async getOrFetchPolicyByName(name: string, refresh = false) {
      const cachedData = this.getPolicyByName(
        replacePolicyTypeNameToLowerCase(name)
      );
      if (cachedData && !refresh) {
        return cachedData;
      }
      try {
        const request = create(GetPolicyRequestSchema, { name });
        const policy = await orgPolicyServiceClientConnect.getPolicy(request, {
          contextValues: createContextValues().set(silentContextKey, true)
        });
        this.policyMapByName.set(policy.name, policy);
        return convertNewPolicyToOld(policy);
      } catch (error) {
        if (error instanceof ConnectError && error.code === Code.NotFound) {
          // To prevent unnecessary requests, cache empty policies if not found.
          const emptyPolicy = create(PolicySchema, { name });
          this.policyMapByName.set(name, emptyPolicy);
        }
      }
    },
    getPolicyByParentAndType({
      parentPath,
      policyType,
    }: {
      parentPath: string;
      policyType: PolicyType;
    }) {
      const name = replacePolicyTypeNameToLowerCase(
        `${parentPath}/${policyNamePrefix}${policyTypeToJSON(policyType)}`
      );
      return this.getPolicyByName(name);
    },
    getPolicyByName(name: string) {
      const policy = this.policyMapByName.get(replacePolicyTypeNameToLowerCase(name));
      return policy ? convertNewPolicyToOld(policy) : undefined;
    },
    async upsertPolicy({
      parentPath,
      policy,
    }: {
      parentPath: string;
      policy: Partial<Policy>;
    }) {
      if (!policy.type) {
        throw new Error("policy type is required");
      }
      policy.name = replacePolicyTypeNameToLowerCase(
        `${parentPath}/${policyNamePrefix}${policyTypeToJSON(policy.type)}`
      );
      const fullPolicy = Policy.fromPartial(policy);
      const newPolicy = convertOldPolicyToNew(fullPolicy);
      const request = create(UpdatePolicyRequestSchema, {
        policy: newPolicy,
        updateMask: { paths: ["payload"] },
        allowMissing: true,
      });
      const response = await orgPolicyServiceClientConnect.updatePolicy(request);
      const updatedPolicy = convertNewPolicyToOld(response);
      this.policyMapByName.set(response.name, response);
      return updatedPolicy;
    },
    async deletePolicy(name: string) {
      const request = create(DeletePolicyRequestSchema, { name });
      await orgPolicyServiceClientConnect.deletePolicy(request);
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
  const currentUserV1 = useCurrentUserV1();
  watchEffect(() => {
    if (currentUserV1.value.name === UNKNOWN_USER_NAME) return;
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
  const currentUserV1 = useCurrentUserV1();
  const ready = ref(false);

  watchEffect(() => {
    if (currentUserV1.value.name === UNKNOWN_USER_NAME) return;
    const { policyType, parentPath } = unref(params);
    store
      .getOrFetchPolicyByParentAndType({
        parentPath,
        policyType,
      })
      .finally(() => (ready.value = true));
  });

  const policy = computed(() => {
    const { parentPath, policyType } = unref(params);
    const name = replacePolicyTypeNameToLowerCase(
      `${parentPath}/${policyNamePrefix}${policyTypeToJSON(policyType)}`
    );
    const res = store.getPolicyByName(name);
    return res;
  });
  return {
    policy,
    ready,
  };
};

// Default RolloutPolicy payload is somehow strict to prevent auto rollout

export const getEmptyRolloutPolicy = (
  parentPath: string,
  resourceType: PolicyResourceType
): Policy => {
  const name = replacePolicyTypeNameToLowerCase(
    `${parentPath}/${policyNamePrefix}${policyTypeToJSON(
      PolicyType.ROLLOUT_POLICY
    )}`
  );
  return Policy.fromPartial({
    name,
    inheritFromParent: false,
    type: PolicyType.ROLLOUT_POLICY,
    resourceType,
    enforce: true,
    rolloutPolicy: {
      automatic: false,
      roles: [],
      issueRoles: [],
    },
  });
};
