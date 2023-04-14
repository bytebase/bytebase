import { defineStore } from "pinia";
import { computed, ref, watchEffect } from "vue";
import axios from "axios";
import { IdType, ResourceObject, unknown, UNKNOWN_ID } from "@/types";
import {
  PipelineApprovalPolicyPayload,
  Policy,
  PolicyResourceType,
  PolicyType,
  PolicyUpsert,
  SensitiveDataPolicyPayload,
} from "@/types/policy";
import { useCurrentUser } from "./auth";

function convert(
  resourceType: PolicyResourceType,
  policy: ResourceObject,
  includedList: ResourceObject[]
): Policy {
  const result: Policy = {
    ...(policy.attributes as Omit<Policy, "id" | "environment" | "payload">),
    id: parseInt(policy.id),
    resourceType,
    environment: unknown("ENVIRONMENT"),
    payload: JSON.parse((policy.attributes.payload as string) || "{}"),
  };

  if (result.type === "bb.policy.pipeline-approval") {
    const payload = result.payload as PipelineApprovalPolicyPayload;
    if (!payload.assigneeGroupList) {
      // Assign an empty array as fallback
      payload.assigneeGroupList = [];
    }
  }
  if (result.type === "bb.policy.sensitive-data") {
    const payload = result.payload as SensitiveDataPolicyPayload;
    if (!payload.sensitiveDataList) {
      // The array might be null, fill it with empty array to fallback.
      payload.sensitiveDataList = [];
    }
  }

  return result;
}

export const useSlowQueryPolicyStore = defineStore("slow-query-policy", () => {
  const policyMapByResourceType = ref(
    new Map<PolicyResourceType, Map<PolicyType, Policy[]>>()
  );

  const getPolicyListByResourceTypeAndPolicyType = (
    resourceType: PolicyResourceType,
    policyType: PolicyType
  ) => {
    return (
      policyMapByResourceType.value.get(resourceType)?.get(policyType) ?? []
    );
  };
  const setPolicyListByResourceTypeAndPolicyType = (
    resourceType: PolicyResourceType,
    policyType: PolicyType,
    policyList: Policy[]
  ) => {
    const map = policyMapByResourceType.value.get(resourceType);
    if (map) {
      map.set(policyType, policyList);
    } else {
      policyMapByResourceType.value.set(
        resourceType,
        new Map([[policyType, policyList]])
      );
    }
  };
  const updatePolicy = (policy: Policy) => {
    const { resourceType, type } = policy;
    const policyList = getPolicyListByResourceTypeAndPolicyType(
      resourceType,
      type
    );
    const index = policyList.findIndex(
      (p) => p.resourceId === policy.resourceId
    );
    if (index >= 0) {
      policyList[index] = policy;
    } else {
      policyList.push(policy);
    }
    setPolicyListByResourceTypeAndPolicyType(resourceType, type, policyList);
  };
  const removePolicy = (
    resourceType: PolicyResourceType,
    resourceId: IdType,
    policyType: PolicyType
  ) => {
    const policyList = getPolicyListByResourceTypeAndPolicyType(
      resourceType,
      policyType
    );
    const index = policyList.findIndex((p) => p.resourceId === resourceId);
    if (index >= 0) {
      policyList.splice(index, 1);
      setPolicyListByResourceTypeAndPolicyType(
        resourceType,
        policyType,
        policyList
      );
    }
  };

  const fetchPolicyListByResourceTypeAndPolicyType = async (
    resourceType: PolicyResourceType,
    policyType: PolicyType
  ) => {
    const url = `/api/policy?resourceType=${resourceType}&type=${policyType}`;
    const data: { data: ResourceObject[]; included: ResourceObject[] } = (
      await axios.get(url)
    ).data;
    const policyList = data.data.map((d) =>
      convert(resourceType, d, data.included)
    );
    setPolicyListByResourceTypeAndPolicyType(
      resourceType,
      policyType,
      policyList
    );
    return policyList;
  };
  const upsertPolicyByResourceTypeAndPolicyType = async (
    resourceType: PolicyResourceType,
    resourceId: IdType,
    policyType: PolicyType,
    policyUpsert: PolicyUpsert
  ) => {
    const data = (
      await axios.patch(
        `/api/policy/${resourceType}/${resourceId}?type=${policyType}`,
        {
          data: {
            type: "policyUpsert",
            attributes: {
              rowStatus: policyUpsert.rowStatus,
              payload: policyUpsert.payload
                ? JSON.stringify(policyUpsert.payload)
                : undefined,
            },
          },
        }
      )
    ).data;
    const policy = convert(resourceType, data.data, data.included);

    updatePolicy(policy);

    return policy;
  };
  const deletePolicyByResourceTypeAndPolicyType = async (
    resourceType: PolicyResourceType,
    resourceId: IdType,
    policyType: PolicyType
  ) => {
    await axios.delete(
      `/api/policy/${resourceType}/${resourceId}?type=${policyType}`
    );
    removePolicy(resourceType, resourceId, policyType);
  };

  return {
    getPolicyListByResourceTypeAndPolicyType,
    fetchPolicyListByResourceTypeAndPolicyType,
    upsertPolicyByResourceTypeAndPolicyType,
    deletePolicyByResourceTypeAndPolicyType,
  };
});

export const useSlowQueryPolicyList = () => {
  const store = useSlowQueryPolicyStore();
  const currentUser = useCurrentUser();
  watchEffect(() => {
    if (currentUser.value.id === UNKNOWN_ID) return;

    store.fetchPolicyListByResourceTypeAndPolicyType(
      "instance",
      "bb.policy.slow-query"
    );
  });

  return computed(() => {
    return store.getPolicyListByResourceTypeAndPolicyType(
      "instance",
      "bb.policy.slow-query"
    );
  });
};
