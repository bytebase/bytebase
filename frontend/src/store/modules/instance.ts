import axios from "axios";
import { UserId, InstanceId, Instance, InstanceState } from "../../types";

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
  async fetchInstanceList({ commit }: any) {
    const instanceList = (await axios.get(`/api/instance`)).data.data;

    commit("setInstanceList", { instanceList });

    return instanceList;
  },

  async fetchInstanceById({ commit }: any, instanceId: InstanceId) {
    const instance = (await axios.get(`/api/instance/${instanceId}`)).data.data;
    commit("setInstanceById", {
      instanceId,
      instance,
    });
    return instance;
  },

  async createInstance(
    { commit }: any,
    { newInstance }: { newInstance: Instance }
  ) {
    const createdInstance = (
      await axios.post(`/api/instance`, {
        data: newInstance,
      })
    ).data.data;

    commit("appendInstance", {
      newInstance: createdInstance,
    });

    return createdInstance;
  },

  async patchInstanceById(
    { commit }: any,
    {
      instanceId,
      instance,
    }: {
      instanceId: InstanceId;
      instance: Instance;
    }
  ) {
    const updatedInstance = (
      await axios.patch(`/api/instance/${instanceId}`, {
        data: instance,
      })
    ).data.data;

    commit("replaceInstanceInList", {
      updatedInstance,
    });

    return updatedInstance;
  },

  async deleteInstanceById(
    { state, commit }: { state: InstanceState; commit: any },
    {
      id,
    }: {
      id: InstanceId;
    }
  ) {
    await axios.delete(`/api/instance/${id}`);

    const newList = state.instanceList.filter((item: Instance) => {
      return item.id != id;
    });

    commit("setInstanceList", {
      instanceList: newList,
    });
  },
};

const mutations = {
  setInstanceList(
    state: InstanceState,
    {
      instanceList,
    }: {
      instanceList: Instance[];
    }
  ) {
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
  appendInstance(
    state: InstanceState,
    {
      newInstance,
    }: {
      newInstance: Instance;
    }
  ) {
    state.instanceList.push(newInstance);
  },

  replaceInstanceInList(
    state: InstanceState,
    {
      updatedInstance,
    }: {
      updatedInstance: Instance;
    }
  ) {
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
