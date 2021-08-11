import axios from "axios";
import {
  BackupCreate,
  BackupId,
  BackupSetting,
  BackupSettingSet,
  Database,
  DatabaseId,
  ResourceIdentifier,
  ResourceObject,
  Backup,
  BackupSettingState,
  BackupState,
  RestoreBackup,
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

function convertBackupSetting(
  backupSetting: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): BackupSetting {
  const databaseId = (backupSetting.relationships!.database.data as ResourceIdentifier)
    .id;

  let database: Database = unknown("DATABASE") as Database;
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseId) {
      database = rootGetters["database/convert"](item, includedList);
      break;
    }
  }
  return {
    ...(backupSetting.attributes as Omit<BackupSetting, "id" | "database">),
    id: parseInt(backupSetting.id),
    database,
  };
}

const state: () => BackupState = () => ({
  backupListByDatabaseId: new Map(),
  backupSettingByDatabaseId: new Map(),
});

const getters = {
  backupListByDatabaseId:
    (state: BackupState) =>
    (databaseId: DatabaseId): Backup[] => {
      return state.backupListByDatabaseId.get(databaseId) || [];
    },
  backupSettingByDatabaseId:
    (state: BackupSettingState) =>
    (databaseId: DatabaseId): BackupSetting => {
      return state.backupSettingByDatabaseId.get(databaseId) || unknown("BACKUP_SETTING") as BackupSetting;
    },
};

const actions = {
  async createBackup(
    { commit, rootGetters }: any,
    { databaseId, newBackup }: { databaseId: DatabaseId; newBackup: BackupCreate }
  ) {
    const data = (
      await axios.post(`/api/database/${newBackup.databaseId}/backup`, {
        data: {
          type: "BackupCreate",
          attributes: newBackup,
        },
      })
    ).data;
    const createdBackup: Backup = convert(
      data.data,
      data.included,
      rootGetters
    );

    commit("setBackupByDatabaseIdAndBackupName", {
      databaseId: databaseId,
      backupName: createdBackup.name,
      backup: createdBackup
    });

    return createdBackup;
  },

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

  async fetchBackupSettingByDatabaseId(
    { commit, rootGetters }: any,
    databaseId: DatabaseId
  ) {
    const data = (await axios.get(`/api/database/${databaseId}/backupSetting`)).data;
    const backupSetting: BackupSetting = convertBackupSetting(
      data.data,
      data.included,
      rootGetters
    );

    commit("setBackupSettingByDatabaseId", { databaseId, backupSetting });
    return backupSetting;
  },

  async restoreFromBackup(
    { commit, rootGetters }: any,
    { databaseId, backupId }: { databaseId: DatabaseId; backupId: BackupId }
  ) {
    const restoreBackup: RestoreBackup = {
      backupId: backupId,
    };
    const data = (
      await axios.post(`/api/database/${databaseId}/restore`, {
        data: {
          type: "RestoreBackup",
          attributes: restoreBackup,
        },
      })
    ).data;
  },

  async setBackupSetting(
    { commit, rootGetters }: any,
    { newBackupSetting }: { newBackupSetting: BackupSettingSet }
  ) {
    const data = (
      await axios.post(`/api/database/${newBackupSetting.databaseId}/backupSetting`, {
        data: {
          type: "BackupSettingSet",
          attributes: newBackupSetting,
        },
      })
    ).data;
    const updatedBackupSetting: BackupSetting = convertBackupSetting(
      data.data,
      data.included,
      rootGetters
    );

    commit("setBackupSettingByDatabaseId", {
      databaseId: newBackupSetting.databaseId,
      backup: updatedBackupSetting
    });

    return updatedBackupSetting;
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

  setBackupByDatabaseIdAndBackupName(
    state: BackupState,
    {
      databaseId,
      backupName,
      backup,
    }: {
      databaseId: DatabaseId;
      backupName: string;
      backup: Backup;
    }
  ) {
    const list = state.backupListByDatabaseId.get(databaseId);
    if (list) {
      const i = list.findIndex((item: Backup) => item.name == backupName);
      if (i != -1) {
        list[i] = backup;
      } else {
        list.push(backup);
      }
    } else {
      state.backupListByDatabaseId.set(databaseId, [backup]);
    }
  },

  setBackupSettingByDatabaseId(
    state: BackupSettingState,
    {
      databaseId,
      backupSetting,
    }: {
      databaseId: DatabaseId;
      backupSetting: BackupSetting;
    }
  ) {
    state.backupSettingByDatabaseId.set(databaseId, backupSetting);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
