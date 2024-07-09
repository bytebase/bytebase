import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch, markRaw } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import type { ComposedInstance, ComposedDatabase, MaybeRef } from "@/types";
import {
  emptyDatabase,
  EMPTY_ID,
  unknownDatabase,
  unknownEnvironment,
  unknownInstance,
  UNKNOWN_ID,
  UNKNOWN_INSTANCE_NAME,
} from "@/types";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import type {
  Database,
  UpdateDatabaseRequest,
  DiffSchemaRequest,
  SearchDatabasesRequest,
  BatchUpdateDatabasesRequest,
} from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName, hasProjectPermissionV2 } from "@/utils";
import { useGracefulRequest } from "../utils";
import { useEnvironmentV1Store } from "./environment";
import { useInstanceV1Store } from "./instance";
import { useProjectV1Store } from "./project";

export const DEFAULT_DATABASE_PAGE_SIZE = 1000000;

export const useDatabaseV1Store = defineStore("database_v1", () => {
  const databaseMapByName = reactive(new Map<string, ComposedDatabase>());
  const databaseMapByUID = reactive(new Map<string, ComposedDatabase>());

  // Getters
  const databaseList = computed(() => {
    return Array.from(databaseMapByName.values());
  });

  // Actions
  const reset = () => {
    databaseMapByName.clear();
    databaseMapByUID.clear();
  };

  const removeCacheByInstance = (instance: string) => {
    for (const db of databaseList.value) {
      if (db.instance === instance) {
        databaseMapByName.delete(db.name);
        databaseMapByUID.delete(db.uid);
      }
    }
  };

  const upsertDatabaseMap = async (databaseList: Database[]) => {
    const composedDatabaseList = await batchComposeDatabase(databaseList);
    composedDatabaseList.forEach((database) => {
      databaseMapByName.set(database.name, database);
      databaseMapByUID.set(database.uid, database);
    });
    return composedDatabaseList;
  };
  const updateDatabaseInstance = (instance: ComposedInstance) => {
    for (const [_, database] of databaseMapByName) {
      if (database.instance !== instance.name) {
        continue;
      }
      if (databaseMapByName.has(database.name)) {
        databaseMapByName.get(database.name)!.instanceEntity = instance;
      }
      if (databaseMapByUID.has(database.uid)) {
        databaseMapByUID.get(database.uid)!.instanceEntity = instance;
      }
    }
  };
  const searchDatabases = async (args: Partial<SearchDatabasesRequest>) => {
    const { databases } = await databaseServiceClient.searchDatabases({
      pageSize: DEFAULT_DATABASE_PAGE_SIZE,
      ...args,
    });
    const composedDatabaseList = await upsertDatabaseMap(databases);
    return composedDatabaseList;
  };
  const syncDatabase = async (database: string, refresh = false) => {
    await databaseServiceClient.syncDatabase({
      name: database,
    });
    if (refresh) {
      await fetchDatabaseByName(database);
    }
  };
  const databaseListByUser = (user: User) => {
    return databaseList.value.filter((db) => {
      if (hasProjectPermissionV2(db.projectEntity, user, "bb.databases.get"))
        return true;
      return false;
    });
  };
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
  const getDatabaseByUID = (uid: string) => {
    if (uid === String(EMPTY_ID)) return emptyDatabase();
    if (uid === String(UNKNOWN_ID)) return unknownDatabase();

    return databaseMapByUID.get(uid) ?? unknownDatabase();
  };
  const fetchDatabaseByUID = async (uid: string, silent = false) => {
    const database = await databaseServiceClient.getDatabase(
      {
        name: `instances/-/databases/${uid}`,
      },
      {
        silent,
      }
    );
    const [composed] = await upsertDatabaseMap([database]);

    return composed;
  };
  const getOrFetchDatabaseByUID = async (uid: string, silent = false) => {
    if (uid === String(EMPTY_ID)) return emptyDatabase();
    if (uid === String(UNKNOWN_ID)) return unknownDatabase();

    const existed = databaseList.value.find((db) => db.uid === uid);
    if (existed) {
      return existed;
    }
    await fetchDatabaseByUID(uid, silent);
    return getDatabaseByUID(uid);
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
  const fetchDatabaseSchema = async (
    name: string,
    sdlFormat = false,
    concise = false
  ) => {
    const schema = await databaseServiceClient.getDatabaseSchema({
      name,
      sdlFormat,
      concise,
    });
    return schema;
  };
  const diffSchema = async (request: DiffSchemaRequest) => {
    const resp = await databaseServiceClient.diffSchema(request);
    return resp;
  };

  const transferOneDatabase = async (database: Database, project: string) => {
    const updated = await updateDatabase({
      database: {
        ...database,
        project: project,
      },
      updateMask: ["project"],
    });
    return updated;
  };

  const transferDatabases = async (
    databaseList: Database[],
    project: string
  ) => {
    await useGracefulRequest(async () => {
      const requests = databaseList.map((db) => {
        transferOneDatabase(db, project);
      });
      await Promise.all(requests);
    });
  };

  return {
    reset,
    removeCacheByInstance,
    databaseList,
    searchDatabases,
    syncDatabase,
    databaseListByUser,
    databaseListByProject,
    databaseListByInstance,
    databaseListByEnvironment,
    getDatabaseByName,
    fetchDatabaseByName,
    getOrFetchDatabaseByName,
    getDatabaseByUID,
    getOrFetchDatabaseByUID,
    batchUpdateDatabases,
    updateDatabase,
    fetchDatabaseSchema,
    updateDatabaseInstance,
    diffSchema,
    transferDatabases,
  };
});

export const useSearchDatabaseV1List = (
  args: MaybeRef<Partial<SearchDatabasesRequest>>
) => {
  const store = useDatabaseV1Store();
  const ready = ref(false);
  const databaseList = ref<ComposedDatabase[]>([]);
  watch(
    () => JSON.stringify(unref(args)),
    () => {
      ready.value = false;
      store.searchDatabases(unref(args)).then((list) => {
        databaseList.value = list;
        ready.value = true;
      });
    },
    { immediate: true }
  );

  return { databaseList, ready };
};

export const useDatabaseV1ListByProject = (
  project: MaybeRef<string | undefined>
) => {
  const store = useDatabaseV1Store();

  const databaseList = computed(() => {
    const proj = unref(project);
    if (proj) {
      return store.databaseListByProject(proj);
    }
    return store.databaseList;
  });

  return { databaseList };
};

export const useDatabaseV1ByName = (name: MaybeRef<string>) => {
  const store = useDatabaseV1Store();
  const ready = ref(true);
  watch(
    () => unref(name),
    (name) => {
      if (store.getDatabaseByName(name).uid === String(UNKNOWN_ID)) {
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
  const projectV1Store = useProjectV1Store();
  const instanceV1Store = useInstanceV1Store();
  const environmentV1Store = useEnvironmentV1Store();

  const distinctProjectList = uniq(databaseList.map((db) => db.project));
  const distinctInstanceList = uniq(
    databaseList
      .map((db) => extractDatabaseResourceName(db.name).instance)
      .filter((instance) => instance !== UNKNOWN_INSTANCE_NAME)
  );

  await Promise.all(
    distinctProjectList.map((project) => {
      if (project === DEFAULT_PROJECT_V1_NAME) {
        return;
      }
      return projectV1Store.getOrFetchProjectByName(project);
    })
  );
  await Promise.all(
    distinctInstanceList.map((instance) =>
      instanceV1Store.getOrFetchInstanceByName(instance)
    )
  );
  return databaseList.map((db) => {
    const composed = db as ComposedDatabase;
    const { databaseName, instance } = extractDatabaseResourceName(db.name);

    composed.databaseName = databaseName;
    composed.instance = instance;
    const instanceEntity =
      composed.instance === UNKNOWN_INSTANCE_NAME
        ? unknownInstance()
        : instanceV1Store.getInstanceByName(composed.instance) ??
          unknownInstance();
    composed.instanceEntity = {
      ...instanceEntity,
      environmentEntity:
        environmentV1Store.getEnvironmentByName(instanceEntity.environment) ??
        unknownEnvironment(),
    };
    composed.projectEntity = projectV1Store.getProjectByName(db.project);
    composed.effectiveEnvironmentEntity =
      environmentV1Store.getEnvironmentByName(db.effectiveEnvironment) ??
      unknownEnvironment();
    return markRaw(composed);
  });
};
