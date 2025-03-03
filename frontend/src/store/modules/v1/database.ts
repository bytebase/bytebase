import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch, markRaw } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import type { ComposedInstance, ComposedDatabase, MaybeRef } from "@/types";
import {
  isValidDatabaseName,
  unknownDatabase,
  unknownEnvironment,
  unknownInstanceResource,
} from "@/types";
import type {
  Database,
  UpdateDatabaseRequest,
  DiffSchemaRequest,
  BatchUpdateDatabasesRequest,
} from "@/types/proto/v1/database_service";
import type { InstanceResource } from "@/types/proto/v1/instance_service";
import { extractDatabaseResourceName, hasProjectPermissionV2 } from "@/utils";
import { useEnvironmentV1Store } from "./environment";
import { batchGetOrFetchProjects, useProjectV1Store } from "./project";

export const useDatabaseV1Store = defineStore("database_v1", () => {
  const databaseMapByName = reactive(new Map<string, ComposedDatabase>());

  // Getters
  const databaseList = computed(() => {
    return Array.from(databaseMapByName.values());
  });

  // Actions
  const reset = () => {
    databaseMapByName.clear();
  };

  const removeCacheByInstance = (instance: string) => {
    for (const db of databaseList.value) {
      if (db.instance === instance) {
        databaseMapByName.delete(db.name);
      }
    }
  };

  const upsertDatabaseMap = async (databaseList: Database[]) => {
    const composedDatabaseList = await batchComposeDatabase(databaseList);
    composedDatabaseList.forEach((database) => {
      databaseMapByName.set(database.name, database);
    });
    return composedDatabaseList;
  };
  const updateDatabaseInstance = (instance: ComposedInstance) => {
    for (const [_, database] of databaseMapByName) {
      if (database.instance !== instance.name) {
        continue;
      }
      if (databaseMapByName.has(database.name)) {
        const db = databaseMapByName.get(database.name);
        if (db) {
          db.instanceResource.activation = instance.activation;
        }
      }
    }
  };
  const syncDatabase = async (database: string, refresh = false) => {
    await databaseServiceClient.syncDatabase({
      name: database,
    });
    if (refresh) {
      await fetchDatabaseByName(database);
    }
  };
  const databaseListByUser = computed(() => {
    return databaseList.value.filter((db) => {
      if (hasProjectPermissionV2(db.projectEntity, "bb.databases.get"))
        return true;
      return false;
    });
  });
  const databaseListByProject = (project: string) => {
    return databaseList.value.filter((db) => db.project === project);
  };
  const databaseListByInstance = (instance: string) => {
    return databaseList.value.filter((db) => db.instance === instance);
  };
  const databaseListByEnvironment = (environment: string) => {
    return databaseList.value.filter(
      (db) => db.effectiveEnvironment === environment
    );
  };
  const getDatabaseByName = (name: string) => {
    return databaseMapByName.get(name) ?? unknownDatabase();
  };
  const fetchDatabaseByName = async (name: string, silent = false) => {
    const database = await databaseServiceClient.getDatabase(
      {
        name,
      },
      {
        silent,
      }
    );

    const [composed] = await upsertDatabaseMap([database]);

    return composed;
  };
  const getOrFetchDatabaseByName = async (name: string, silent = false) => {
    const existed = databaseMapByName.get(name);
    if (existed) {
      return existed;
    }
    await fetchDatabaseByName(name, silent);
    return getDatabaseByName(name);
  };
  const batchUpdateDatabases = async (params: BatchUpdateDatabasesRequest) => {
    const updated = await databaseServiceClient.batchUpdateDatabases(params);
    const composed = await upsertDatabaseMap(updated.databases);
    return composed;
  };
  const updateDatabase = async (params: UpdateDatabaseRequest) => {
    const updated = await databaseServiceClient.updateDatabase(params);
    const [composed] = await upsertDatabaseMap([updated]);
    return composed;
  };
  const fetchDatabaseSchema = async (name: string, sdlFormat = false) => {
    const schema = await databaseServiceClient.getDatabaseSchema({
      name,
      sdlFormat,
    });
    return schema;
  };
  const diffSchema = async (request: DiffSchemaRequest) => {
    const resp = await databaseServiceClient.diffSchema(request);
    return resp;
  };

  const transferDatabases = async (
    databaseList: Database[],
    project: string
  ) => {
    const updates = databaseList.map((db) => {
      const databasePatch = {
        ...db,
      };
      databasePatch.project = project;
      const updateMask = ["project"];
      return {
        database: databasePatch,
        updateMask,
      } as UpdateDatabaseRequest;
    });

    const response = await batchUpdateDatabases({
      parent: "-",
      requests: updates,
    });
    return response;
  };

  return {
    reset,
    removeCacheByInstance,
    databaseList,
    upsertDatabaseMap,
    syncDatabase,
    databaseListByUser,
    databaseListByProject,
    databaseListByInstance,
    databaseListByEnvironment,
    getDatabaseByName,
    fetchDatabaseByName,
    getOrFetchDatabaseByName,
    batchUpdateDatabases,
    updateDatabase,
    fetchDatabaseSchema,
    updateDatabaseInstance,
    diffSchema,
    transferDatabases,
  };
});

export const useDatabaseV1ByName = (name: MaybeRef<string>) => {
  const store = useDatabaseV1Store();
  const ready = ref(true);
  watch(
    () => unref(name),
    (name) => {
      if (!isValidDatabaseName(store.getDatabaseByName(name).name)) {
        ready.value = false;
        store.fetchDatabaseByName(name).then(() => {
          ready.value = true;
        });
      }
    },
    { immediate: true }
  );
  const database = computed(() => store.getDatabaseByName(unref(name)));

  return {
    database,
    ready,
  };
};

export const batchComposeDatabase = async (databaseList: Database[]) => {
  const environmentV1Store = useEnvironmentV1Store();
  const projectV1Store = useProjectV1Store();

  await batchGetOrFetchProjects(databaseList.map((db) => db.project));

  return databaseList.map((db) => {
    const composed = db as ComposedDatabase;
    const { databaseName, instance } = extractDatabaseResourceName(db.name);

    composed.databaseName = databaseName;
    composed.instance = instance;
    composed.instanceResource = composeInstanceResourceForDatabase(
      instance,
      db
    );
    composed.environment = composed.instanceResource.environment;
    composed.projectEntity = projectV1Store.getProjectByName(db.project);
    composed.effectiveEnvironmentEntity =
      environmentV1Store.getEnvironmentByName(db.effectiveEnvironment) ??
      unknownEnvironment();
    return markRaw(composed);
  });
};

export const composeInstanceResourceForDatabase = (
  name: string,
  db: Database
): InstanceResource => {
  return (
    db.instanceResource ?? {
      ...unknownInstanceResource(),
      name,
    }
  );
};
