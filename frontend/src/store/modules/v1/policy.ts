import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watchEffect } from "vue";
import { orgPolicyServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { policyNamePrefix } from "@/store/modules/v1/common";
import type { MaybeRef } from "@/types";
import { UNKNOWN_USER_NAME } from "@/types";
import type {
  Policy,
  QueryDataPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  DeletePolicyRequestSchema,
  GetPolicyRequestSchema,
  ListPoliciesRequestSchema,
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  QueryDataPolicySchema,
  RolloutPolicySchema,
  UpdatePolicyRequestSchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { useCurrentUserV1 } from "./auth";

interface PolicyState {
  policyMapByName: Map<string, Policy>;
}

export const DEFAULT_MAX_RESULT_SIZE_IN_MB = 100;

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

export const usePolicyV1Store = defineStore("policy_v1", () => {
  const state = reactive<PolicyState>({
    policyMapByName: new Map(),
  });

  const policyList = computed(() => Array.from(state.policyMapByName.values()));

  const getQueryDataPolicyByParent = (parent: string): QueryDataPolicy => {
    const policy = getPolicyByParentAndType({
      parentPath: parent,
      policyType: PolicyType.DATA_QUERY,
    });
    return policy?.policy?.case === "queryDataPolicy"
      ? policy.policy.value
      : create(QueryDataPolicySchema, {
          maximumResultSize: BigInt(
            DEFAULT_MAX_RESULT_SIZE_IN_MB * 1024 * 1024
          ),
          maximumResultRows: -1,
          timeout: create(DurationSchema, {
            seconds: BigInt(0),
          }),
        });
  };

  const formatQueryDataPolicy = (policy: QueryDataPolicy) => {
    let maximumResultSize = Number(policy.maximumResultSize);
    let maximumResultRows = policy.maximumResultRows;
    let queryTimeoutInSeconds = Number(policy.timeout?.seconds ?? 0);

    if (maximumResultSize <= 0) {
      maximumResultSize = DEFAULT_MAX_RESULT_SIZE_IN_MB * 1024 * 1024;
    }
    if (maximumResultRows <= 0) {
      maximumResultRows = Number.MAX_VALUE;
    }
    if (queryTimeoutInSeconds <= 0) {
      queryTimeoutInSeconds = Number.MAX_VALUE;
    }

    return {
      disableCopyData: policy.disableCopyData,
      disableExport: policy.disableExport,
      maximumResultSize,
      maximumResultRows,
      queryTimeoutInSeconds,
    };
  };

  const getEffectiveQueryDataPolicyForProject = (project: string) => {
    const workspacePolicy = formatQueryDataPolicy(
      getQueryDataPolicyByParent("")
    );
    const projectPolicy = formatQueryDataPolicy(
      getQueryDataPolicyByParent(project)
    );

    return {
      disableCopyData:
        projectPolicy.disableCopyData || workspacePolicy.disableCopyData,
      disableExport:
        projectPolicy.disableExport || workspacePolicy.disableExport,
      maximumResultRows: Math.min(
        projectPolicy.maximumResultRows,
        workspacePolicy.maximumResultRows
      ),
      maximumResultSize: Math.min(
        projectPolicy.maximumResultSize,
        workspacePolicy.maximumResultSize
      ),
      queryTimeoutInSeconds: Math.min(
        projectPolicy.queryTimeoutInSeconds,
        workspacePolicy.queryTimeoutInSeconds
      ),
    };
  };

  const fetchPolicies = async ({
    resourceType,
    policyType,
    parent,
    showDeleted = false,
  }: {
    resourceType: PolicyResourceType;
    policyType?: PolicyType;
    parent?: string;
    showDeleted?: boolean;
  }) => {
    const request = create(ListPoliciesRequestSchema, {
      parent: parent ?? getPolicyParentByResourceType(resourceType),
      policyType: policyType,
      showDeleted,
    });
    const response = await orgPolicyServiceClientConnect.listPolicies(request);
    const policies = response.policies;
    for (const policy of policies) {
      state.policyMapByName.set(policy.name, policy);
    }
    return policies;
  };

  const getPolicies = ({
    resourceType,
    policyType,
    showDeleted,
  }: {
    resourceType: PolicyResourceType;
    policyType: PolicyType;
    showDeleted?: boolean;
  }) => {
    const response: Policy[] = [];
    for (const [_, policy] of state.policyMapByName) {
      if (policy.resourceType != resourceType || policy.type != policyType) {
        continue;
      }
      if (!showDeleted && !policy.enforce) {
        continue;
      }
      response.push(policy);
    }
    return response;
  };

  const getOrFetchPolicyByParentAndType = async ({
    parentPath,
    policyType,
    refresh,
  }: {
    parentPath: string;
    policyType: PolicyType;
    refresh?: boolean;
  }) => {
    const name = replacePolicyTypeNameToLowerCase(
      `${parentPath}/${policyNamePrefix}${PolicyType[policyType]}`
    );
    return getOrFetchPolicyByName(name, refresh);
  };

  const getOrFetchPolicyByName = async (name: string, refresh = false) => {
    const cachedData = getPolicyByName(replacePolicyTypeNameToLowerCase(name));
    if (cachedData && !refresh) {
      return cachedData;
    }
    try {
      const request = create(GetPolicyRequestSchema, { name });
      const policy = await orgPolicyServiceClientConnect.getPolicy(request, {
        contextValues: createContextValues().set(silentContextKey, true),
      });
      state.policyMapByName.set(policy.name, policy);
      return policy;
    } catch (error) {
      if (error instanceof ConnectError && error.code === Code.NotFound) {
        // To prevent unnecessary requests, cache empty policies if not found.
        const emptyPolicy = create(PolicySchema, { name });
        state.policyMapByName.set(name, emptyPolicy);
      }
    }
  };

  const getPolicyByParentAndType = ({
    parentPath,
    policyType,
  }: {
    parentPath: string;
    policyType: PolicyType;
  }) => {
    const name = replacePolicyTypeNameToLowerCase(
      `${parentPath}/${policyNamePrefix}${PolicyType[policyType]}`
    );
    return getPolicyByName(name);
  };

  const getPolicyByName = (name: string) => {
    const policy = state.policyMapByName.get(
      replacePolicyTypeNameToLowerCase(name)
    );
    return policy;
  };

  const upsertPolicy = async ({
    parentPath,
    policy,
  }: {
    parentPath: string;
    policy: Partial<Policy>;
  }) => {
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
    const response = await orgPolicyServiceClientConnect.updatePolicy(request);
    state.policyMapByName.set(response.name, response);
    return response;
  };

  const deletePolicy = async (name: string) => {
    const request = create(DeletePolicyRequestSchema, { name });
    await orgPolicyServiceClientConnect.deletePolicy(request);
    state.policyMapByName.delete(name);
  };

  return {
    policyList,
    fetchPolicies,
    getPolicies,
    getOrFetchPolicyByParentAndType,
    getOrFetchPolicyByName,
    getPolicyByParentAndType,
    getPolicyByName,
    upsertPolicy,
    deletePolicy,
    getQueryDataPolicyByParent,
    getEffectiveQueryDataPolicyForProject,
  };
});

const getUpdateMaskFromPolicyType = (policyType: PolicyType) => {
  switch (policyType) {
    case PolicyType.ROLLOUT_POLICY:
      return [PolicySchema.field.rolloutPolicy.name];
    case PolicyType.MASKING_EXEMPTION:
      return [PolicySchema.field.maskingExemptionPolicy.name];
    case PolicyType.MASKING_RULE:
      return [PolicySchema.field.maskingRulePolicy.name];
    case PolicyType.DATA_QUERY:
      return [PolicySchema.field.queryDataPolicy.name];
    case PolicyType.DATA_SOURCE_QUERY:
      return [PolicySchema.field.dataSourceQueryPolicy.name];
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
    return store.getPolicyByParentAndType({
      parentPath,
      policyType,
    });
  });
  return {
    policy,
    ready,
  };
};

// getEmptyRolloutPolicy returns a default rollout policy for UI display purposes.
//
// IMPORTANT: These defaults MUST match the backend defaults defined in:
// - backend/store/policy.go: GetDefaultRolloutPolicy()
// - backend/api/v1/org_policy_service.go: getDefaultRolloutPolicy()
//
// This function is used for:
// 1. Showing default policy in creation forms (so users see what they'll get)
// 2. Allowing users to customize before creation
// 3. Detecting if user customized from defaults (only persist if customized)
//
// The backend automatically returns these defaults via GetPolicy API when no
// custom policy exists in the database, so we only persist customizations.
//
// Default values:
// - automatic: false (manual rollout required)
// - roles: [] (no role restrictions)
// - requiredIssueApproval: true (issue must be approved before rollout)
// - planCheckEnforcement: ERROR_ONLY (block rollout only on errors, not warnings)
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
      }),
    },
  });
};
