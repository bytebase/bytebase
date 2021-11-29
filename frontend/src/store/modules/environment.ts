import axios from "axios";
import {
  empty,
  EMPTY_ID,
  Environment,
  EnvironmentCreate,
  EnvironmentID,
  EnvironmentPatch,
  EnvironmentState,
  ResourceObject,
  RowStatus,
  unknown,
} from "../../types";

function convert(
  environment: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Environment {
  return {
    ...(environment.attributes as Omit<Environment, "id">),
    id: parseInt(environment.id),
  };
}

const state: () => EnvironmentState = () => ({
  environmentList: [],
});

const getters = {
  convert:
    (state: EnvironmentState, getters: any, rootState: any, rootGetters: any) =>
    (
      environment: ResourceObject,
      inlcudedList: ResourceObject[]
    ): Environment => {
      return convert(environment, inlcudedList, rootGetters);
    },

  environmentList:
    (state: EnvironmentState) =>
    (rowStatusList?: RowStatus[]): Environment[] => {
      return state.environmentList.filter((environment: Environment) => {
        return (
          (!rowStatusList && environment.rowStatus == "NORMAL") ||
          (rowStatusList && rowStatusList.includes(environment.rowStatus))
        );
      });
    },

  environmentByID:
    (state: EnvironmentState) =>
    (environmentID: EnvironmentID): Environment => {
      if (environmentID == EMPTY_ID) {
        return empty("ENVIRONMENT") as Environment;
      }

      for (const environment of state.environmentList) {
        if (environment.id == environmentID) {
          return environment;
        }
      }
      return unknown("ENVIRONMENT") as Environment;
    },
};

const actions = {
  async fetchEnvironmentList(
    { dispatch, commit, rootGetters }: any,
    rowStatusList?: RowStatus[]
  ) {
    const path =
      "/api/environment" +
      (rowStatusList ? "?rowstatus=" + rowStatusList.join(",") : "");
    const data = (await axios.get(path)).data;
    const environmentList = data.data.map((env: ResourceObject) => {
      return convert(env, data.included, rootGetters);
    });

    commit("upsertEnvironmentList", environmentList);

    for (const environment of environmentList) {
      await dispatch(
        "policy/fetchPolicyByEnvironmentAndType",
        {
          environmentID: environment.id,
          type: "bb.policy.pipeline-approval",
        },
        {
          root: true,
        }
      );
    }

    return environmentList;
  },

  async createEnvironment(
    { dispatch, commit, rootGetters }: any,
    newEnvironment: EnvironmentCreate
  ) {
    const data = (
      await axios.post(`/api/environment`, {
        data: {
          type: "environment",
          attributes: newEnvironment,
        },
      })
    ).data;
    const createdEnvironment = convert(data.data, data.included, rootGetters);

    commit("upsertEnvironmentList", [createdEnvironment]);

    await dispatch(
      "policy/fetchPolicyByEnvironmentAndType",
      {
        environmentID: createdEnvironment.id,
        type: "bb.policy.pipeline-approval",
      },
      {
        root: true,
      }
    );

    return createdEnvironment;
  },

  async reorderEnvironmentList(
    { commit, rootGetters }: any,
    orderedEnvironmentList: Environment[]
  ) {
    const list: any[] = [];
    orderedEnvironmentList.forEach((item, index) => {
      list.push({
        // Server uses google/jsonapi which expects a string type for the special id field.
        // Afterwards, server will automatically serialize into int as declared by the EnvironmentPatch interface.
        id: item.id.toString(),
        type: "environmentPatch",
        attributes: {
          order: index,
        },
      });
    });
    const data = (
      await axios.patch(`/api/environment/reorder`, {
        data: list,
      })
    ).data;
    const environmentList = data.data.map((env: ResourceObject) => {
      return convert(env, data.included, rootGetters);
    });

    commit("upsertEnvironmentList", environmentList);

    return environmentList;
  },

  async patchEnvironment(
    { commit, rootGetters }: any,
    {
      environmentID,
      environmentPatch,
    }: {
      environmentID: EnvironmentID;
      environmentPatch: EnvironmentPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/environment/${environmentID}`, {
        data: {
          type: "environmentPatch",
          attributes: environmentPatch,
        },
      })
    ).data;
    const updatedEnvironment = convert(data.data, data.included, rootGetters);

    commit("upsertEnvironmentList", [updatedEnvironment]);

    return updatedEnvironment;
  },
};

const mutations = {
  upsertEnvironmentList(
    state: EnvironmentState,
    environmentList: Environment[]
  ) {
    for (const environment of environmentList) {
      const i = state.environmentList.findIndex(
        (item: Environment) => item.id == environment.id
      );
      if (i != -1) {
        state.environmentList[i] = environment;
      } else {
        state.environmentList.push(environment);
      }

      state.environmentList.sort((a, b) => a.order - b.order);
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
