import axios from "axios";
import {
  Backup,
  BackupCreate,
  BackupSetting,
  BackupSettingState,
  BackupSettingUpsert,
  BackupState,
  DatabaseId,
  ResourceObject,
  unknown,
} from "../../types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  backup: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Backup {
  return {
    ...(backup.attributes as Omit<Backup, "id" | "creator" | "updater">),
    id: parseInt(backup.id),
    creator: getPrincipalFromIncludedList(
      backup.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      backup.relationships!.updater.data,
      includedList
    ),
  };
}

function convertBackupSetting(
  backupSetting: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): BackupSetting {
  return {
    ...(backupSetting.attributes as Omit<
      BackupSetting,
      "id" | "creator" | "updater"
    >),
    id: parseInt(backupSetting.id),
    creator: getPrincipalFromIncludedList(
      backupSetting.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      backupSetting.relationships!.updater.data,
      includedList
    ),
  };
}

const state: () => BackupState = () => ({
  backupListByDatabaseId: new Map(),
  backupSettingByDatabaseId: new Map(),
});

const getters = {
  convert:
    (state: BackupState, getters: any, rootState: any, rootGetters: any) =>
    (backup: ResourceObject, includedList: ResourceObject[]): Backup => {
      return convert(backup, includedList || [], rootGetters);
    },

  backupListByDatabaseId:
    (state: BackupState) =>
    (databaseId: DatabaseId): Backup[] => {
      return state.backupListByDatabaseId.get(databaseId) || [];
    },
  backupSettingByDatabaseId:
    (state: BackupSettingState) =>
    (databaseId: DatabaseId): BackupSetting => {
      return (
        state.backupSettingByDatabaseId.get(databaseId) ||
        (unknown("BACKUP_SETTING") as BackupSetting)
      );
    },
};

const actions = {
  async createBackup(
    { commit, rootGetters }: any,
    {
      databaseId,
      newBackup,
    }: { databaseId: DatabaseId; newBackup: BackupCreate }
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
      backup: createdBackup,
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
    const data = (await axios.get(`/api/database/${databaseId}/backup-setting`))
      .data;
    const backupSetting: BackupSetting = convertBackupSetting(
      data.data,
      data.included,
      rootGetters
    );

    commit("upsertBackupSettingByDatabaseId", { databaseId, backupSetting });
    return backupSetting;
  },

  async upsertBackupSetting(
    { commit, rootGetters }: any,
    { newBackupSetting }: { newBackupSetting: BackupSettingUpsert }
  ) {
    const data = (
      await axios.patch(
        `/api/database/${newBackupSetting.databaseId}/backup-setting`,
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

    commit("upsertBackupSettingByDatabaseId", {
      databaseId: newBackupSetting.databaseId,
      backup: updatedBackupSetting,
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

  upsertBackupSettingByDatabaseId(
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
