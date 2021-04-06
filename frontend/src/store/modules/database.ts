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
  DataSourceMember,
} from "../../types";

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
    const list: Database[] = [];
    for (let [_, databaseList] of state.databaseListByInstanceId) {
      databaseList.forEach((item: Database) => {
        if (item.ownerId == userId) {
          list.push(item);
        } else {
          for (const dataSource of item.dataSourceList) {
            if (
              dataSource.memberList.find(
                (member: DataSourceMember) => member.principal.id == userId
              )
            ) {
              list.push(item);
              break;
            }
          }
        }
      });
    }
    return list;
  },

  databaseListByEnvironmentId: (state: DatabaseState) => (
    environmentId: EnvironmentId
  ): Database[] => {
    const list: Database[] = [];
    for (let [_, databaseList] of state.databaseListByInstanceId) {
      databaseList.forEach((item: Database) => {
        if (item.instance.environment.id == environmentId) {
          list.push(item);
        }
      });
    }
    return list;
  },

  // If caller provides scoped search in any of accepted idParams, we search that first.
  // If none is found, we then do an exhaustive search.
  // We have to do this because we store the fetched database info differently based on
  // how is requested.
  databaseById: (state: DatabaseState) => (
    databaseId: DatabaseId,
    instanceId?: InstanceId
  ): Database => {
    if (instanceId) {
      const list = state.databaseListByInstanceId.get(instanceId) || [];
      return (
        list.find((item) => item.id == databaseId) ||
        (unknown("DATABASE") as Database)
      );
    }

    for (let [_, list] of state.databaseListByInstanceId) {
      const database = list.find((item) => item.id == databaseId);
      if (database) {
        return database;
      }
    }

    return unknown("DATABASE") as Database;
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

    commit("upsertDatabaseList", { databaseList, instanceId });

    return databaseList;
  },

  async fetchDatabaseListByUser({ commit, rootGetters }: any, userId: UserId) {
    const data = (
      await axios.get(
        `/api/database?user=${userId}&include=instance,dataSource`
      )
    ).data;
    const databaseList = data.data.map((database: ResourceObject) => {
      return convert(database, data.included, rootGetters);
    });

    commit("upsertDatabaseList", { databaseList });

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

    commit("upsertDatabaseList", { databaseList });

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

    commit("upsertDatabaseList", {
      databaseList: [database],
      instanceId,
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

    commit("upsertDatabaseList", {
      databaseList: [createdDatabase],
      instanceId: createdDatabase.instance.id,
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

    commit("upsertDatabaseList", {
      databaseList: [updatedDatabase],
      instanceId: updatedDatabase.instance.id,
    });

    return updatedDatabase;
  },
};

const mutations = {
  upsertDatabaseList(
    state: DatabaseState,
    {
      databaseList,
      instanceId,
    }: {
      databaseList: Database[];
      instanceId?: InstanceId;
    }
  ) {
    if (instanceId) {
      state.databaseListByInstanceId.set(instanceId, databaseList);
    } else {
      for (const database of databaseList) {
        const list = state.databaseListByInstanceId.get(database.instance.id);
        if (list) {
          const i = list.findIndex((item: Database) => item.id == database.id);
          if (i != -1) {
            list[i] = database;
          } else {
            list.push(database);
          }
        } else {
          state.databaseListByInstanceId.set(database.instance.id, [database]);
        }
      }
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
