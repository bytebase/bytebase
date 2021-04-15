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
  unknown,
  InstancePatch,
  RowStatus,
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
  const creator = rootGetters["principal/principalById"](
    instance.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    instance.attributes.updaterId
  );

  return {
    id: instance.id,
    creator,
    updater,
    ...(instance.attributes as Omit<
      Instance,
      "id" | "environment" | "creator" | "updater"
    >),
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

  instanceList: (state: InstanceState) => (
    rowStatus?: RowStatus
  ): Instance[] => {
    const list = [];
    for (const [_, instance] of state.instanceById) {
      if (!rowStatus || rowStatus == instance.rowStatus) {
        list.push(instance);
      }
    }
    return list.sort((a: Instance, b: Instance) => {
      return b.createdTs - a.createdTs;
    });
  },

  instanceListByEnvironmentId: (state: InstanceState, getters: any) => (
    environmentId: EnvironmentId,
    rowStatus?: RowStatus
  ): Instance[] => {
    const list = getters["instanceList"](rowStatus);
    return list.filter((item: Instance) => {
      return item.environment.id == environmentId;
    });
  },

  instanceById: (state: InstanceState) => (
    instanceId: InstanceId
  ): Instance => {
    return (
      state.instanceById.get(instanceId) || (unknown("INSTANCE") as Instance)
    );
  },
};

const actions = {
  async fetchInstanceList({ commit, rootGetters }: any, rowStatus?: RowStatus) {
    const path =
      "/api/instance" +
      (rowStatus ? "?rowstatus=" + rowStatus.toLowerCase() : "");
    const instanceList = (await axios.get(path)).data.data.map(
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
            type: "instancenew",
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

  async patchInstance(
    { commit, rootGetters }: any,
    {
      instanceId,
      instancePatch,
    }: {
      instanceId: InstanceId;
      instancePatch: InstancePatch;
    }
  ) {
    const updatedInstance = convert(
      (
        await axios.patch(`/api/instance/${instanceId}`, {
          data: {
            type: "instancepatch",
            attributes: instancePatch,
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
