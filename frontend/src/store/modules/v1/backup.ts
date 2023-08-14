import { defineStore } from "pinia";
import { computed, reactive, unref, watch } from "vue";
import { databaseServiceClient, environmentServiceClient } from "@/grpcweb";
import { MaybeRef } from "@/types";
import {
  Backup,
  BackupSetting,
  ListBackupsRequest,
} from "@/types/proto/v1/database_service";
import { EnvironmentBackupSetting } from "@/types/proto/v1/environment_service";
import { useAuthStore } from "../auth";

export const useBackupV1Store = defineStore("backup_v1", () => {
  const backupListMapByDatabase = reactive(new Map<string, Backup[]>());

  const upsertBackupListMap = (parent: string, backups: Backup[]) => {
    backupListMapByDatabase.set(parent, backups);
  };

  const backupListByDatabase = (database: string) => {
    return backupListMapByDatabase.get(database) ?? [];
  };
  const fetchBackupList = async (params: Partial<ListBackupsRequest>) => {
    const { parent } = params;
    if (!parent) {
      throw new Error('"parent" parameter is required');
    }
    const { backups } = await databaseServiceClient.listBackups(params);
    upsertBackupListMap(parent, backups);
    return backups;
  };
  const createBackup = async (
    backup: Backup,
    parent: string,
    refreshList = true
  ) => {
    const created = await databaseServiceClient.createBackup({
      parent,
      backup,
    });
    if (refreshList) {
      await fetchBackupList({ parent });
    }
    return created;
  };

  const fetchBackupSetting = async (databaseName: string) => {
    try {
      const backupSetting = await databaseServiceClient.getBackupSetting({
        name: `${databaseName}/backupSetting`,
      });
      return backupSetting;
    } catch {
      return;
    }
  };

  const upsertBackupSetting = async (backupSetting: BackupSetting) => {
    const updated = await databaseServiceClient.updateBackupSetting({
      setting: backupSetting,
    });
    return updated;
  };

  const upsertEnvironmentBackupSetting = async (
    backupSetting: EnvironmentBackupSetting
  ) => {
    await environmentServiceClient.updateBackupSetting({
      setting: backupSetting,
    });
  };

  const parseBackupSchedule = (
    schedule: string
  ): {
    hourOfDay: number;
    dayOfWeek: number;
  } => {
    const sections = schedule.split(" ");
    if (sections.length !== 5) {
      return {
        hourOfDay: 0,
        dayOfWeek: -1,
      };
    }
    const hourOfDay = Number(sections[1]);
    const dayOfWeek = sections[4] === "*" ? -1 : Number(sections[4]);
    return {
      hourOfDay,
      dayOfWeek,
    };
  };

  const buildSimpleSchedule = ({
    enabled,
    hourOfDay,
    dayOfWeek,
  }: {
    enabled: boolean;
    hourOfDay: number;
    dayOfWeek: number;
  }): string => {
    if (!enabled) {
      return "";
    }
    if (dayOfWeek === -1) {
      return `0 ${hourOfDay} * * *`;
    }
    return `0 ${hourOfDay} * * ${dayOfWeek}`;
  };

  return {
    backupListByDatabase,
    fetchBackupList,
    createBackup,
    fetchBackupSetting,
    upsertBackupSetting,
    upsertEnvironmentBackupSetting,
    parseBackupSchedule,
    buildSimpleSchedule,
  };
});

export const useBackupListByDatabaseName = (name: MaybeRef<string>) => {
  const store = useBackupV1Store();
  const authStore = useAuthStore();
  watch(
    [() => authStore.isLoggedIn(), () => unref(name)],
    ([isLoggedIn, name]) => {
      if (!isLoggedIn) return;

      store.fetchBackupList({
        parent: name,
      });
    },
    {
      immediate: true,
    }
  );

  return computed(() => store.backupListByDatabase(unref(name)));
};
