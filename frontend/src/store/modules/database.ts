import axios from "axios";
import {
  UserId,
  Database,
  DatabaseId,
  InstanceId,
  DatabaseState,
  ResourceObject,
  ResourceIdentifier,
} from "../../types";
import { isDevOrDemo, randomString } from "../../utils";
import instance from "./instance";

function convert(database: ResourceObject, rootGetters: any): Database {
  const instanceId = (database.relationships!.instance
    .data as ResourceIdentifier).id;
  const instance = rootGetters["instance/instanceById"](instanceId);
  return {
    id: database.id,
    instance,
    ...(database.attributes as Omit<Database, "id" | "instance">),
  };
}

const state: () => DatabaseState = () => ({
  databaseListByInstanceId: new Map(),
  databaseListByUserId: new Map(),
});

const getters = {
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

  databaseById: (state: DatabaseState) => (
    databaseId: DatabaseId,
    instanceId?: InstanceId,
    userId?: UserId
  ): Database => {
    let database = undefined;
    if (instanceId) {
      const list = state.databaseListByInstanceId.get(instanceId) || [];
      database = list.find((item) => item.id == databaseId);
    } else if (userId) {
      const list = state.databaseListByUserId.get(userId) || [];
      database = list.find((item) => item.id == databaseId);
    } else {
      for (let [_, list] of state.databaseListByInstanceId) {
        database = list.find((item) => item.id == databaseId);
        if (database) {
          break;
        }
      }
    }
    if (database) {
      return database;
    }

    const ts = Date.now();
    return {
      id: "-1",
      name: "<<Unknown database>>",
      createdTs: 0,
      lastUpdatedTs: 0,
      syncStatus: "NOT_FOUND",
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
    };
  },
};

const actions = {
  async fetchDatabaseListByInstanceId(
    { commit, rootGetters }: any,
    instanceId: InstanceId
  ) {
    const databaseList = (
      await axios.get(`/api/instance/${instanceId}/database`)
    ).data.data.map((database: ResourceObject) => {
      return convert(database, rootGetters);
    });

    commit("setDatabaseListByInstanceId", { instanceId, databaseList });

    return databaseList;
  },

  async fetchDatabaseListByUser({ commit, rootGetters }: any, userId: UserId) {
    const databaseList = (
      await axios.get(`/api/user/${userId}/database`)
    ).data.data.map((database: ResourceObject) => {
      return convert(database, rootGetters);
    });

    commit("setDatabaseListByUserId", { userId, databaseList });

    return databaseList;
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
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
