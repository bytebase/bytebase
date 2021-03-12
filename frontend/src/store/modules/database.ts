import axios from "axios";
import {
  Database,
  InstanceId,
  DatabaseState,
  ResourceObject,
} from "../../types";
import { isDevOrDemo, randomString } from "../../utils";

function convert(database: ResourceObject, rootGetters: any): Database {
  const instance = rootGetters["instance/instanceById"](
    database.attributes.instanceId
  );
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
