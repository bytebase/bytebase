import axios from "axios";
import {
  Environment,
  EnvironmentId,
  PolicyState,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "../../types";
import { Policy, PolicyType, PolicyUpsert } from "../../types/policy";
import { getPrincipalFromIncludedList } from "../pinia";

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

const state: () => PolicyState = () => ({
  policyMapByEnvironmentId: new Map(),
});

const getters = {
  policyByEnvironmentIdAndType:
    (state: PolicyState) =>
    (environmentId: EnvironmentId, type: PolicyType): Policy | undefined => {
      const map = state.policyMapByEnvironmentId.get(environmentId);
      if (map) {
        return map.get(type);
      }
      return undefined;
    },
};

const actions = {
  async fetchPolicyByEnvironmentAndType(
    { commit, rootGetters }: any,
    { environmentId, type }: { environmentId: EnvironmentId; type: PolicyType }
  ): Promise<Policy> {
    const data = (
      await axios.get(`/api/policy/environment/${environmentId}?type=${type}`)
    ).data;
    const policy = convert(data.data, data.included, rootGetters);
    commit("setPolicyByEnvironmentId", { environmentId, policy });

    return policy;
  },

  async upsertPolicyByEnvironmentAndType(
    { commit, rootGetters }: any,
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
    const policy = convert(data.data, data.included, rootGetters);

    commit("setPolicyByEnvironmentId", { environmentId, policy });

    return policy;
  },
};

const mutations = {
  setPolicyByEnvironmentId(
    state: PolicyState,
    {
      environmentId,
      policy,
    }: {
      environmentId: EnvironmentId;
      policy: Policy;
    }
  ) {
    const map = state.policyMapByEnvironmentId.get(environmentId);
    if (map) {
      map.set(policy.type, policy);
    } else {
      state.policyMapByEnvironmentId.set(
        environmentId,
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
