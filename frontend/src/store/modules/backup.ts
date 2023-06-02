import { defineStore } from "pinia";
import axios from "axios";
import {
  Backup,
  BackupSetting,
  BackupSettingState,
  BackupSettingUpsert,
  BackupState,
  DatabaseId,
  ResourceObject,
  unknown,
} from "@/types";

export function convertBackup(
  backup: ResourceObject,
  includedList: ResourceObject[]
): Backup {
  return {
    ...(backup.attributes as Omit<Backup, "id">),
    id: parseInt(backup.id),
  };
}

function convertBackupSetting(
  backupSetting: ResourceObject,
  includedList: ResourceObject[]
): BackupSetting {
  return {
    ...(backupSetting.attributes as Omit<BackupSetting, "id">),
    id: parseInt(backupSetting.id),
  };
}

export const useLegacyBackupStore = defineStore("backup", {
  state: (): BackupState & BackupSettingState => ({
    backupList: new Map(),
    backupSetting: new Map(),
  }),

  actions: {
    convert(backup: ResourceObject, includedList: ResourceObject[]): Backup {
      return convertBackup(backup, includedList || []);
    },
    backupSettingByDatabaseId(databaseId: DatabaseId): BackupSetting {
      return (
        this.backupSetting.get(databaseId) ||
        (unknown("BACKUP_SETTING") as BackupSetting)
      );
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

    async upsertBackupSettingByEnvironmentId(
      environmentId: string,
      backupSettingUpsert: Omit<BackupSettingUpsert, "databaseId">
    ) {
      const url = `/api/environment/${environmentId}/backup-setting`;
      await axios.patch(url, {
        data: {
          type: "backupSettingUpsert",
          attributes: backupSettingUpsert,
        },
      });
    },
  },
});
