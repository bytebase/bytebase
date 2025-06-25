import { defineStore } from "pinia";
import { reactive } from "vue";
import { create } from "@bufbuild/protobuf";
import { createContextValues, Code } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/grpcweb";
import { ignoredCodesContextKey } from "@/grpcweb/context-key";
import {
  ListSecretsRequestSchema,
  UpdateSecretRequestSchema,
  DeleteSecretRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  convertNewSecretToOld,
  convertOldSecretToNew,
} from "@/utils/v1/database-conversions";
import type { Secret } from "@/types/proto/v1/database_service";
import { secretNamePrefix } from "./common";

export const useDatabaseSecretStore = defineStore("database-secret", () => {
  const secretMapByDatabase = reactive(new Map<string, Secret[]>());

  const getSecretListByDatabase = (database: string) => {
    return secretMapByDatabase.get(database) ?? [];
  };

  const fetchSecretList = async (parent: string) => {
    const request = create(ListSecretsRequestSchema, {
      parent,
    });
    const response = await databaseServiceClientConnect.listSecrets(
      request,
      {
        contextValues: createContextValues().set(ignoredCodesContextKey, [Code.NotFound, Code.PermissionDenied]),
      }
    );
    const secrets = response.secrets.map((s) => convertNewSecretToOld(s));
    secretMapByDatabase.set(parent, secrets);
    return secrets;
  };

  const upsertSecret = async (
    secret: Secret,
    updateMask: string[],
    allowMissing: boolean
  ) => {
    const database = secret.name.split(secretNamePrefix)[0];
    const newSecret = convertOldSecretToNew(secret);
    const request = create(UpdateSecretRequestSchema, {
      secret: newSecret,
      updateMask: { paths: updateMask },
      allowMissing,
    });
    const response = await databaseServiceClientConnect.updateSecret(request);
    const updatedSecret = convertNewSecretToOld(response);
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
    const request = create(DeleteSecretRequestSchema, {
      name,
    });
    await databaseServiceClientConnect.deleteSecret(request);
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
