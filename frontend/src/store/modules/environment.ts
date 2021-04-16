import axios from "axios";
import {
  EnvironmentId,
  Environment,
  EnvironmentNew,
  EnvironmentState,
  ResourceObject,
  unknown,
  RowStatus,
  EnvironmentPatch,
  BatchUpdate,
} from "../../types";

function convert(environment: ResourceObject): Environment {
  return {
    ...(environment.attributes as Omit<Environment, "id">),
    id: environment.id,
  };
}

const state: () => EnvironmentState = () => ({
  environmentList: [],
});

const getters = {
  environmentList: (state: EnvironmentState) => (
    rowStatusList?: RowStatus[]
  ): Environment[] => {
    return state.environmentList.filter((environment: Environment) => {
      return (
        (!rowStatusList && environment.rowStatus == "NORMAL") ||
        (rowStatusList && rowStatusList.includes(environment.rowStatus))
      );
    });
  },

  environmentById: (state: EnvironmentState) => (
    environmentId: EnvironmentId
  ): Environment => {
    for (const environment of state.environmentList) {
      if (environment.id == environmentId) {
        return environment;
      }
    }
    return unknown("ENVIRONMENT") as Environment;
  },
};

const actions = {
  async fetchEnvironmentList({ commit }: any, rowStatusList?: RowStatus[]) {
    const path =
      "/api/environment" +
      (rowStatusList ? "?rowstatus=" + rowStatusList.join(",") : "");
    const environmentList = (await axios.get(path)).data.data.map(
      (env: ResourceObject) => {
        return convert(env);
      }
    );

    commit("upsertEnvironmentList", environmentList);

    return environmentList;
  },

  async createEnvironment({ commit }: any, newEnvironment: EnvironmentNew) {
    const createdEnvironment = convert(
      (
        await axios.post(`/api/environment`, {
          data: {
            type: "environment",
            attributes: newEnvironment,
          },
        })
      ).data.data
    );

    commit("upsertEnvironmentList", [createdEnvironment]);

    return createdEnvironment;
  },

  async reorderEnvironmentList(
    { commit }: any,
    orderedEnvironmentList: Environment[]
  ) {
    const batchUpdate: BatchUpdate = {
      idList: orderedEnvironmentList.map((item) => item.id),
      fieldMaskList: ["order"],
      rowValueList: orderedEnvironmentList.map((_, index) => [index]),
    };
    const environmentList = (
      await axios.patch(`/api/environment/batch`, {
        data: {
          attributes: batchUpdate,
          type: "batchupdate",
        },
      })
    ).data.data.map((env: ResourceObject) => {
      return convert(env);
    });

    commit("upsertEnvironmentList", environmentList);

    return environmentList;
  },

  async patchEnvironment(
    { commit }: any,
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
      ).data.data
    );

    commit("upsertEnvironmentList", [updatedEnvironment]);

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

    commit("deleteEnvironmentById", id);
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

  deleteEnvironmentById(state: EnvironmentState, environmentId: EnvironmentId) {
    const i = state.environmentList.findIndex(
      (item: Environment) => item.id == environmentId
    );

    if (i >= 0) {
      state.environmentList.splice(i, 1);
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
