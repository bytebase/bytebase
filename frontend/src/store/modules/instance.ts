import axios from "axios";
import {
  InstanceId,
  Instance,
  InstanceNew,
  InstanceState,
  ResourceObject,
  Environment,
} from "../../types";

function convert(instance: ResourceObject, rootGetters: any): Instance {
  const environment = rootGetters["environment/environmentList"]().find(
    (env: Environment) => env.id == instance.attributes.environmentId
  );
  return {
    id: instance.id,
    environment,
    ...(instance.attributes as Omit<Instance, "id" | "environment">),
  };
}

const state: () => InstanceState = () => ({
  instanceList: [],
  instanceById: new Map(),
});

const getters = {
  instanceList: (state: InstanceState) => () => {
    return state.instanceList;
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

    commit("appendInstance", createdInstance);

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

    commit("replaceInstanceInList", updatedInstance);

    return updatedInstance;
  },

  async deleteInstanceById(
    { state, commit }: { state: InstanceState; commit: any },
    instanceId: InstanceId
  ) {
    await axios.delete(`/api/instance/${instanceId}`);

    const newList = state.instanceList.filter((item: Instance) => {
      return item.id != instanceId;
    });

    commit("setInstanceList", newList);
  },
};

const mutations = {
  setInstanceList(state: InstanceState, instanceList: Instance[]) {
    state.instanceList = instanceList;
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

  appendInstance(state: InstanceState, newInstance: Instance) {
    state.instanceList.push(newInstance);
  },

  replaceInstanceInList(state: InstanceState, updatedInstance: Instance) {
    const i = state.instanceList.findIndex(
      (item: Instance) => item.id == updatedInstance.id
    );
    if (i != -1) {
      state.instanceList[i] = updatedInstance;
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
