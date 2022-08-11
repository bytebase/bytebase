import { defineStore } from "pinia";
import axios from "axios";
import {
  Environment,
  EnvironmentId,
  PolicyState,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  UNKNOWN_ID,
} from "@/types";
import {
  PipelineApprovalPolicyPayload,
  Policy,
  PolicyType,
  PolicyUpsert,
} from "@/types/policy";
import { getPrincipalFromIncludedList } from "./principal";
import { useEnvironmentStore } from "./environment";
import { computed, Ref, watchEffect } from "vue";
import { useCurrentUser } from "./auth";

function convert(
  policy: ResourceObject,
  includedList: ResourceObject[]
): Policy {
  const environmentId = (
    policy.relationships!.environment.data as ResourceIdentifier
  ).id;
  let environment: Environment = unknown("ENVIRONMENT") as Environment;
  environment.id = parseInt(environmentId);

  const environmentStore = useEnvironmentStore();
  for (const item of includedList || []) {
    if (
      item.type == "environment" &&
      (policy.relationships!.environment.data as ResourceIdentifier).id ==
        item.id
    ) {
      environment = environmentStore.convert(item, includedList);
    }
  }

  const result = {
    ...(policy.attributes as Omit<
      Policy,
      "id" | "environment" | "payload" | "creator" | "updater"
    >),
    id: parseInt(policy.id),
    creator: getPrincipalFromIncludedList(
      policy.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      policy.relationships!.updater.data,
      includedList
    ),
    environment,
    payload: JSON.parse((policy.attributes.payload as string) || "{}"),
  };
  if (result.type === "bb.policy.pipeline-approval") {
    const payload = result.payload as PipelineApprovalPolicyPayload;
    if (!payload.assigneeGroupList) {
      // Assign an empty array as fallback
      payload.assigneeGroupList = [];
    }
  }

  return result;
}

export const usePolicyStore = defineStore("policy", {
  state: (): PolicyState => ({
    policyMapByEnvironmentId: new Map(),
  }),
  actions: {
    getPolicyByEnvironmentIdAndType(
      environmentId: EnvironmentId,
      type: PolicyType
    ): Policy | undefined {
      const map = this.policyMapByEnvironmentId.get(environmentId);
      if (map) {
        return map.get(type);
      }
      return undefined;
    },
    setPolicyByEnvironmentId({
      environmentId,
      policy,
    }: {
      environmentId: EnvironmentId;
      policy: Policy;
    }) {
      const map = this.policyMapByEnvironmentId.get(environmentId);
      if (map) {
        map.set(policy.type, policy);
      } else {
        this.policyMapByEnvironmentId.set(
          environmentId,
          new Map([[policy.type, policy]])
        );
      }
    },
    async fetchPolicyListByType(type: PolicyType): Promise<Policy[]> {
      const data: { data: ResourceObject[]; included: ResourceObject[] } = (
        await axios.get(`/api/policy?type=${type}`)
      ).data;

      return data.data.map((d) => convert(d, data.included));
    },
    async fetchPolicyByEnvironmentAndType({
      environmentId,
      type,
    }: {
      environmentId: EnvironmentId;
      type: PolicyType;
    }): Promise<Policy> {
      const data = (
        await axios.get(`/api/policy/environment/${environmentId}?type=${type}`)
      ).data;
      const policy = convert(data.data, data.included);
      this.setPolicyByEnvironmentId({ environmentId, policy });

      return policy;
    },
    async upsertPolicyByEnvironmentAndType({
      environmentId,
      type,
      policyUpsert,
    }: {
      environmentId: EnvironmentId;
      type: PolicyType;
      policyUpsert: PolicyUpsert;
    }): Promise<Policy> {
      const data = (
        await axios.patch(
          `/api/policy/environment/${environmentId}?type=${type}`,
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
      const policy = convert(data.data, data.included);

      this.setPolicyByEnvironmentId({ environmentId, policy });

      return policy;
    },
    async deletePolicyByEnvironmentAndType({
      environmentId,
      type,
    }: {
      environmentId: EnvironmentId;
      type: PolicyType;
    }) {
      await axios.delete(
        `/api/policy/environment/${environmentId}?type=${type}`
      );
    },
  },
});

export const usePolicyByEnvironmentAndType = (
  params: Ref<{ environmentId: EnvironmentId; type: PolicyType }>
) => {
  const store = usePolicyStore();
  const currentUser = useCurrentUser();
  watchEffect(() => {
    if (currentUser.value.id === UNKNOWN_ID) return;

    store.fetchPolicyByEnvironmentAndType(params.value);
  });

  return computed(() =>
    store.getPolicyByEnvironmentIdAndType(
      params.value.environmentId,
      params.value.type
    )
  );
};
