import axios from "axios";
import {
  EnvironmentId,
  Environment,
  EnvironmentNew,
  EnvironmentState,
  ResourceObject,
} from "../../types";

function convert(environment: ResourceObject): Environment {
  return {
    id: environment.id,
    ...(environment.attributes as Omit<Environment, "id">),
  };
}

const state: () => EnvironmentState = () => ({
  environmentList: [],
});

const getters = {
  environmentList: (state: EnvironmentState) => () => {
    return state.environmentList;
  },

  environmentById: (state: EnvironmentState) => (
    environmentId: EnvironmentId
  ): Environment | undefined => {
    for (const environment of state.environmentList) {
      if (environment.id == environmentId) {
        return environment;
      }
    }
    return undefined;
  },
};

const actions = {
  async fetchEnvironmentList({ commit }: any) {
    const environmentList = (await axios.get(`/api/environment`)).data.data.map(
      (env: ResourceObject) => {
        return convert(env);
      }
    );

    commit("setEnvironmentList", environmentList);

    return environmentList;
  },

  async createEnvironment({ commit }: any, newEnvironment: EnvironmentNew) {
    const createdEnvironment = convert(
      (
        await axios.post(`/api/environment`, {
          data: {
            type: "environment",
            attributes: {
              ...newEnvironment,
            },
          },
        })
      ).data.data
    );

    commit("appendEnvironment", createdEnvironment);

    return createdEnvironment;
  },

  async reorderEnvironmentList(
    { commit }: any,
    orderedEnvironmentList: Environment[]
  ) {
    const environmentList = (
      await axios.patch(`/api/environment/batch`, {
        data: {
          attributes: {
            idList: orderedEnvironmentList.map((item) => item.id),
            fieldMaskList: ["order"],
            rowValueList: orderedEnvironmentList.map((_, index) => [index]),
          },
          type: "batchupdate",
        },
      })
    ).data.data.map((env: ResourceObject) => {
      return convert(env);
    });

    commit("setEnvironmentList", environmentList);

    return environmentList;
  },

  async patchEnvironment({ commit }: any, environment: Environment) {
    const { id, ...attrs } = environment;
    const updatedEnvironment = convert(
      (
        await axios.patch(`/api/environment/${environment.id}`, {
          data: {
            type: "environment",
            attributes: {
              ...attrs,
            },
          },
        })
      ).data.data
    );

    commit("replaceEnvironmentInList", updatedEnvironment);

    return updatedEnvironment;
  },

  async deleteEnvironmentById(
    { state, commit }: { state: EnvironmentState; commit: any },
    id: EnvironmentId
  ) {
    await axios.delete(`/api/environment/${id}`);

    const newList = state.environmentList.filter((item: Environment) => {
      return item.id != id;
    });

    commit("setEnvironmentList", newList);
  },
};

const mutations = {
  setEnvironmentList(state: EnvironmentState, environmentList: Environment[]) {
    state.environmentList = environmentList;
  },

  appendEnvironment(state: EnvironmentState, newEnvironment: Environment) {
    state.environmentList.push(newEnvironment);
  },

  replaceEnvironmentInList(
    state: EnvironmentState,
    updatedEnvironment: Environment
  ) {
    const i = state.environmentList.findIndex(
      (item: Environment) => item.id == updatedEnvironment.id
    );
    if (i != -1) {
      state.environmentList[i] = updatedEnvironment;
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
