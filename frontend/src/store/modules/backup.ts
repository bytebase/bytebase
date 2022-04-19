import { defineStore } from "pinia";
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
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";

export function convertBackup(
  backup: ResourceObject,
  includedList: ResourceObject[]
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
  includedList: ResourceObject[]
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

export const useBackupStore = defineStore("backup", {
  state: (): BackupState & BackupSettingState => ({
    backupList: new Map(),
    backupSetting: new Map(),
  }),

  actions: {
    convert(backup: ResourceObject, includedList: ResourceObject[]): Backup {
      return convertBackup(backup, includedList || []);
    },

    backupListByDatabaseId(databaseId: DatabaseId): Backup[] {
      return this.backupList.get(databaseId) || [];
    },
    backupSettingByDatabaseId(databaseId: DatabaseId): BackupSetting {
      return (
        this.backupSetting.get(databaseId) ||
        (unknown("BACKUP_SETTING") as BackupSetting)
      );
    },

    setTableListByDatabaseId({
      databaseId,
      backupList,
    }: {
      databaseId: DatabaseId;
      backupList: Backup[];
    }) {
      this.backupList.set(databaseId, backupList);
    },

    setBackupByDatabaseIdAndBackupName({
      databaseId,
      backupName,
      backup,
    }: {
      databaseId: DatabaseId;
      backupName: string;
      backup: Backup;
    }) {
      const list = this.backupList.get(databaseId);
      if (list) {
        const i = list.findIndex((item: Backup) => item.name == backupName);
        if (i != -1) {
          list[i] = backup;
        } else {
          list.push(backup);
        }
      } else {
        this.backupList.set(databaseId, [backup]);
      }
    },

    upsertBackupSettingByDatabaseId({
      databaseId,
      backupSetting,
    }: {
      databaseId: DatabaseId;
      backupSetting: BackupSetting;
    }) {
      this.backupSetting.set(databaseId, backupSetting);
    },
    async createBackup({
      databaseId,
      newBackup,
    }: {
      databaseId: DatabaseId;
      newBackup: BackupCreate;
    }) {
      const data = (
        await axios.post(`/api/database/${newBackup.databaseId}/backup`, {
          data: {
            type: "BackupCreate",
            attributes: newBackup,
          },
        })
      ).data;
      const createdBackup: Backup = convertBackup(data.data, data.included);

      this.setBackupByDatabaseIdAndBackupName({
        databaseId: databaseId,
        backupName: createdBackup.name,
        backup: createdBackup,
      });

      return createdBackup;
    },

    async fetchBackupListByDatabaseId(databaseId: DatabaseId) {
      const data = (await axios.get(`/api/database/${databaseId}/backup`)).data;
      const backupList = data.data.map((backup: ResourceObject) => {
        return convertBackup(backup, data.included);
      });

      this.setTableListByDatabaseId({ databaseId, backupList });
      return backupList;
    },

    async fetchBackupSettingByDatabaseId(databaseId: DatabaseId) {
      const data = (
        await axios.get(`/api/database/${databaseId}/backup-setting`)
      ).data;
      const backupSetting: BackupSetting = convertBackupSetting(
        data.data,
        data.included
      );

      this.upsertBackupSettingByDatabaseId({ databaseId, backupSetting });
      return backupSetting;
    },

    async upsertBackupSetting({
      newBackupSetting,
    }: {
      newBackupSetting: BackupSettingUpsert;
    }) {
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
        data.included
      );

      this.upsertBackupSettingByDatabaseId({
        databaseId: newBackupSetting.databaseId,
        backupSetting: updatedBackupSetting,
      });

      return updatedBackupSetting;
    },
  },
});
