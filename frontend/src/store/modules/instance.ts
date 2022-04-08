import axios from "axios";
import {
  Anomaly,
  DataSource,
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
  INSTANCE_OPERATION_TIMEOUT,
  MigrationHistory,
  MigrationHistoryId,
  ResourceIdentifier,
  ResourceObject,
  RowStatus,
  SqlResultSet,
  unknown,
} from "../../types";
import { InstanceUser } from "../../types/InstanceUser";
import { useEnvironmentStore } from "../pinia-modules";
import { getPrincipalFromIncludedList } from "./principal";
import { useAnomalyStore } from "@/store";

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

  const anomalyIdList = instance.relationships!.anomalyList
    .data as ResourceIdentifier[];
  const anomalyList: Anomaly[] = [];
  for (const item of anomalyIdList) {
    const anomaly = unknown("ANOMALY") as Anomaly;
    anomaly.id = parseInt(item.id);
    anomalyList.push(anomaly);
  }

  const dataSourceIdList = instance.relationships!.dataSourceList
    .data as ResourceIdentifier[];
  const dataSourceList: DataSource[] = [];
  for (const item of dataSourceIdList) {
    const dataSource = unknown("DATA_SOURCE") as DataSource;
    dataSource.id = parseInt(item.id);
    dataSourceList.push(dataSource);
  }

  const instancePartial = {
    ...(instance.attributes as Omit<
      Instance,
      | "id"
      | "environment"
      | "anomalyList"
      | "dataSourceList"
      | "creator"
      | "updater"
    >),
    id: parseInt(instance.id),
    creator: getPrincipalFromIncludedList(
      instance.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      instance.relationships!.updater.data,
      includedList
    ),
    environment,
    anomalyList: [],
    dataSourceList: [],
  };

  const environmentStore = useEnvironmentStore();
  for (const item of includedList || []) {
    if (
      item.type == "environment" &&
      (instance.relationships!.environment.data as ResourceIdentifier).id ==
        item.id
    ) {
      environment = environmentStore.convert(item, includedList);
    }

    if (
      item.type == "anomaly" &&
      item.attributes.instanceId == instancePartial.id
    ) {
      const i = anomalyList.findIndex(
        (anomaly: Anomaly) => parseInt(item.id) == anomaly.id
      );
      if (i != -1) {
        anomalyList[i] = useAnomalyStore().convert(item);
        anomalyList[i].instance = instancePartial;
      }
    }

    if (
      item.type == "dataSource" &&
      item.attributes.instanceId == instancePartial.id
    ) {
      const i = dataSourceList.findIndex(
        (dataSource: DataSource) => parseInt(item.id) == dataSource.id
      );
      if (i != -1) {
        dataSourceList[i] = rootGetters["dataSource/convert"](item);
      }
    }
  }

  for (const anomaly of anomalyList) {
    anomaly.instance.environment = environment;
  }

  return {
    ...(instancePartial as Omit<
      Instance,
      "environment" | "anomalyList" | "dataSourceList"
    >),
    environment,
    anomalyList,
    dataSourceList,
  };
}

function convertUser(
  instanceUser: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): InstanceUser {
  return {
    ...(instanceUser.attributes as Omit<InstanceUser, "id">),
    id: parseInt(instanceUser.id),
  };
}

function convertMigrationHistory(history: ResourceObject): MigrationHistory {
  const payload = history.attributes.payload
    ? JSON.parse((history.attributes.payload as string) || "{}")
    : {};
  return {
    ...(history.attributes as Omit<
      MigrationHistory,
      "id" | "issueId" | "payload"
    >),
    id: parseInt(history.id),
    // This issueId is special since it's stored in the migration history table
    // and may refer to the issueId from the external system in the future.
    issueId: parseInt(history.attributes.issueId as string),
    payload,
  };
}

const state: () => InstanceState = () => ({
  instanceById: new Map(),
  instanceUserListById: new Map(),
  migrationHistoryById: new Map(),
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

  instanceUserListById:
    (state: InstanceState) =>
    (instanceId: InstanceId): InstanceUser[] => {
      return state.instanceUserListById.get(instanceId) || [];
    },

  // Get the formated engine string from instance for SQL transformer.
  instanceFormatedEngine:
    () =>
    (instance: Instance): string => {
      switch (instance.engine) {
        case "POSTGRES":
          return "PostgreSQL";
        // Use MySQL as default engine.
        default:
          return "MySQL";
      }
    },

  migrationHistoryById:
    (state: InstanceState) =>
    (migrationHistoryId: MigrationHistoryId): MigrationHistory | undefined => {
      return state.migrationHistoryById.get(migrationHistoryId);
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
      await axios.post(
        `/api/instance`,
        {
          data: {
            type: "InstanceCreate",
            attributes: newInstance,
          },
        },
        {
          timeout: INSTANCE_OPERATION_TIMEOUT,
        }
      )
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
      await axios.patch(
        `/api/instance/${instanceId}`,
        {
          data: {
            type: "instancePatch",
            attributes: instancePatch,
          },
        },
        {
          timeout: INSTANCE_OPERATION_TIMEOUT,
        }
      )
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

  async fetchInstanceUserListById(
    { commit, rootGetters }: any,
    instanceId: InstanceId
  ) {
    const data = (await axios.get(`/api/instance/${instanceId}/user`)).data;
    const instanceUserList = data.data.map((instanceUser: ResourceObject) => {
      return convertUser(instanceUser, data.included, rootGetters);
    });

    commit("setInstanceUserListById", {
      instanceId,
      instanceUserList,
    });
    return instanceUserList;
  },

  async checkMigrationSetup(
    {}: any,
    instanceId: InstanceId
  ): Promise<InstanceMigration> {
    const data = (
      await axios.get(`/api/instance/${instanceId}/migration/status`, {
        timeout: INSTANCE_OPERATION_TIMEOUT,
      })
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
    const data = (
      await axios.post(`/api/instance/${instanceId}/migration`, undefined, {
        timeout: INSTANCE_OPERATION_TIMEOUT,
      })
    ).data.data;

    return rootGetters["sql/convert"](data);
  },

  async fetchMigrationHistoryById(
    { commit, rootGetters }: any,
    {
      instanceId,
      migrationHistoryId,
    }: {
      instanceId: InstanceId;
      migrationHistoryId: MigrationHistoryId;
    }
  ) {
    const data = (
      await axios.get(
        `/api/instance/${instanceId}/migration/history/${migrationHistoryId}`,
        {
          timeout: INSTANCE_OPERATION_TIMEOUT,
        }
      )
    ).data;
    const migrationHistory = convertMigrationHistory(data.data);

    commit("setMigrationHistoryById", {
      migrationHistoryId,
      migrationHistory,
    });
    return migrationHistory;
  },

  async fetchMigrationHistoryByVersion(
    { commit }: any,
    {
      instanceId,
      databaseName,
      version,
    }: {
      instanceId: InstanceId;
      databaseName: string;
      version: string;
    }
  ) {
    const data = (
      await axios.get(
        `/api/instance/${instanceId}/migration/history?database=${databaseName}&version=${version}`,
        {
          timeout: INSTANCE_OPERATION_TIMEOUT,
        }
      )
    ).data.data;
    const historyList = data.map((history: ResourceObject) => {
      return convertMigrationHistory(history);
    });

    if (historyList.length > 0) {
      commit("setMigrationHistoryById", {
        migrationHistoryId: historyList[0].id,
        migrationHistory: historyList[0],
      });
      return historyList[0];
    }
    throw new Error(
      `Migration version ${version} not found in database ${databaseName}.`
    );
  },

  async fetchMigrationHistory(
    { commit }: any,
    {
      instanceId,
      databaseName,
      limit,
    }: {
      instanceId: InstanceId;
      databaseName: string;
      limit?: number;
    }
  ): Promise<MigrationHistory> {
    let url = `/api/instance/${instanceId}/migration/history?database=${databaseName}`;
    if (limit) {
      url += `&limit=${limit}`;
    }
    const data = (
      await axios.get(url, {
        timeout: INSTANCE_OPERATION_TIMEOUT,
      })
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

  setInstanceUserListById(
    state: InstanceState,
    {
      instanceId,
      instanceUserList,
    }: {
      instanceId: InstanceId;
      instanceUserList: InstanceUser[];
    }
  ) {
    state.instanceUserListById.set(instanceId, instanceUserList);
  },

  setMigrationHistoryById(
    state: InstanceState,
    {
      migrationHistoryId,
      migrationHistory,
    }: {
      migrationHistoryId: MigrationHistoryId;
      migrationHistory: MigrationHistory;
    }
  ) {
    state.migrationHistoryById.set(migrationHistoryId, migrationHistory);
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
