import { create as createProto } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { orgPolicyServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import {
  GetPolicyRequestSchema,
  PolicySchema,
  PolicyType,
  type QueryDataPolicy,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { AppSliceCreator, PolicySlice } from "./types";

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
});
