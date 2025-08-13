import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch, markRaw } from "vue";
import { databaseServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import type { ComposedInstance, ComposedDatabase, MaybeRef } from "@/types";
import {
  isValidProjectName,
  isValidInstanceName,
  isValidDatabaseName,
  unknownDatabase,
  unknownInstanceResource,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  GetDatabaseRequestSchema,
  ListDatabasesRequestSchema,
  BatchGetDatabasesRequestSchema,
  BatchUpdateDatabasesRequestSchema,
  UpdateDatabaseRequestSchema,
  BatchSyncDatabasesRequestSchema,
  SyncDatabaseRequestSchema,
  GetDatabaseSchemaRequestSchema,
  DiffSchemaRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type {
  Database,
  UpdateDatabaseRequest,
  DiffSchemaRequest,
  BatchUpdateDatabasesRequest,
} from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName, isNullOrUndefined } from "@/utils";
import {
  instanceNamePrefix,
  projectNamePrefix,
  workspaceNamePrefix,
} from "./common";
import { useDBSchemaV1Store } from "./dbSchema";
import { useEnvironmentV1Store } from "./environment";
import { batchGetOrFetchProjects, useProjectV1Store } from "./project";

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

const getListDatabaseFilter = (filter: DatabaseFilter): string => {
  const params: string[] = [];
  if (isValidProjectName(filter.project)) {
    params.push(`project == "${filter.project}"`);
  }
  if (isValidInstanceName(filter.instance)) {
    params.push(`instance == "${filter.instance}"`);
  }
  if (filter.environment !== undefined) {
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
    // label filter like:
    // label == "region:asia,europe" && label == "tenant:bytebase"
    const labelMap = new Map<string, Set<string>>();
    for (const label of filter.labels) {
      const sections = label.split(":");
      if (sections.length !== 2) {
        continue;
      }
      if (!labelMap.has(sections[0])) {
        labelMap.set(sections[0], new Set());
      }
      labelMap.get(sections[0])!.add(sections[1]);
    }
    for (const [labelKey, labelValues] of labelMap.entries()) {
      params.push(`label == "${labelKey}:${[...labelValues].join(",")}"`);
    }
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
  const updateDatabaseInstance = (instance: ComposedInstance) => {
    for (const [_, database] of databaseMapByName) {
      if (database.instance !== instance.name) {
        continue;
      }
      // Conversion boundary: Extract InstanceResource fields from ComposedInstance
      database.instanceResource = {
        name: instance.name,
        uid: "",
        state: instance.state,
        title: instance.title,
        engine: instance.engine,
        externalLink: "",
        maximumConnections: 0,
        environment: instance.environment,
        activation: true,
        dataSources: [],
        lastSyncTime: undefined,
        syncInterval: undefined,
        options: undefined,
      } as any; // Cross-service boundary conversion
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
    const request = create(BatchGetDatabasesRequestSchema, {
      names,
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
  const fetchDatabaseSchema = async (database: string, sdlFormat = false) => {
    const request = create(GetDatabaseSchemaRequestSchema, {
      name: `${database}/schema`,
      sdlFormat,
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
    batchGetDatabases,
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
  const distinctDatabaseList = uniq(databaseNames).filter((databaseName) => {
    if (!databaseName || !isValidDatabaseName(databaseName)) {
      return false;
    }
    if (
      store.getDatabaseByName(databaseName) &&
      isValidDatabaseName(store.getDatabaseByName(databaseName).name)
    ) {
      return false;
    }
    return true;
  });
  if (distinctDatabaseList.length > 0) {
    await store.batchGetDatabases(distinctDatabaseList);
  }
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
