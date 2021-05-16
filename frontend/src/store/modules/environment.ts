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
} from "../../types";

function convert(environment: ResourceObject, rootGetters: any): Environment {
  const creator = rootGetters["principal/principalById"](
    environment.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    environment.attributes.updaterId
  );
  return {
    ...(environment.attributes as Omit<Environment, "id">),
    id: environment.id,
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
    (environment: ResourceObject): Environment => {
      return convert(environment, rootGetters);
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
    const environmentList = (await axios.get(path)).data.data.map(
      (env: ResourceObject) => {
        return convert(env, rootGetters);
      }
    );

    commit("upsertEnvironmentList", environmentList);

    return environmentList;
  },

  async createEnvironment(
    { commit, rootGetters }: any,
    newEnvironment: EnvironmentCreate
  ) {
    const createdEnvironment = convert(
      (
        await axios.post(`/api/environment`, {
          data: {
            type: "environment",
            attributes: newEnvironment,
          },
        })
      ).data.data,
      rootGetters
    );

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
        type: "environmentpatch",
        attributes: {
          updaterId,
          order: index,
        },
      });
    });
    const environmentList = (
      await axios.patch(`/api/environment/reorder`, {
        data: list,
      })
    ).data.data.map((env: ResourceObject) => {
      return convert(env, rootGetters);
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
    const updatedEnvironment = convert(
      (
        await axios.patch(`/api/environment/${environmentId}`, {
          data: {
            type: "environmentpatch",
            attributes: environmentPatch,
          },
        })
      ).data.data,
      rootGetters
    );

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
