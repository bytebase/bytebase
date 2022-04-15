import { defineStore } from "pinia";
import axios from "axios";
import {
  Environment,
  EnvironmentId,
  PolicyState,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "@/types";
import { Policy, PolicyType, PolicyUpsert } from "@/types/policy";
import { getPrincipalFromIncludedList } from "./principal";
import { useEnvironmentStore } from "./environment";

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

  return {
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
                payload: JSON.stringify(policyUpsert.payload),
              },
            },
          }
        )
      ).data;
      const policy = convert(data.data, data.included);

      this.setPolicyByEnvironmentId({ environmentId, policy });

      return policy;
    },
  },
});
