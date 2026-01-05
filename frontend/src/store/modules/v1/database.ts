import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, markRaw, reactive, ref, unref, watch } from "vue";
import { databaseServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { ComposedDatabase, MaybeRef } from "@/types";
import {
  isValidDatabaseName,
  isValidEnvironmentName,
  isValidInstanceName,
  isValidProjectName,
  unknownDatabase,
  unknownEnvironment,
  unknownInstanceResource,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  BatchUpdateDatabasesRequest,
  Database,
  DiffSchemaRequest,
  UpdateDatabaseRequest,
} from "@/types/proto-es/v1/database_service_pb";
import {
  BatchGetDatabasesRequestSchema,
  BatchSyncDatabasesRequestSchema,
  BatchUpdateDatabasesRequestSchema,
  DiffSchemaRequestSchema,
  GetDatabaseRequestSchema,
  GetDatabaseSchemaRequestSchema,
  ListDatabasesRequestSchema,
  SyncDatabaseRequestSchema,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  type Instance,
  InstanceResourceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { extractDatabaseResourceName, isNullOrUndefined } from "@/utils";
import {
  instanceNamePrefix,
  projectNamePrefix,
  workspaceNamePrefix,
} from "./common";
import { useDBSchemaV1Store } from "./dbSchema";
import { useEnvironmentV1Store } from "./environment";
import { useProjectV1Store } from "./project";

export interface DatabaseFilter {
  project?: string;
  instance?: string;
  environment?: string;
  query?: string;
  showDeleted?: boolean;
  excludeUnassigned?: boolean;
  // label should be "{label key}:{label value}" format
  labels?: string[];
  engines?: Engine[];
  excludeEngines?: Engine[];
  drifted?: boolean;
  table?: string;
}

const isValidParentName = (parent: string): boolean => {
  if (parent.startsWith(workspaceNamePrefix)) {
    return true;
  }
  if (parent.startsWith(projectNamePrefix)) {
    return isValidProjectName(parent);
  }
  if (parent.startsWith(instanceNamePrefix)) {
    return isValidInstanceName(parent);
  }
  return false;
};

// getLabelFilter convert the input to API label filter format like: labels.{key} == {value}
// the input label should be "{label key}:{value1},{value2}" format
export const getLabelFilter = (labels: string[]): string[] => {
  const labelMap = new Map<string, string[]>(); // key: value[]
  for (const label of labels) {
    const sections = label.split(":");
    if (sections.length !== 2) {
      continue;
    }
    const [key, rawValue] = sections;
    const values = rawValue.split(",");
    if (!labelMap.has(key)) {
      labelMap.set(key, []);
    }
    labelMap.get(key)?.push(...values);
  }

  return [...labelMap.entries()].reduce((result, [key, values]) => {
    switch (values.length) {
      case 0:
        return result;
      case 1:
        result.push(`labels.${key} == "${values[0]}"`);
        return result;
      default:
        result.push(
          `labels.${key} in [${values.map((v) => `"${v}"`).join(", ")}]`
        );
        return result;
    }
  }, [] as string[]);
};

const getListDatabaseFilter = (filter: DatabaseFilter): string => {
  const params: string[] = [];
  if (isValidProjectName(filter.project)) {
    params.push(`project == "${filter.project}"`);
  }
  if (isValidInstanceName(filter.instance)) {
    params.push(`instance == "${filter.instance}"`);
  }
  if (filter.environment === unknownEnvironment().name) {
    params.push(`environment == ""`);
  } else if (isValidEnvironmentName(filter.environment)) {
    params.push(`environment == "${filter.environment}"`);
  }
  if (filter.excludeUnassigned) {
    params.push(`exclude_unassigned == true`);
  }
  if (filter.engines && filter.engines.length > 0) {
    // engine filter should be:
    // engine in ["MYSQL", "POSTGRES"]
    params.push(
      `engine in [${filter.engines.map((e) => `"${Engine[e]}"`).join(", ")}]`
    );
  } else if (filter.excludeEngines && filter.excludeEngines.length > 0) {
    // engine filter should be:
    // !(engine in ["REDIS", "MONGODB"])
    params.push(
      `!(engine in [${filter.excludeEngines.map((e) => `"${Engine[e]}"`).join(", ")}])`
    );
  }
  if (!isNullOrUndefined(filter.drifted)) {
    params.push(`drifted == ${filter.drifted}`);
  }
  const keyword = filter.query?.trim()?.toLowerCase();
  if (keyword) {
    params.push(`name.matches("${keyword}")`);
  }

  if (filter.labels) {
    params.push(...getLabelFilter(filter.labels));
  }

  if (filter.table) {
    params.push(`table.matches("${filter.table}")`);
  }

  return params.join(" && ");
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
    filter?: DatabaseFilter;
    skipCacheRemoval?: boolean;
    orderBy?: string;
  }): Promise<{
    databases: ComposedDatabase[];
    nextPageToken: string;
  }> => {
    if (!isValidParentName(params.parent)) {
      return {
        databases: [],
        nextPageToken: "",
      };
    }

    const request = create(ListDatabasesRequestSchema, {
      parent: params.parent,
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      orderBy: params.orderBy,
      showDeleted: params.filter?.showDeleted,
      filter: getListDatabaseFilter(params.filter ?? {}),
    });
    const response = await databaseServiceClientConnect.listDatabases(request);
    const databases = response.databases; // Work directly with proto-es types
    const { nextPageToken } = response;
    if (
      params.parent.startsWith(instanceNamePrefix) &&
      !params.skipCacheRemoval
    ) {
      removeCacheByInstance(params.parent);
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
  const updateDatabaseInstance = (instance: Instance) => {
    for (const [_, database] of databaseMapByName) {
      if (database.instance !== instance.name) {
        continue;
      }
      database.instanceResource = create(InstanceResourceSchema, {
        name: instance.name,
        title: instance.title,
        engine: instance.engine,
        environment: instance.environment,
        activation: instance.activation,
        dataSources: [],
      });
    }
  };
  const batchSyncDatabases = async (databases: string[]) => {
    const request = create(BatchSyncDatabasesRequestSchema, {
      parent: `${instanceNamePrefix}-`,
      names: databases,
    });
    await databaseServiceClientConnect.batchSyncDatabases(request);
  };
  const syncDatabase = async (database: string, refresh = false) => {
    const request = create(SyncDatabaseRequestSchema, {
      name: database,
    });
    await databaseServiceClientConnect.syncDatabase(request);
    if (refresh) {
      await fetchDatabaseByName(database);
    }
  };
  const getDatabaseByName = (name: string) => {
    return databaseMapByName.get(name) ?? unknownDatabase();
  };
  const fetchDatabaseByName = async (name: string, silent = false) => {
    const request = create(GetDatabaseRequestSchema, {
      name,
    });
    const database = await databaseServiceClientConnect.getDatabase(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });

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

  const batchGetDatabases = async (names: string[], silent = true) => {
    const validNames = names.filter(isValidDatabaseName);
    if (validNames.length === 0) {
      return [];
    }
    const request = create(BatchGetDatabasesRequestSchema, {
      names: validNames,
    });
    const response = await databaseServiceClientConnect.batchGetDatabases(
      request,
      {
        contextValues: createContextValues().set(silentContextKey, silent),
      }
    );
    const databases = response.databases; // Work directly with proto-es types
    const composed = await upsertDatabaseMap(databases);
    return composed;
  };

  const batchGetOrFetchDatabases = async (databaseNames: string[]) => {
    const validDatabaseList = uniq(databaseNames).filter(isValidDatabaseName);
    const pendingFetch = validDatabaseList.filter((databaseName) => {
      if (
        getDatabaseByName(databaseName) &&
        isValidDatabaseName(getDatabaseByName(databaseName).name)
      ) {
        return false;
      }
      return true;
    });
    await batchGetDatabases(pendingFetch);
    return validDatabaseList.map(getDatabaseByName);
  };

  const batchUpdateDatabases = async (params: BatchUpdateDatabasesRequest) => {
    const request = create(BatchUpdateDatabasesRequestSchema, {
      parent: params.parent,
      requests: params.requests.map((req) => ({
        database: req.database,
        updateMask: req.updateMask,
      })),
    });
    const response =
      await databaseServiceClientConnect.batchUpdateDatabases(request);
    const updatedDatabases = response.databases; // Work directly with proto-es types
    const composed = await upsertDatabaseMap(updatedDatabases);
    return composed;
  };
  const updateDatabase = async (params: UpdateDatabaseRequest) => {
    if (!params.database) {
      throw new Error("Database is required for update");
    }
    const request = create(UpdateDatabaseRequestSchema, {
      ...params,
      database: params.database,
      updateMask: params.updateMask,
    });
    const updated = await databaseServiceClientConnect.updateDatabase(request);
    const [composed] = await upsertDatabaseMap([updated]);
    return composed;
  };
  const fetchDatabaseSchema = async (database: string) => {
    const request = create(GetDatabaseSchemaRequestSchema, {
      name: `${database}/schema`,
    });
    const schema =
      await databaseServiceClientConnect.getDatabaseSchema(request);
    return schema;
  };
  const diffSchema = async (params: DiffSchemaRequest) => {
    const request = create(DiffSchemaRequestSchema, params);
    const resp = await databaseServiceClientConnect.diffSchema(request);
    return resp;
  };

  return {
    reset,
    databaseList,
    removeCacheByInstance,
    upsertDatabaseMap,
    syncDatabase,
    batchSyncDatabases,
    getDatabaseByName,
    fetchDatabaseByName,
    getOrFetchDatabaseByName,
    batchGetOrFetchDatabases,
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

export const batchComposeDatabase = async (databaseList: Database[]) => {
  const environmentV1Store = useEnvironmentV1Store();
  const projectV1Store = useProjectV1Store();

  await projectV1Store.batchGetOrFetchProjects(
    databaseList.map((db) => db.project)
  );

  return databaseList.map((db) => {
    const composed = db as ComposedDatabase;
    const { databaseName, instance } = extractDatabaseResourceName(db.name);

    composed.databaseName = databaseName;
    composed.instance = instance;
    composed.instanceResource = db.instanceResource ?? {
      ...unknownInstanceResource(),
      name: instance,
    };
    composed.environment = composed.instanceResource.environment;
    composed.projectEntity = projectV1Store.getProjectByName(db.project);
    composed.effectiveEnvironmentEntity =
      environmentV1Store.getEnvironmentByName(db.effectiveEnvironment ?? "");
    return markRaw(composed);
  });
};
