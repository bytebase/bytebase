import { defineStore } from "pinia";
import { databaseServiceClient } from "@/grpcweb";
import { Secret } from "@/types/proto/v1/database_service";

export const useDatabaseSecretStore = defineStore("database-secret", () => {
  const fetchSecretList = async (parent: string) => {
    const res = await databaseServiceClient.listSecrets({
      parent,
    });
    return res.secrets;
  };

  const upsertSecret = async (
    secret: Secret,
    updateMask: string[],
    allowMissing: boolean
  ) => {
    return await databaseServiceClient.updateSecret({
      secret,
      updateMask,
      allowMissing,
    });
  };

  const deleteSecret = async (name: string) => {
    return await databaseServiceClient.deleteSecret({
      name,
    });
  };

  return { fetchSecretList, upsertSecret, deleteSecret };
});
