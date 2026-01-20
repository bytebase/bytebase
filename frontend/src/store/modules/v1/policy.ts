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
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type {
  Policy,
  QueryDataPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  DeletePolicyRequestSchema,
  GetPolicyRequestSchema,
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  QueryDataPolicy_Restriction,
  QueryDataPolicySchema,
  RolloutPolicySchema,
  UpdatePolicyRequestSchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { useCurrentUserV1 } from "./auth";

interface PolicyState {
  policyMapByName: Map<string, Policy>;
}

export const DEFAULT_MAX_RESULT_SIZE_IN_MB = 100;

export const replacePolicyTypeNameToLowerCase = (name: string) => {
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

export const usePolicyV1Store = defineStore("policy_v1", () => {
  const state = reactive<PolicyState>({
    policyMapByName: new Map(),
  });

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
          adminDataSourceRestriction:
            QueryDataPolicy_Restriction.RESTRICTION_UNSPECIFIED,
          disallowDdl: false,
          disallowDml: false,
        });
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
    getOrFetchPolicyByParentAndType,
    getOrFetchPolicyByName,
    getPolicyByParentAndType,
    upsertPolicy,
    deletePolicy,
    getQueryDataPolicyByParent,
  };
});

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
    case PolicyType.TAG:
      return [PolicySchema.field.tagPolicy.name];
    case PolicyType.POLICY_TYPE_UNSPECIFIED:
      throw new Error("unexpected POLICY_TYPE_UNSPECIFIED");
    default:
      throw new Error(`unknown policyType ${policyType satisfies never}`);
  }
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

export const useDataSourceRestrictionPolicy = (
  database: MaybeRef<Database>
) => {
  const store = usePolicyV1Store();
  const ready = ref(false);

  watchEffect(async () => {
    await store.getOrFetchPolicyByParentAndType({
      parentPath: unref(database).project,
      policyType: PolicyType.DATA_QUERY,
    });

    const environment = unref(database).effectiveEnvironment;
    if (environment) {
      await store.getOrFetchPolicyByParentAndType({
        parentPath: environment,
        policyType: PolicyType.DATA_QUERY,
      });
    }
    ready.value = true;
  });

  const dataSourceRestriction = computed(() => {
    const projectLevelPolicy = store.getQueryDataPolicyByParent(
      unref(database).project
    );
    const projectLevelAdminDSRestriction =
      projectLevelPolicy.adminDataSourceRestriction;

    let envLevelAdminDSRestriction =
      QueryDataPolicy_Restriction.RESTRICTION_UNSPECIFIED;
    const environment = unref(database).effectiveEnvironment;
    if (environment) {
      const envLevelPolicy = store.getQueryDataPolicyByParent(environment);
      envLevelAdminDSRestriction = envLevelPolicy.adminDataSourceRestriction;
    }

    return {
      environmentPolicy: envLevelAdminDSRestriction,
      projectPolicy: projectLevelAdminDSRestriction,
    };
  });

  return {
    ready,
    dataSourceRestriction,
  };
};

export const useEffectiveQueryDataPolicyForProject = (
  project: MaybeRef<string>
) => {
  const store = usePolicyV1Store();
  const ready = ref(false);

  watchEffect(() => {
    const projectName = unref(project);
    // Fetch both workspace-level and project-level DATA_QUERY policies
    Promise.all([
      store.getOrFetchPolicyByParentAndType({
        parentPath: "",
        policyType: PolicyType.DATA_QUERY,
      }),
      store.getOrFetchPolicyByParentAndType({
        parentPath: projectName,
        policyType: PolicyType.DATA_QUERY,
      }),
    ]).finally(() => (ready.value = true));
  });

  const policy = computed(() => {
    const workspacePolicy = formatQueryDataPolicy(
      store.getQueryDataPolicyByParent("")
    );
    const projectPolicy = formatQueryDataPolicy(
      store.getQueryDataPolicyByParent(unref(project))
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
