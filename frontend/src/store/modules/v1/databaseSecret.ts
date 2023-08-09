import { defineStore } from "pinia";
import { reactive } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import { Secret } from "@/types/proto/v1/database_service";
import { secretNamePrefix } from "./common";

export const useDatabaseSecretStore = defineStore("database-secret", () => {
  const secretMapByDatabase = reactive(new Map<string, Secret[]>());

  const getSecretListByDatabase = (database: string) => {
    return secretMapByDatabase.get(database) ?? [];
  };

  const fetchSecretList = async (parent: string) => {
    const res = await databaseServiceClient.listSecrets({
      parent,
    });
    secretMapByDatabase.set(parent, res.secrets);
    return res.secrets;
  };

  const upsertSecret = async (
    secret: Secret,
    updateMask: string[],
    allowMissing: boolean
  ) => {
    const database = secret.name.split(secretNamePrefix)[0];
    const updatedSecret = await databaseServiceClient.updateSecret({
      secret,
      updateMask,
      allowMissing,
    });
    if (secretMapByDatabase.has(database)) {
      const list = secretMapByDatabase.get(database) ?? [];
      const index = list.findIndex((s) => s.name === secret.name);
      if (index >= 0) {
        list[index] = updatedSecret;
        secretMapByDatabase.set(database, list);
      }
    }
    return updatedSecret;
  };

  const deleteSecret = async (name: string) => {
    await databaseServiceClient.deleteSecret({
      name,
    });
    const database = name.split(secretNamePrefix)[0];
    if (secretMapByDatabase.has(database)) {
      const list = secretMapByDatabase.get(database) ?? [];
      const index = list.findIndex((s) => s.name === name);
      if (index >= 0) {
        list.splice(index, 1);
        secretMapByDatabase.set(database, list);
      }
    }
  };

  return {
    fetchSecretList,
    upsertSecret,
    deleteSecret,
    getSecretListByDatabase,
  };
});
