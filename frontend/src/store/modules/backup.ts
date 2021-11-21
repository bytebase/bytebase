import axios from "axios";
import {
  Backup,
  BackupCreate,
  BackupSetting,
  BackupSettingState,
  BackupSettingUpsert,
  BackupState,
  DatabaseID,
  ResourceObject,
  unknown,
} from "../../types";

function convert(
  backup: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Backup {
  return {
    ...(backup.attributes as Omit<Backup, "id">),
    id: parseInt(backup.id),
  };
}

function convertBackupSetting(
  backupSetting: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): BackupSetting {
  return {
    ...(backupSetting.attributes as Omit<BackupSetting, "id">),
    id: parseInt(backupSetting.id),
  };
}

const state: () => BackupState = () => ({
  backupListByDatabaseID: new Map(),
  backupSettingByDatabaseID: new Map(),
});

const getters = {
  convert:
    (state: BackupState, getters: any, rootState: any, rootGetters: any) =>
    (backup: ResourceObject, includedList: ResourceObject[]): Backup => {
      return convert(backup, includedList || [], rootGetters);
    },

  backupListByDatabaseID:
    (state: BackupState) =>
    (databaseID: DatabaseID): Backup[] => {
      return state.backupListByDatabaseID.get(databaseID) || [];
    },
  backupSettingByDatabaseID:
    (state: BackupSettingState) =>
    (databaseID: DatabaseID): BackupSetting => {
      return (
        state.backupSettingByDatabaseID.get(databaseID) ||
        (unknown("BACKUP_SETTING") as BackupSetting)
      );
    },
};

const actions = {
  async createBackup(
    { commit, rootGetters }: any,
    {
      databaseID,
      newBackup,
    }: { databaseID: DatabaseID; newBackup: BackupCreate }
  ) {
    const data = (
      await axios.post(`/api/database/${newBackup.databaseID}/backup`, {
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

    commit("setBackupByDatabaseIDAndBackupName", {
      databaseID: databaseID,
      backupName: createdBackup.name,
      backup: createdBackup,
    });

    return createdBackup;
  },

  async fetchBackupListByDatabaseID(
    { commit, rootGetters }: any,
    databaseID: DatabaseID
  ) {
    const data = (await axios.get(`/api/database/${databaseID}/backup`)).data;
    const backupList = data.data.map((backup: ResourceObject) => {
      return convert(backup, data.included, rootGetters);
    });

    commit("setTableListByDatabaseID", { databaseID, backupList });
    return backupList;
  },

  async fetchBackupSettingByDatabaseID(
    { commit, rootGetters }: any,
    databaseID: DatabaseID
  ) {
    const data = (await axios.get(`/api/database/${databaseID}/backupsetting`))
      .data;
    const backupSetting: BackupSetting = convertBackupSetting(
      data.data,
      data.included,
      rootGetters
    );

    commit("upsertBackupSettingByDatabaseID", { databaseID, backupSetting });
    return backupSetting;
  },

  async upsertBackupSetting(
    { commit, rootGetters }: any,
    { newBackupSetting }: { newBackupSetting: BackupSettingUpsert }
  ) {
    const data = (
      await axios.patch(
        `/api/database/${newBackupSetting.databaseID}/backupsetting`,
        {
          data: {
            type: "BackupSettingUpsert",
            attributes: newBackupSetting,
          },
        }
      )
    ).data;
    const updatedBackupSetting: BackupSetting = convertBackupSetting(
      data.data,
      data.included,
      rootGetters
    );

    commit("upsertBackupSettingByDatabaseID", {
      databaseID: newBackupSetting.databaseID,
      backup: updatedBackupSetting,
    });

    return updatedBackupSetting;
  },
};

const mutations = {
  setTableListByDatabaseID(
    state: BackupState,
    {
      databaseID,
      backupList,
    }: {
      databaseID: DatabaseID;
      backupList: Backup[];
    }
  ) {
    state.backupListByDatabaseID.set(databaseID, backupList);
  },

  setBackupByDatabaseIDAndBackupName(
    state: BackupState,
    {
      databaseID,
      backupName,
      backup,
    }: {
      databaseID: DatabaseID;
      backupName: string;
      backup: Backup;
    }
  ) {
    const list = state.backupListByDatabaseID.get(databaseID);
    if (list) {
      const i = list.findIndex((item: Backup) => item.name == backupName);
      if (i != -1) {
        list[i] = backup;
      } else {
        list.push(backup);
      }
    } else {
      state.backupListByDatabaseID.set(databaseID, [backup]);
    }
  },

  upsertBackupSettingByDatabaseID(
    state: BackupSettingState,
    {
      databaseID,
      backupSetting,
    }: {
      databaseID: DatabaseID;
      backupSetting: BackupSetting;
    }
  ) {
    state.backupSettingByDatabaseID.set(databaseID, backupSetting);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
