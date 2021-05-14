import axios from "axios";
import {
  InstanceId,
  Instance,
  InstanceCreate,
  InstanceState,
  ResourceObject,
  Environment,
  EnvironmentId,
  ResourceIdentifier,
  unknown,
  InstancePatch,
  RowStatus,
  empty,
  EMPTY_ID,
} from "../../types";

function convert(
  instance: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Instance {
  const environment = rootGetters["environment/environmentList"]([
    "NORMAL",
    "ARCHIVED",
  ]).find((env: Environment) => {
    return (
      env.id ==
      (instance.relationships!.environment.data as ResourceIdentifier).id
    );
  });
  const creator = rootGetters["principal/principalById"](
    instance.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    instance.attributes.updaterId
  );

  let username = undefined;
  let password = undefined;

  for (const item of includedList || []) {
    if (
      item.type == "data-source" &&
      item.attributes.type == "ADMIN" &&
      (item.relationships!.instance.data as ResourceIdentifier).id ==
        instance.id
    ) {
      username = item.attributes.username as string;
      password = item.attributes.password as string;
      break;
    }
  }

  return {
    ...(instance.attributes as Omit<
      Instance,
      "id" | "environment" | "creator" | "updater"
    >),
    id: instance.id,
    creator,
    updater,
    environment,
    username,
    password,
  };
}

const state: () => InstanceState = () => ({
  instanceById: new Map(),
});

const getters = {
  convert:
    (state: InstanceState, getters: any, rootState: any, rootGetters: any) =>
    (instance: ResourceObject): Instance => {
      return convert(instance, [], rootGetters);
    },

  instanceList:
    (state: InstanceState) =>
    (rowStatusList?: RowStatus[]): Instance[] => {
      const list = [];
      for (const [_, instance] of state.instanceById) {
        if (
          (!rowStatusList && instance.rowStatus == "NORMAL") ||
          (rowStatusList && rowStatusList.includes(instance.rowStatus))
        ) {
          list.push(instance);
        }
      }
      return list.sort((a: Instance, b: Instance) => {
        return b.createdTs - a.createdTs;
      });
    },

  instanceListByEnvironmentId:
    (state: InstanceState, getters: any) =>
    (environmentId: EnvironmentId, rowStatusList?: RowStatus[]): Instance[] => {
      const list = getters["instanceList"](rowStatusList);
      return list.filter((item: Instance) => {
        return item.environment.id == environmentId;
      });
    },

  instanceById:
    (state: InstanceState) =>
    (instanceId: InstanceId): Instance => {
      if (instanceId == EMPTY_ID) {
        return empty("INSTANCE") as Instance;
      }

      return (
        state.instanceById.get(instanceId) || (unknown("INSTANCE") as Instance)
      );
    },
};

const actions = {
  async fetchInstanceList(
    { commit, rootGetters }: any,
    rowStatusList?: RowStatus[]
  ) {
    const path =
      "/api/instance" +
      (rowStatusList ? "?rowstatus=" + rowStatusList.join(",") : "");
    const data = (await axios.get(path)).data;
    const instanceList = data.data.map((instance: ResourceObject) => {
      return convert(instance, data.included, rootGetters);
    });

    commit("setInstanceList", instanceList);

    return instanceList;
  },

  async fetchInstanceById(
    { commit, rootGetters }: any,
    instanceId: InstanceId
  ) {
    // Data source contains sensitive info such as username, password so
    // we only fetch if requesting the specific instance id
    const data = (
      await axios.get(`/api/instance/${instanceId}?include=dataSource`)
    ).data;
    const instance = convert(data.data, data.included, rootGetters);

    commit("setInstanceById", {
      instanceId,
      instance,
    });
    return instance;
  },

  async createInstance(
    { commit, rootGetters }: any,
    newInstance: InstanceCreate
  ) {
    const data = (
      await axios.post(`/api/instance?include=dataSource`, {
        data: {
          type: "InstanceCreate",
          attributes: newInstance,
        },
      })
    ).data;
    const createdInstance = convert(data.data, data.included, rootGetters);

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
    const data = (
      await axios.patch(`/api/instance/${instanceId}?include=dataSource`, {
        data: {
          type: "instancePatch",
          attributes: instancePatch,
        },
      })
    ).data;
    const updatedInstance = convert(data.data, data.included, rootGetters);

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
