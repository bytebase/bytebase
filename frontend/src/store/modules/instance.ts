import axios from "axios";
import {
  empty,
  EMPTY_ID,
  Environment,
  EnvironmentId,
  Instance,
  InstanceCreate,
  InstanceId,
  InstanceMigration,
  InstancePatch,
  InstanceState,
  MigrationHistory,
  ResourceIdentifier,
  ResourceObject,
  RowStatus,
  SqlResultSet,
  unknown,
} from "../../types";

function convert(
  instance: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Instance {
  const environmentId = (
    instance.relationships!.environment.data as ResourceIdentifier
  ).id;
  let environment: Environment = unknown("ENVIRONMENT") as Environment;
  environment.id = parseInt(environmentId);

  for (const item of includedList || []) {
    if (
      item.type == "environment" &&
      (instance.relationships!.environment.data as ResourceIdentifier).id ==
        item.id
    ) {
      environment = rootGetters["environment/convert"](item, includedList);
    }
  }

  return {
    ...(instance.attributes as Omit<Instance, "id" | "environment">),
    id: parseInt(instance.id),
    environment,
    // Password is not returned by the server, we just take extra caution here to redact it.
    password: "",
  };
}

function convertMigrationHistory(history: ResourceObject): MigrationHistory {
  return {
    ...(history.attributes as Omit<MigrationHistory, "id">),
    id: parseInt(history.id),
  };
}

const state: () => InstanceState = () => ({
  instanceById: new Map(),
  migrationHistoryListByIdAndDatabaseName: new Map(),
});

const getters = {
  convert:
    (state: InstanceState, getters: any, rootState: any, rootGetters: any) =>
    (instance: ResourceObject, includedList: ResourceObject[]): Instance => {
      return convert(instance, includedList, rootGetters);
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

  migrationHistoryListByInstanceIdAndDatabaseName:
    (state: InstanceState) =>
    (instanceId: InstanceId, databaseName: string): MigrationHistory[] => {
      return (
        state.migrationHistoryListByIdAndDatabaseName.get(
          [instanceId, databaseName].join(".")
        ) || []
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
    const data = (await axios.get(`/api/instance/${instanceId}`)).data;
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
      await axios.post(`/api/instance`, {
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
      await axios.patch(`/api/instance/${instanceId}`, {
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

  async checkMigrationSetup(
    {}: any,
    instanceId: InstanceId
  ): Promise<InstanceMigration> {
    const data = (
      await axios.get(`/api/instance/${instanceId}/migration/status`)
    ).data.data;

    return {
      status: data.attributes.status,
      error: data.attributes.error,
    };
  },

  async createMigrationSetup(
    { rootGetters }: any,
    instanceId: InstanceId
  ): Promise<SqlResultSet> {
    const data = (await axios.post(`/api/instance/${instanceId}/migration`))
      .data.data;

    return rootGetters["sql/convert"](data);
  },

  async migrationHistory(
    { commit }: any,
    {
      instanceId,
      databaseName,
    }: {
      instanceId: InstanceId;
      databaseName: string;
    }
  ): Promise<MigrationHistory> {
    const data = (
      await axios.get(
        `/api/instance/${instanceId}/migration/history?database=${databaseName}`
      )
    ).data.data;
    const historyList = data.map((history: ResourceObject) => {
      return convertMigrationHistory(history);
    });

    commit("setMigrationHistoryListByInstanceIdAndDatabaseName", {
      instanceId,
      databaseName,
      historyList,
    });

    return historyList;
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

  setMigrationHistoryListByInstanceIdAndDatabaseName(
    state: InstanceState,
    {
      instanceId,
      databaseName,
      historyList,
    }: {
      instanceId: InstanceId;
      databaseName: string;
      historyList: MigrationHistory[];
    }
  ) {
    state.migrationHistoryListByIdAndDatabaseName.set(
      [instanceId, databaseName].join("."),
      historyList
    );
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
