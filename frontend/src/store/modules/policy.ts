import axios from "axios";
import {
  Environment,
  EnvironmentID,
  PolicyState,
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
  const environmentID = (
    policy.relationships!.environment.data as ResourceIdentifier
  ).id;
  let environment: Environment = unknown("ENVIRONMENT") as Environment;
  environment.id = parseInt(environmentID);

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

const state: () => PolicyState = () => ({
  policyMapByEnvironmentID: new Map(),
});

const getters = {
  policyByEnvironmentIDAndType:
    (state: PolicyState) =>
    (environmentID: EnvironmentID, type: PolicyType): Policy | undefined => {
      const map = state.policyMapByEnvironmentID.get(environmentID);
      if (map) {
        return map.get(type);
      }
      return undefined;
    },
};

const actions = {
  async fetchPolicyByEnvironmentAndType(
    { commit, rootGetters }: any,
    { environmentID, type }: { environmentID: EnvironmentID; type: PolicyType }
  ): Promise<Policy> {
    const data = (
      await axios.get(`/api/policy/environment/${environmentID}?type=${type}`)
    ).data;
    const policy = convert(data.data, data.included, rootGetters);
    commit("setPolicyByEnvironmentID", { environmentID, policy });

    return policy;
  },

  async upsertPolicyByEnvironmentAndType(
    { commit, rootGetters }: any,
    {
      environmentID,
      type,
      policyUpsert,
    }: {
      environmentID: EnvironmentID;
      type: PolicyType;
      policyUpsert: PolicyUpsert;
    }
  ): Promise<Policy> {
    const data = (
      await axios.patch(
        `/api/policy/environment/${environmentID}?type=${type}`,
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
    const policy = convert(data.data, data.included, rootGetters);

    commit("setPolicyByEnvironmentID", { environmentID, policy });

    return policy;
  },
};

const mutations = {
  setPolicyByEnvironmentID(
    state: PolicyState,
    {
      environmentID,
      policy,
    }: {
      environmentID: EnvironmentID;
      policy: Policy;
    }
  ) {
    const map = state.policyMapByEnvironmentID.get(environmentID);
    if (map) {
      map.set(policy.type, policy);
    } else {
      state.policyMapByEnvironmentID.set(
        environmentID,
        new Map([[policy.type, policy]])
      );
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
