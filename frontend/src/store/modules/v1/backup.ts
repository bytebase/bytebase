import { defineStore } from "pinia";
import { computed, reactive, unref, watch } from "vue";

import { databaseServiceClient } from "@/grpcweb";
import { Backup, ListBackupRequest } from "@/types/proto/v1/database_service";
import { MaybeRef } from "@/types";
import { useAuthStore } from "../auth";

export const useBackupV1Store = defineStore("backup_v1", () => {
  const backupListMapByDatabase = reactive(new Map<string, Backup[]>());

  const upsertBackupListMap = (parent: string, backups: Backup[]) => {
    backupListMapByDatabase.set(parent, backups);
  };

  const backupListByDatabase = (database: string) => {
    return backupListMapByDatabase.get(database) ?? [];
  };
  const fetchBackupList = async (params: Partial<ListBackupRequest>) => {
    const { parent } = params;
    if (!parent) {
      throw new Error('"parent" parameter is required');
    }
    const { backups } = await databaseServiceClient.listBackup(params);
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

  return { backupListByDatabase, fetchBackupList, createBackup };
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
