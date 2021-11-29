import axios from "axios";
import {
  Anomaly,
  empty,
  EMPTY_ID,
  Environment,
  EnvironmentID,
  Instance,
  InstanceCreate,
  InstanceID,
  InstanceMigration,
  InstancePatch,
  InstanceState,
  INSTANCE_OPERATION_TIMEOUT,
  MigrationHistory,
  MigrationHistoryID,
  ResourceIdentifier,
  ResourceObject,
  RowStatus,
  SqlResultSet,
  unknown,
} from "../../types";
import { InstanceUser } from "../../types/InstanceUser";

function convert(
  instance: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Instance {
  const environmentID = (
    instance.relationships!.environment.data as ResourceIdentifier
  ).id;
  let environment: Environment = unknown("ENVIRONMENT") as Environment;
  environment.id = parseInt(environmentID);

  const anomalyIDList = instance.relationships!.anomaly
    .data as ResourceIdentifier[];
  const anomalyList: Anomaly[] = [];
  for (const item of anomalyIDList) {
    const anomaly = unknown("ANOMALY") as Anomaly;
    anomaly.id = parseInt(item.id);
    anomalyList.push(anomaly);
  }

  const instancePartial = {
    ...(instance.attributes as Omit<
      Instance,
      "id" | "environment" | "anomalyList"
    >),
    id: parseInt(instance.id),
    environment,
    anomalyList: [],
  };

  for (const item of includedList || []) {
    if (
      item.type == "environment" &&
      (instance.relationships!.environment.data as ResourceIdentifier).id ==
        item.id
    ) {
      environment = rootGetters["environment/convert"](item, includedList);
    }

    if (
      item.type == "anomaly" &&
      item.attributes.instanceID == instancePartial.id
    ) {
      const i = anomalyList.findIndex(
        (anomaly: Anomaly) => parseInt(item.id) == anomaly.id
      );
      if (i != -1) {
        anomalyList[i] = rootGetters["anomaly/convert"](item);
        anomalyList[i].instance = instancePartial;
      }
    }
  }

  for (const anomaly of anomalyList) {
    anomaly.instance.environment = environment;
  }

  return {
    ...(instancePartial as Omit<
      Instance,
      "environment" | "password" | "anomalyList"
    >),
    environment,
    // Password is not returned by the server, we just take extra caution here to redact it.
    password: "",
    anomalyList,
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
      "id" | "issueID" | "payload"
    >),
    id: parseInt(history.id),
    // This issueID is special since it's stored in the migration history table
    // and may refer to the issueID from the external system in the future.
    issueID: parseInt(history.attributes.issueID as string),
    payload,
  };
}

const state: () => InstanceState = () => ({
  instanceByID: new Map(),
  instanceUserListByID: new Map(),
  migrationHistoryByID: new Map(),
  migrationHistoryListByIDAndDatabaseName: new Map(),
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
      for (const [_, instance] of state.instanceByID) {
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

  instanceListByEnvironmentID:
    (state: InstanceState, getters: any) =>
    (environmentID: EnvironmentID, rowStatusList?: RowStatus[]): Instance[] => {
      const list = getters["instanceList"](rowStatusList);
      return list.filter((item: Instance) => {
        return item.environment.id == environmentID;
      });
    },

  instanceByID:
    (state: InstanceState) =>
    (instanceID: InstanceID): Instance => {
      if (instanceID == EMPTY_ID) {
        return empty("INSTANCE") as Instance;
      }

      return (
        state.instanceByID.get(instanceID) || (unknown("INSTANCE") as Instance)
      );
    },

  instanceUserListByID:
    (state: InstanceState) =>
    (instanceID: InstanceID): InstanceUser[] => {
      return state.instanceUserListByID.get(instanceID) || [];
    },

  migrationHistoryByID:
    (state: InstanceState) =>
    (migrationHistoryID: MigrationHistoryID): MigrationHistory | undefined => {
      return state.migrationHistoryByID.get(migrationHistoryID);
    },

  migrationHistoryListByInstanceIDAndDatabaseName:
    (state: InstanceState) =>
    (instanceID: InstanceID, databaseName: string): MigrationHistory[] => {
      return (
        state.migrationHistoryListByIDAndDatabaseName.get(
          [instanceID, databaseName].join(".")
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

  async fetchInstanceByID(
    { commit, rootGetters }: any,
    instanceID: InstanceID
  ) {
    const data = (await axios.get(`/api/instance/${instanceID}`)).data;
    const instance = convert(data.data, data.included, rootGetters);

    commit("setInstanceByID", {
      instanceID,
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

    commit("setInstanceByID", {
      instanceID: createdInstance.id,
      instance: createdInstance,
    });

    return createdInstance;
  },

  async patchInstance(
    { commit, rootGetters }: any,
    {
      instanceID,
      instancePatch,
    }: {
      instanceID: InstanceID;
      instancePatch: InstancePatch;
    }
  ) {
    const data = (
      await axios.patch(
        `/api/instance/${instanceID}`,
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

    commit("setInstanceByID", {
      instanceID: updatedInstance.id,
      instance: updatedInstance,
    });

    return updatedInstance;
  },

  async deleteInstanceByID(
    { commit }: { state: InstanceState; commit: any },
    instanceID: InstanceID
  ) {
    await axios.delete(`/api/instance/${instanceID}`);

    commit("deleteInstanceByID", instanceID);
  },

  async fetchInstanceUserListByID(
    { commit, rootGetters }: any,
    instanceID: InstanceID
  ) {
    const data = (await axios.get(`/api/instance/${instanceID}/user`)).data;
    const instanceUserList = data.data.map((instanceUser: ResourceObject) => {
      return convertUser(instanceUser, data.included, rootGetters);
    });

    commit("setInstanceUserListByID", {
      instanceID,
      instanceUserList,
    });
    return instanceUserList;
  },

  async checkMigrationSetup(
    {}: any,
    instanceID: InstanceID
  ): Promise<InstanceMigration> {
    const data = (
      await axios.get(`/api/instance/${instanceID}/migration/status`, {
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
    instanceID: InstanceID
  ): Promise<SqlResultSet> {
    const data = (
      await axios.post(`/api/instance/${instanceID}/migration`, undefined, {
        timeout: INSTANCE_OPERATION_TIMEOUT,
      })
    ).data.data;

    return rootGetters["sql/convert"](data);
  },

  async fetchMigrationHistoryByID(
    { commit, rootGetters }: any,
    {
      instanceID,
      migrationHistoryID,
    }: {
      instanceID: InstanceID;
      migrationHistoryID: MigrationHistoryID;
    }
  ) {
    const data = (
      await axios.get(
        `/api/instance/${instanceID}/migration/history/${migrationHistoryID}`,
        {
          timeout: INSTANCE_OPERATION_TIMEOUT,
        }
      )
    ).data;
    const migrationHistory = convertMigrationHistory(data.data);

    commit("setMigrationHistoryByID", {
      migrationHistoryID,
      migrationHistory,
    });
    return migrationHistory;
  },

  async fetchMigrationHistoryByVersion(
    { commit }: any,
    {
      instanceID,
      databaseName,
      version,
    }: {
      instanceID: InstanceID;
      databaseName: string;
      version: string;
    }
  ) {
    const data = (
      await axios.get(
        `/api/instance/${instanceID}/migration/history?database=${databaseName}&version=${version}`,
        {
          timeout: INSTANCE_OPERATION_TIMEOUT,
        }
      )
    ).data.data;
    const historyList = data.map((history: ResourceObject) => {
      return convertMigrationHistory(history);
    });

    if (historyList.length > 0) {
      commit("setMigrationHistoryByID", {
        migrationHistoryID: historyList[0].id,
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
      instanceID,
      databaseName,
      limit,
    }: {
      instanceID: InstanceID;
      databaseName: string;
      limit?: number;
    }
  ): Promise<MigrationHistory> {
    let url = `/api/instance/${instanceID}/migration/history?database=${databaseName}`;
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

    commit("setMigrationHistoryListByInstanceIDAndDatabaseName", {
      instanceID,
      databaseName,
      historyList,
    });

    return historyList;
  },
};

const mutations = {
  setInstanceList(state: InstanceState, instanceList: Instance[]) {
    instanceList.forEach((instance) => {
      state.instanceByID.set(instance.id, instance);
    });
  },

  setInstanceByID(
    state: InstanceState,
    {
      instanceID,
      instance,
    }: {
      instanceID: InstanceID;
      instance: Instance;
    }
  ) {
    state.instanceByID.set(instanceID, instance);
  },

  deleteInstanceByID(state: InstanceState, instanceID: InstanceID) {
    state.instanceByID.delete(instanceID);
  },

  setInstanceUserListByID(
    state: InstanceState,
    {
      instanceID,
      instanceUserList,
    }: {
      instanceID: InstanceID;
      instanceUserList: InstanceUser[];
    }
  ) {
    state.instanceUserListByID.set(instanceID, instanceUserList);
  },

  setMigrationHistoryByID(
    state: InstanceState,
    {
      migrationHistoryID,
      migrationHistory,
    }: {
      migrationHistoryID: MigrationHistoryID;
      migrationHistory: MigrationHistory;
    }
  ) {
    state.migrationHistoryByID.set(migrationHistoryID, migrationHistory);
  },

  setMigrationHistoryListByInstanceIDAndDatabaseName(
    state: InstanceState,
    {
      instanceID,
      databaseName,
      historyList,
    }: {
      instanceID: InstanceID;
      databaseName: string;
      historyList: MigrationHistory[];
    }
  ) {
    state.migrationHistoryListByIDAndDatabaseName.set(
      [instanceID, databaseName].join("."),
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
