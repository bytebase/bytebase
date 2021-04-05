import axios from "axios";
import {
  UserId,
  Database,
  DatabaseNew,
  DatabaseId,
  Instance,
  InstanceId,
  DatabaseState,
  ResourceObject,
  ResourceIdentifier,
  EnvironmentId,
  PrincipalId,
  unknown,
  DataSource,
} from "../../types";

type IDParams = {
  instanceId?: InstanceId;
  userId?: UserId;
  environmentId?: EnvironmentId;
};

function convert(
  database: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Database {
  const instanceId = (database.relationships!.instance
    .data as ResourceIdentifier).id;
  let instance: Instance = unknown("INSTANCE") as Instance;

  const dataSourceList: DataSource[] = [];
  for (const item of includedList) {
    if (item.type == "instance" && item.id == instanceId) {
      instance = rootGetters["instance/convert"](item);
    }
    if (
      item.type == "data-source" &&
      (item.relationships!.database.data as ResourceIdentifier).id ==
        database.id
    ) {
      dataSourceList.push(rootGetters["dataSource/convert"](item));
    }
  }

  return {
    id: database.id,
    instance,
    dataSourceList,
    ...(database.attributes as Omit<
      Database,
      "id" | "instance" | "dataSourceList"
    >),
  };
}

const state: () => DatabaseState = () => ({
  databaseListByInstanceId: new Map(),
  databaseListByUserId: new Map(),
  databaseListByEnvironmentId: new Map(),
});

const getters = {
  convert: (
    state: DatabaseState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (database: ResourceObject, inlcudedList: ResourceObject[]): Database => {
    return convert(database, [], rootGetters);
  },

  databaseListByInstanceId: (state: DatabaseState) => (
    instanceId: InstanceId
  ): Database[] => {
    return state.databaseListByInstanceId.get(instanceId) || [];
  },

  databaseListByUserId: (state: DatabaseState) => (
    userId: UserId
  ): Database[] => {
    return state.databaseListByUserId.get(userId) || [];
  },

  databaseListByEnvironmentId: (state: DatabaseState) => (
    environmentId: EnvironmentId
  ): Database[] => {
    return state.databaseListByEnvironmentId.get(environmentId) || [];
  },

  // If caller provides scoped search in any of accepted idParams, we search that first.
  // If none is found, we then do an exhaustive search.
  // We have to do this because we store the fetched database info differently based on
  // how is requested.
  databaseById: (state: DatabaseState) => (
    databaseId: DatabaseId,
    idParams?: IDParams
  ): Database => {
    let database = undefined;
    if (idParams?.instanceId) {
      const list =
        state.databaseListByInstanceId.get(idParams?.instanceId) || [];
      database = list.find((item) => item.id == databaseId);
      if (database) {
        return database;
      }
    }
    if (idParams?.userId) {
      const list = state.databaseListByUserId.get(idParams?.userId) || [];
      database = list.find((item) => item.id == databaseId);
      if (database) {
        return database;
      }
    }

    if (idParams?.environmentId) {
      const list =
        state.databaseListByEnvironmentId.get(idParams?.environmentId) || [];
      database = list.find((item) => item.id == databaseId);
      if (database) {
        return database;
      }
    }

    for (let [_, list] of state.databaseListByInstanceId) {
      database = list.find((item) => item.id == databaseId);
      if (database) {
        return database;
      }
    }

    for (let [_, list] of state.databaseListByUserId) {
      database = list.find((item) => item.id == databaseId);
      if (database) {
        return database;
      }
    }

    for (let [_, list] of state.databaseListByEnvironmentId) {
      database = list.find((item) => item.id == databaseId);
      if (database) {
        return database;
      }
    }

    const ts = Date.now();
    return {
      id: "-1",
      name: "<<Unknown database>>",
      createdTs: 0,
      lastUpdatedTs: 0,
      ownerId: "-1",
      syncStatus: "NOT_FOUND",
      lastSuccessfulSyncTs: ts,
      fingerprint: "",
      instance: {
        id: "-1",
        name: "<<Unknown instance>>",
        environment: {
          id: "-1",
          name: "<<Unknown environment>>",
          order: -1,
        },
        host: "",
        createdTs: ts,
        lastUpdatedTs: ts,
      },
      dataSourceList: [],
    };
  },
};

const actions = {
  async fetchDatabaseListByInstanceId(
    { commit, rootGetters }: any,
    instanceId: InstanceId
  ) {
    const data = (
      await axios.get(
        `/api/database?instance=${instanceId}&include=instance,dataSource`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("setDatabaseListByInstanceId", { instanceId, databaseList });

    return databaseList;
  },

  async fetchDatabaseListByUser({ commit, rootGetters }: any, userId: UserId) {
    const data = (
      await axios.get(
        `/api/user/${userId}/database?include=instance,dataSource`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("setDatabaseListByUserId", { userId, databaseList });

    return databaseList;
  },

  async fetchDatabaseListByEnvironmentId(
    { commit, rootGetters }: any,
    environmentId: EnvironmentId
  ) {
    const data = (
      await axios.get(
        `/api/database?environment=${environmentId}&include=instance,dataSource`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("setDatabaseListByEnvironmentId", { environmentId, databaseList });

    return databaseList;
  },

  async fetchDatabaseById(
    { commit, rootGetters }: any,
    {
      databaseId,
      instanceId,
    }: { databaseId: DatabaseId; instanceId?: InstanceId }
  ) {
    const url = instanceId
      ? `/api/instance/${instanceId}/database/${databaseId}?include=instance,dataSource`
      : `/api/database/${databaseId}?include=instance,dataSource`;
    const data = (await axios.get(url)).data;
    const database = convert(data.data, data.included, rootGetters);

    commit("upsertDatabaseInListByInstanceId", {
      instanceId,
      database,
    });

    return database;
  },

  async createDatabase({ commit, rootGetters }: any, newDatabase: DatabaseNew) {
    const data = (
      await axios.post(`/api/database?include=instance,dataSource`, {
        data: {
          type: "database",
          attributes: newDatabase,
        },
      })
    ).data;
    const createdDatabase: Database = convert(
      data.data,
      data.included,
      rootGetters
    );

    commit("upsertDatabaseInListByInstanceId", {
      instanceId: createdDatabase.instance.id,
      database: createdDatabase,
    });

    return createdDatabase;
  },

  async updateOwner(
    { commit, rootGetters }: any,
    {
      instanceId,
      databaseId,
      ownerId,
    }: {
      instanceId: InstanceId;
      databaseId: DatabaseId;
      ownerId: PrincipalId;
    }
  ) {
    const data = (
      await axios.patch(
        `/api/database/${databaseId}?include=instance,dataSource`,
        {
          data: {
            type: "databasepatch",
            attributes: {
              ownerId,
            },
          },
        }
      )
    ).data;
    const updatedDatabase = convert(data.data, data.included, rootGetters);

    commit("upsertDatabaseInListByInstanceId", {
      instanceId,
      database: updatedDatabase,
    });

    return updatedDatabase;
  },
};

const mutations = {
  setDatabaseListByInstanceId(
    state: DatabaseState,
    {
      instanceId,
      databaseList,
    }: {
      instanceId: InstanceId;
      databaseList: Database[];
    }
  ) {
    state.databaseListByInstanceId.set(instanceId, databaseList);
  },

  upsertDatabaseInListByInstanceId(
    state: DatabaseState,
    {
      instanceId,
      database,
    }: {
      instanceId: InstanceId;
      database: Database;
    }
  ) {
    const list = state.databaseListByInstanceId.get(instanceId);
    if (list) {
      const i = list.findIndex((item: Database) => item.id == database.id);
      if (i != -1) {
        list[i] = database;
      } else {
        list.push(database);
      }
    } else {
      state.databaseListByInstanceId.set(instanceId, [database]);
    }
  },

  setDatabaseListByUserId(
    state: DatabaseState,
    {
      userId,
      databaseList,
    }: {
      userId: UserId;
      databaseList: Database[];
    }
  ) {
    state.databaseListByUserId.set(userId, databaseList);
  },

  setDatabaseListByEnvironmentId(
    state: DatabaseState,
    {
      environmentId,
      databaseList,
    }: {
      environmentId: EnvironmentId;
      databaseList: Database[];
    }
  ) {
    state.databaseListByEnvironmentId.set(environmentId, databaseList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
