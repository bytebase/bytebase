import axios from "axios";
import {
  EnvironmentId,
  Environment,
  EnvironmentCreate,
  EnvironmentState,
  ResourceObject,
  unknown,
  RowStatus,
  EnvironmentPatch,
  PrincipalId,
  empty,
  EMPTY_ID,
  Principal,
} from "../../types";

function convert(
  environment: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Environment {
  const creator = environment.attributes.creator as Principal;
  const updater = environment.attributes.updater as Principal;

  return {
    ...(environment.attributes as Omit<
      Environment,
      "id" | "creator" | "updater"
    >),
    id: parseInt(environment.id),
    creator,
    updater,
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

  environmentById:
    (state: EnvironmentState) =>
    (environmentId: EnvironmentId): Environment => {
      if (environmentId == EMPTY_ID) {
        return empty("ENVIRONMENT") as Environment;
      }

      for (const environment of state.environmentList) {
        if (environment.id == environmentId) {
          return environment;
        }
      }
      return unknown("ENVIRONMENT") as Environment;
    },
};

const actions = {
  async fetchEnvironmentList(
    { commit, rootGetters }: any,
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

    return environmentList;
  },

  async createEnvironment(
    { commit, rootGetters }: any,
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

    return createdEnvironment;
  },

  async reorderEnvironmentList(
    { commit, rootGetters }: any,
    {
      updaterId,
      orderedEnvironmentList,
    }: {
      updaterId: PrincipalId;
      orderedEnvironmentList: Environment[];
    }
  ) {
    const list: any[] = [];
    orderedEnvironmentList.forEach((item, index) => {
      list.push({
        id: item.id,
        type: "environmentPatch",
        attributes: {
          updaterId,
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
      environmentId,
      environmentPatch,
    }: {
      environmentId: EnvironmentId;
      environmentPatch: EnvironmentPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/environment/${environmentId}`, {
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
