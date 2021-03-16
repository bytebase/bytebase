import axios from "axios";
import {
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
});

const getters = {
  databaseListByInstanceId: (state: DatabaseState) => (
    instanceId: InstanceId
  ): Database[] => {
    return state.databaseListByInstanceId.get(instanceId) || [];
  },

  databaseById: (state: DatabaseState) => (
    databaseId: DatabaseId,
    instanceId?: InstanceId
  ): Database => {
    let database = undefined;
    if (instanceId) {
      const list = state.databaseListByInstanceId.get(instanceId) || [];
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
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
