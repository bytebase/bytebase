import { create as createProto } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { orgPolicyServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import {
  DeletePolicyRequestSchema,
  GetPolicyRequestSchema,
  type Policy,
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  type QueryDataPolicy,
  QueryDataPolicySchema,
  RolloutPolicySchema,
  UpdatePolicyRequestSchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { AppSliceCreator, PolicySlice } from "./types";

// Mirror of the Pinia `getUpdateMaskFromPolicyType`.
const getUpdateMaskFromPolicyType = (policyType: PolicyType): string[] => {
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
    default:
      throw new Error(`unexpected policy type ${policyType}`);
  }
};

// Inlined to keep the app store's load graph free of `@/store/modules/v1/common`.
const POLICY_NAME_PREFIX = "policies/";

// Mirror of the Pinia helper — normalizes `policies/{TYPE}` segments to
// lowercase so the cache key matches what the server returns.
const replacePolicyTypeNameToLowerCase = (name: string): string => {
  const pattern = /(^|\/)policies\/([^/]+)($|\/)/;
  const replaced = name.replace(
    pattern,
    (_, left: string, policyType: string, right: string) =>
      `${left}policies/${policyType.toLowerCase()}${right}`
  );
  return replaced.startsWith("/") ? replaced.slice(1) : replaced;
};

// Stable empty singleton — subscribers compare by reference, so we MUST NOT
// build a fresh object on every call (mirrors the Pinia behavior).
const EMPTY_QUERY_DATA_POLICY: QueryDataPolicy = createProto(
  QueryDataPolicySchema,
  { maximumResultRows: -1 }
);

// Pure helper relocated from the legacy Pinia policy module.
export const getEmptyRolloutPolicy = (
  parentPath: string,
  resourceType: PolicyResourceType
): Policy =>
  createProto(PolicySchema, {
    name: policyResourceName(parentPath, PolicyType.ROLLOUT_POLICY),
    inheritFromParent: false,
    type: PolicyType.ROLLOUT_POLICY,
    resourceType,
    enforce: true,
    policy: {
      case: "rolloutPolicy",
      value: createProto(RolloutPolicySchema, { automatic: false, roles: [] }),
    },
  });

const policyResourceName = (parent: string, policyType: PolicyType) =>
  replacePolicyTypeNameToLowerCase(
    `${parent}/${POLICY_NAME_PREFIX}${PolicyType[policyType]}`
  );

/**
 * Port of the SQL-editor-used subset of the legacy Pinia `usePolicyV1Store`:
 * `policyMapByName` cache + sync/async getters keyed by resource name.
 * Failures on `getOrFetchPolicyByName` cache an empty Policy under the
 * requested name (matches Pinia's "don't re-hit a missing policy" behavior).
 */
export const createPolicySlice: AppSliceCreator<PolicySlice> = (set, get) => ({
  policyMapByName: {},
  policyRequests: {},

  getPolicyByName: (name) =>
    get().policyMapByName[replacePolicyTypeNameToLowerCase(name)],

  getOrFetchPolicyByName: async (name, refresh = false) => {
    const key = replacePolicyTypeNameToLowerCase(name);
    const cached = get().policyMapByName[key];
    if (cached && !refresh) return cached;
    const pending = get().policyRequests[key];
    if (pending) return pending;

    const request = orgPolicyServiceClientConnect
      .getPolicy(createProto(GetPolicyRequestSchema, { name }), {
        contextValues: createContextValues().set(silentContextKey, true),
      })
      .then((policy) => {
        set((state) => {
          const { [key]: _, ...policyRequests } = state.policyRequests;
          return {
            policyMapByName: {
              ...state.policyMapByName,
              [policy.name]: policy,
            },
            policyRequests,
          };
        });
        return policy;
      })
      .catch((error): undefined => {
        set((state) => {
          const { [key]: _, ...policyRequests } = state.policyRequests;
          // Cache an empty policy on NotFound so repeated lookups don't
          // hammer the backend (parity with Pinia).
          if (error instanceof ConnectError && error.code === Code.NotFound) {
            return {
              policyMapByName: {
                ...state.policyMapByName,
                [key]: createProto(PolicySchema, { name }),
              },
              policyRequests,
            };
          }
          return { policyRequests };
        });
        return undefined;
      });
    set((state) => ({
      policyRequests: { ...state.policyRequests, [key]: request },
    }));
    return request;
  },

  getPolicyByParentAndType: ({ parentPath, policyType }) =>
    get().policyMapByName[policyResourceName(parentPath, policyType)],

  getOrFetchPolicyByParentAndType: ({ parentPath, policyType, refresh }) =>
    get().getOrFetchPolicyByName(
      policyResourceName(parentPath, policyType),
      refresh
    ),

  getQueryDataPolicyByParent: (parent) => {
    const policy = get().getPolicyByParentAndType({
      parentPath: parent,
      policyType: PolicyType.DATA_QUERY,
    });
    return policy?.policy?.case === "queryDataPolicy"
      ? policy.policy.value
      : EMPTY_QUERY_DATA_POLICY;
  },

  upsertPolicy: async ({ parentPath, policy }) => {
    if (!policy.type) {
      throw new Error("policy type is required");
    }
    const name = policyResourceName(parentPath, policy.type);
    const response = await orgPolicyServiceClientConnect.updatePolicy(
      createProto(UpdatePolicyRequestSchema, {
        policy: createProto(PolicySchema, {
          name,
          inheritFromParent: policy.inheritFromParent ?? false,
          type: policy.type,
          resourceType:
            policy.resourceType ?? PolicyResourceType.RESOURCE_TYPE_UNSPECIFIED,
          enforce: policy.enforce ?? false,
          policy: policy.policy,
        }),
        updateMask: { paths: getUpdateMaskFromPolicyType(policy.type) },
        allowMissing: true,
      })
    );
    set((state) => ({
      policyMapByName: { ...state.policyMapByName, [response.name]: response },
    }));
    return response;
  },

  deletePolicy: async (name) => {
    await orgPolicyServiceClientConnect.deletePolicy(
      createProto(DeletePolicyRequestSchema, { name })
    );
    const key = replacePolicyTypeNameToLowerCase(name);
    set((state) => {
      const { [key]: _removed, ...policyMapByName } = state.policyMapByName;
      return { policyMapByName };
    });
  },
});
