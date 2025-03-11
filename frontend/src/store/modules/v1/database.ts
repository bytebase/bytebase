import { uniq } from "lodash-es";
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
import {
  extractDatabaseResourceName,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils";
import {
  instanceNamePrefix,
  projectNamePrefix,
  workspaceNamePrefix,
} from "./common";
import { useDBSchemaV1Store } from "./dbSchema";
import { useEnvironmentV1Store } from "./environment";
import { batchGetOrFetchProjects, useProjectV1Store } from "./project";

const formatListDatabaseParent = async (
  parent: string
): Promise<{ parent: string; filter?: string }> => {
  if (parent.startsWith(projectNamePrefix)) {
    const project = await useProjectV1Store().getOrFetchProjectByName(parent);
    if (!hasProjectPermissionV2(project, "bb.projects.get")) {
      return {
        parent: `${workspaceNamePrefix}-`,
        filter: `project == "${parent}"`,
      };
    }
    return { parent };
  }
  if (parent.startsWith(instanceNamePrefix)) {
    if (!hasWorkspacePermissionV2("bb.instances.get")) {
      return {
        parent: `${workspaceNamePrefix}-`,
        filter: `instance == "${parent}"`,
      };
    }
    return { parent };
  }
  return { parent: `${workspaceNamePrefix}-` };
};

export const useDatabaseV1Store = defineStore("database_v1", () => {
  const databaseMapByName = reactive(new Map<string, ComposedDatabase>());
  const dbSchemaStore = useDBSchemaV1Store();
  const databaseRequestCache = new Map<string, Promise<ComposedDatabase>>();

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
        dbSchemaStore.removeCache(db.name);
      }
    }
  };

  const fetchDatabases = async (params: {
    pageSize: number;
    pageToken?: string;
    parent: string;
    filter?: string;
    showDeleted?: boolean;
  }): Promise<{
    databases: ComposedDatabase[];
    nextPageToken: string;
  }> => {
    const { parent, filter } = await formatListDatabaseParent(params.parent);

    const { databases, nextPageToken } =
      await databaseServiceClient.listDatabases({
        ...params,
        parent,
        filter: filter
          ? params.filter
            ? `${params.filter} && ${filter}`
            : filter
          : params.filter,
      });
    if (parent.startsWith(instanceNamePrefix)) {
      removeCacheByInstance(parent);
    }

    const composedDatabases = await upsertDatabaseMap(databases);
    return {
      databases: composedDatabases,
      nextPageToken,
    };
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
  // TODO(ed): deprecate it.
  const databaseListByProject = (project: string) => {
    return databaseList.value.filter((db) => db.project === project);
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
  const getOrFetchDatabaseByName = async (name: string, silent = true) => {
    const existed = databaseMapByName.get(name);
    if (existed) {
      return existed;
    }
    if (!isValidDatabaseName(name)) {
      return unknownDatabase();
    }
    const cached = databaseRequestCache.get(name);
    if (cached) return cached;
    const request = fetchDatabaseByName(name, silent);
    databaseRequestCache.set(name, request);
    return request;
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

  return {
    reset,
    removeCacheByInstance,
    upsertDatabaseMap,
    syncDatabase,
    databaseListByProject,
    getDatabaseByName,
    fetchDatabaseByName,
    getOrFetchDatabaseByName,
    batchUpdateDatabases,
    updateDatabase,
    fetchDatabaseSchema,
    updateDatabaseInstance,
    diffSchema,
    fetchDatabases,
  };
});

export const useDatabaseV1ByName = (name: MaybeRef<string>) => {
  const store = useDatabaseV1Store();
  const ready = ref(true);
  watch(
    () => unref(name),
    (name) => {
      ready.value = false;
      store.getOrFetchDatabaseByName(name).then(() => {
        ready.value = true;
      });
    },
    { immediate: true }
  );
  const database = computed(() => store.getDatabaseByName(unref(name)));

  return {
    database,
    ready,
  };
};

export const batchGetOrFetchDatabases = async (databaseNames: string[]) => {
  const store = useDatabaseV1Store();

  const distinctDatabaseList = uniq(databaseNames);
  await Promise.all(
    distinctDatabaseList.map((databaseName) => {
      if (!databaseName || !isValidDatabaseName(databaseName)) {
        return;
      }
      return store.getOrFetchDatabaseByName(databaseName, true /* silent */);
    })
  );
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
