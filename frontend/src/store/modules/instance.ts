import axios from "axios";
import {
  InstanceId,
  Instance,
  InstanceNew,
  InstanceState,
  ResourceObject,
  Environment,
  EnvironmentId,
  ResourceIdentifier,
} from "../../types";

function convert(instance: ResourceObject, rootGetters: any): Instance {
  const environment = rootGetters["environment/environmentList"]().find(
    (env: Environment) => {
      return (
        env.id ==
        (instance.relationships!.environment.data as ResourceIdentifier).id
      );
    }
  );
  return {
    id: instance.id,
    ...(instance.attributes as Omit<Instance, "id" | "environment">),
    environment,
  };
}

const state: () => InstanceState = () => ({
  instanceById: new Map(),
});

const getters = {
  convert: (
    state: InstanceState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (instance: ResourceObject): Instance => {
    return convert(instance, rootGetters);
  },

  instanceList: (state: InstanceState) => () => {
    const list = [];
    for (const [_, instance] of state.instanceById) {
      list.push(instance);
    }
    return list.sort((a: Instance, b: Instance) => {
      return b.createdTs - a.createdTs;
    });
  },

  instanceListByEnvironmentId: (state: InstanceState, getters: any) => (
    environmentId: EnvironmentId
  ) => {
    const list = getters["instanceList"]();
    return list.filter(
      (item: Instance) => item.environment.id == environmentId
    );
  },

  instanceById: (state: InstanceState) => (instanceId: InstanceId) => {
    return state.instanceById.get(instanceId);
  },
};

const actions = {
  async fetchInstanceList({ commit, rootGetters }: any) {
    const instanceList = (await axios.get(`/api/instance`)).data.data.map(
      (instance: ResourceObject) => {
        return convert(instance, rootGetters);
      }
    );

    commit("setInstanceList", instanceList);

    return instanceList;
  },

  async fetchInstanceById(
    { commit, rootGetters }: any,
    instanceId: InstanceId
  ) {
    const instance = convert(
      (await axios.get(`/api/instance/${instanceId}`)).data.data,
      rootGetters
    );

    commit("setInstanceById", {
      instanceId,
      instance,
    });
    return instance;
  },

  async createInstance({ commit, rootGetters }: any, newInstance: InstanceNew) {
    const createdInstance = convert(
      (
        await axios.post(`/api/instance`, {
          data: {
            type: "instance",
            attributes: newInstance,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("setInstanceById", {
      instanceId: createdInstance.id,
      instance: createdInstance,
    });

    return createdInstance;
  },

  async patchInstance({ commit, rootGetters }: any, instance: Instance) {
    const { id, ...attrs } = instance;
    const updatedInstance = convert(
      (
        await axios.patch(`/api/instance/${instance.id}`, {
          data: {
            type: "instance",
            attributes: attrs,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("setInstanceById", {
      instanceId: updatedInstance.id,
      instance: updatedInstance,
    });

    return updatedInstance;
  },

  async deleteInstanceById(
    { commit }: { state: InstanceState; commit: any },
    instanceId: InstanceId
  ) {
    await axios.delete(`/api/instance/${instanceId}`);

    commit("deleteInstanceById", instanceId);
  },
};

const mutations = {
  setInstanceList(state: InstanceState, instanceList: Instance[]) {
    instanceList.forEach((instance) => {
      state.instanceById.set(instance.id, instance);
    });
  },

  setInstanceById(
    state: InstanceState,
    {
      instanceId,
      instance,
    }: {
      instanceId: InstanceId;
      instance: Instance;
    }
  ) {
    state.instanceById.set(instanceId, instance);
  },

  deleteInstanceById(state: InstanceState, instanceId: InstanceId) {
    state.instanceById.delete(instanceId);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
