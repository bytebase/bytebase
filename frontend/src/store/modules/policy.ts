import axios from "axios";
import {
  Environment,
  EnvironmentId,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "../../types";
import { Policy, PolicyType, PolicyUpsert } from "../../types/policy";

function convert(
  policy: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Policy {
  const environmentId = (
    policy.relationships!.environment.data as ResourceIdentifier
  ).id;
  let environment: Environment = unknown("ENVIRONMENT") as Environment;
  environment.id = parseInt(environmentId);

  for (const item of includedList || []) {
    if (
      item.type == "environment" &&
      (policy.relationships!.environment.data as ResourceIdentifier).id ==
        item.id
    ) {
      environment = rootGetters["environment/convert"](item, includedList);
    }
  }

  return {
    ...(policy.attributes as Omit<Policy, "id" | "environment" | "payload">),
    id: parseInt(policy.id),
    environment,
    payload: JSON.parse((policy.attributes.payload as string) || "{}"),
  };
}

const getters = {};

const actions = {
  async fetchPolicyByEnvironmentAndType(
    { rootGetters }: any,
    { environmentId, type }: { environmentId: EnvironmentId; type: PolicyType }
  ): Promise<Policy> {
    const data = (
      await axios.get(`/api/policy/environment/${environmentId}?type=${type}`)
    ).data;
    return convert(data.data, data.included, rootGetters);
  },

  async upsertPolicyByEnvironmentAndType(
    { rootGetters }: any,
    {
      environmentId,
      type,
      policyUpsert,
    }: {
      environmentId: EnvironmentId;
      type: PolicyType;
      policyUpsert: PolicyUpsert;
    }
  ): Promise<Policy> {
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
    return convert(data.data, data.included, rootGetters);
  },
};

const mutations = {};

export default {
  namespaced: true,
  getters,
  actions,
  mutations,
};
