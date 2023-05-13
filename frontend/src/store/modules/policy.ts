import { defineStore } from "pinia";
import axios from "axios";
import {
  EMPTY_ID,
  EnvironmentId,
  PolicyState,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  UNKNOWN_ID,
} from "@/types";
import { Policy, PolicyType, PolicyUpsert } from "@/types/policy";
import { useEnvironmentStore } from "./environment";

function convertEnvironment(
  policy: ResourceObject,
  includedList: ResourceObject[]
) {
  // The `environment` relationship cannot retire now.
  // But for database-level policies it will be null.
  // In order not to break the typings, we will fallback to <<Unknown Environment>>
  // for database-level policies here.
  let environment = unknown("ENVIRONMENT");
  const data = policy.relationships?.environment?.data as
    | ResourceIdentifier
    | undefined;
  if (data) {
    const environmentId = data.id;
    environment.id = parseInt(environmentId);

    const environmentStore = useEnvironmentStore();
    for (const item of includedList || []) {
      if (item.type == "environment" && data.id == item.id) {
        environment = environmentStore.convert(item, includedList);
      }
    }
  }
  return environment;
}

function convert(
  policy: ResourceObject,
  includedList: ResourceObject[]
): Policy {
  const environment = convertEnvironment(policy, includedList);

  const result = {
    ...(policy.attributes as Omit<Policy, "id" | "environment" | "payload">),
    id: parseInt(policy.id),
    environment,
    payload: JSON.parse((policy.attributes.payload as string) || "{}"),
  };

  // [GET]/api/policy/database/${databaseId}?type=${type} sometimes
  // accidentally returns empty object with resourceId={databaseId} when the
  // policy entity doesn't exist.
  // So we need to rewrite the resourceId here to improve robustness.
  if (result.id === UNKNOWN_ID) result.resourceId = UNKNOWN_ID;
  if (result.id === EMPTY_ID) result.resourceId = EMPTY_ID;

  return result;
}

export const usePolicyStore = defineStore("policy", {
  state: (): PolicyState => ({
    policyMapByEnvironmentId: new Map(),
  }),
  actions: {
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
  },
});
