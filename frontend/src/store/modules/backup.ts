import axios from "axios";
import {
  Database,
  DatabaseId,
  ResourceIdentifier,
  ResourceObject,
  Backup,
  BackupState,
  unknown,
} from "../../types";

function convert(
  backup: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Backup {
  const databaseId = (backup.relationships!.database.data as ResourceIdentifier)
    .id;

  let database: Database = unknown("DATABASE") as Database;
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseId) {
      database = rootGetters["database/convert"](item, includedList);
      break;
    }
  }
  return {
    ...(backup.attributes as Omit<Backup, "id" | "database">),
    id: parseInt(backup.id),
    database,
  };
}

const state: () => BackupState = () => ({
  backupListByDatabaseId: new Map(),
});

const getters = {
  backupListByDatabaseId:
    (state: BackupState) =>
    (databaseId: DatabaseId): Backup[] => {
      return state.backupListByDatabaseId.get(databaseId) || [];
    },
};

const actions = {
  async fetchBackupListByDatabaseId(
    { commit, rootGetters }: any,
    databaseId: DatabaseId
  ) {
    const data = (await axios.get(`/api/database/${databaseId}/backup`)).data;
    const backupList = data.data.map((backup: ResourceObject) => {
      return convert(backup, data.included, rootGetters);
    });

    commit("setTableListByDatabaseId", { databaseId, backupList });
    return backupList;
  },
};

const mutations = {
  setTableListByDatabaseId(
    state: BackupState,
    {
      databaseId,
      backupList,
    }: {
      databaseId: DatabaseId;
      backupList: Backup[];
    }
  ) {
    state.backupListByDatabaseId.set(databaseId, backupList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
