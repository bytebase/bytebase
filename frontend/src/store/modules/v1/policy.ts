import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { computed, ref, unref, watchEffect } from "vue";
import { orgPolicyServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { policyNamePrefix } from "@/store/modules/v1/common";
import type { MaybeRef } from "@/types";
import { UNKNOWN_USER_NAME } from "@/types";
import type { Policy } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  PolicyResourceType,
  PolicyType,
  GetPolicyRequestSchema,
  ListPoliciesRequestSchema,
  PolicySchema,
  UpdatePolicyRequestSchema,
  DeletePolicyRequestSchema,
  RolloutPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { useCurrentUserV1 } from "./auth";

interface PolicyState {
  policyMapByName: Map<string, Policy>;
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
      return Array.from(state.policyMapByName.values());
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
        policyType: policyType,
        showDeleted,
      });
      const response =
        await orgPolicyServiceClientConnect.listPolicies(request);
      const policies = response.policies;
      for (const policy of policies) {
        this.policyMapByName.set(policy.name, policy);
      }
      return policies;
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
      for (const [_, policy] of this.policyMapByName) {
        if (policy.resourceType != resourceType || policy.type != policyType) {
          continue;
        }
        if (!showDeleted && !policy.enforce) {
          continue;
        }
        response.push(policy);
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
        `${parentPath}/${policyNamePrefix}${PolicyType[policyType]}`
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
          contextValues: createContextValues().set(silentContextKey, true),
        });
        this.policyMapByName.set(policy.name, policy);
        return policy;
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
        `${parentPath}/${policyNamePrefix}${PolicyType[policyType]}`
      );
      return this.getPolicyByName(name);
    },
    getPolicyByName(name: string) {
      const policy = this.policyMapByName.get(
        replacePolicyTypeNameToLowerCase(name)
      );
      return policy;
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
      const policyName = replacePolicyTypeNameToLowerCase(
        `${parentPath}/${policyNamePrefix}${PolicyType[policy.type]}`
      );
      const fullPolicy = create(PolicySchema, {
        name: policyName,
        inheritFromParent: policy.inheritFromParent ?? false,
        type: policy.type,
        resourceType:
          policy.resourceType ?? PolicyResourceType.RESOURCE_TYPE_UNSPECIFIED,
        enforce: policy.enforce ?? false,
        policy: policy.policy,
      });
      const request = create(UpdatePolicyRequestSchema, {
        policy: fullPolicy,
        updateMask: { paths: getUpdateMaskFromPolicyType(policy.type) },
        allowMissing: true,
      });
      const response =
        await orgPolicyServiceClientConnect.updatePolicy(request);
      this.policyMapByName.set(response.name, response);
      return response;
    },
    async deletePolicy(name: string) {
      const request = create(DeletePolicyRequestSchema, { name });
      await orgPolicyServiceClientConnect.deletePolicy(request);
      this.policyMapByName.delete(name);
    },
  },
});

const getUpdateMaskFromPolicyType = (policyType: PolicyType) => {
  switch (policyType) {
    case PolicyType.ROLLOUT_POLICY:
      return [PolicySchema.field.rolloutPolicy.name];
    case PolicyType.MASKING_EXCEPTION:
      return [PolicySchema.field.maskingExceptionPolicy.name];
    case PolicyType.MASKING_RULE:
      return [PolicySchema.field.maskingRulePolicy.name];
    case PolicyType.DATA_EXPORT:
      return [PolicySchema.field.exportDataPolicy.name];
    case PolicyType.DATA_QUERY:
      return [PolicySchema.field.queryDataPolicy.name];
    case PolicyType.DATA_SOURCE_QUERY:
      return [PolicySchema.field.dataSourceQueryPolicy.name];
    case PolicyType.DISABLE_COPY_DATA:
      return [PolicySchema.field.disableCopyDataPolicy.name];
    case PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
      return [PolicySchema.field.restrictIssueCreationForSqlReviewPolicy.name];
    case PolicyType.TAG:
      return [PolicySchema.field.tagPolicy.name];
    case PolicyType.POLICY_TYPE_UNSPECIFIED:
      throw new Error("unexpected POLICY_TYPE_UNSPECIFIED");
    default:
      throw new Error(`unknown policyType ${policyType satisfies never}`);
  }
};

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
      `${parentPath}/${policyNamePrefix}${PolicyType[policyType]}`
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
    `${parentPath}/${policyNamePrefix}${PolicyType[PolicyType.ROLLOUT_POLICY]}`
  );
  return create(PolicySchema, {
    name,
    inheritFromParent: false,
    type: PolicyType.ROLLOUT_POLICY,
    resourceType,
    enforce: true,
    policy: {
      case: "rolloutPolicy",
      value: create(RolloutPolicySchema, {
        automatic: false,
        roles: [],
        issueRoles: [],
      }),
    },
  });
};
