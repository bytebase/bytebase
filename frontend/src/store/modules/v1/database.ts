import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch, markRaw } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import type { ComposedInstance, ComposedDatabase, MaybeRef } from "@/types";
import {
  isValidEnvironmentName,
  isValidProjectName,
  isValidInstanceName,
  isValidDatabaseName,
  unknownDatabase,
  unknownEnvironment,
  unknownInstanceResource,
} from "@/types";
import { type Engine, engineToJSON } from "@/types/proto/v1/common";
import type {
  Database,
  UpdateDatabaseRequest,
  DiffSchemaRequest,
  BatchUpdateDatabasesRequest,
} from "@/types/proto/v1/database_service";
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
  if (isValidEnvironmentName(filter.environment)) {
    params.push(`environment == "${filter.environment}"`);
  }
  if (filter.excludeUnassigned) {
    params.push(`exclude_unassigned == true`);
  }
  if (filter.engines && filter.engines.length > 0) {
    // engine filter should be:
    // engine in ["MYSQL", "POSTGRES"]
    params.push(
      `engine in [${filter.engines.map((e) => `"${engineToJSON(e)}"`).join(", ")}]`
    );
  } else if (filter.excludeEngines && filter.excludeEngines.length > 0) {
    // engine filter should be:
    // !(engine in ["REDIS", "MONGODB"])
    params.push(
      `!(engine in [${filter.excludeEngines.map((e) => `"${engineToJSON(e)}"`).join(", ")}])`
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

    const { databases, nextPageToken } =
      await databaseServiceClient.listDatabases({
        parent: params.parent,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
        showDeleted: params.filter?.showDeleted,
        filter: getListDatabaseFilter(params.filter ?? {}),
      });
    if (params.parent.startsWith(instanceNamePrefix)) {
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
      database.instanceResource = instance;
    }
  };
  const batchSyncDatabases = async (databases: string[]) => {
    await databaseServiceClient.batchSyncDatabases({
      parent: `${instanceNamePrefix}-`,
      names: databases,
    });
  };
  const syncDatabase = async (database: string, refresh = false) => {
    await databaseServiceClient.syncDatabase({
      name: database,
    });
    if (refresh) {
      await fetchDatabaseByName(database);
    }
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
  const batchGetDatabases = async (names: string[], silent = true) => {
    const { databases } = await databaseServiceClient.batchGetDatabases(
      {
        names,
      },
      {
        silent,
      }
    );
    const composed = await upsertDatabaseMap(databases);
    return composed;
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
  const fetchDatabaseSchema = async (database: string, sdlFormat = false) => {
    const schema = await databaseServiceClient.getDatabaseSchema({
      name: `${database}/schema`,
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
      environmentV1Store.getEnvironmentByName(db.effectiveEnvironment) ??
      unknownEnvironment();
    return markRaw(composed);
  });
};
