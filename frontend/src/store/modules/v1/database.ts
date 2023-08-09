import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import {
  ComposedInstance,
  ComposedDatabase,
  emptyDatabase,
  EMPTY_ID,
  MaybeRef,
  unknownDatabase,
  unknownEnvironment,
  unknownInstance,
  UNKNOWN_ID,
  UNKNOWN_INSTANCE_NAME,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import {
  Database,
  ListDatabasesRequest,
  SearchDatabasesRequest,
  UpdateDatabaseRequest,
} from "@/types/proto/v1/database_service";
import {
  extractDatabaseResourceName,
  hasWorkspacePermissionV1,
  isMemberOfProjectV1,
} from "@/utils";
import { useEnvironmentV1Store } from "./environment";
import { useInstanceV1Store } from "./instance";
import { useProjectV1Store } from "./project";

export const useDatabaseV1Store = defineStore("database_v1", () => {
  const databaseMapByName = reactive(new Map<string, ComposedDatabase>());
  const databaseMapByUID = reactive(new Map<string, ComposedDatabase>());

  // Getters
  const databaseList = computed(() => {
    return Array.from(databaseMapByName.values());
  });

  // Actions
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
  const fetchDatabaseList = async (args: Partial<ListDatabasesRequest>) => {
    const { databases } = await databaseServiceClient.listDatabases(args);
    const composedDatabaseList = await upsertDatabaseMap(databases);
    return composedDatabaseList;
  };
  const searchDatabaseList = async (args: Partial<SearchDatabasesRequest>) => {
    const { databases } = await databaseServiceClient.searchDatabases(args);
    const composedDatabaseList = await upsertDatabaseMap(databases);
    return composedDatabaseList;
  };
  const syncDatabase = async (database: string) => {
    await databaseServiceClient.syncDatabase({
      name: database,
    });
  };
  const databaseListByUser = (user: User) => {
    const canManageDatabase = hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-database",
      user.userRole
    );
    return databaseList.value.filter((db) => {
      if (canManageDatabase) return true;
      if (isMemberOfProjectV1(db.projectEntity.iamPolicy, user)) return true;
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
      (db) => db.instanceEntity.environment === environment
    );
  };
  const getDatabaseByName = (name: string) => {
    return databaseMapByName.get(name) ?? unknownDatabase();
  };
  const fetchDatabaseByName = async (name: string) => {
    const database = await databaseServiceClient.getDatabase({
      name,
    });

    const [composed] = await upsertDatabaseMap([database]);

    return composed;
  };
  const getOrFetchDatabaseByName = async (name: string) => {
    const existed = databaseMapByName.get(name);
    if (existed) {
      return existed;
    }
    await fetchDatabaseByName(name);
    return getDatabaseByName(name);
  };
  const getDatabaseByUID = (uid: string) => {
    if (uid === String(EMPTY_ID)) return emptyDatabase();
    if (uid === String(UNKNOWN_ID)) return unknownDatabase();

    return databaseMapByUID.get(uid) ?? unknownDatabase();
  };
  const fetchDatabaseByUID = async (uid: string) => {
    const database = await databaseServiceClient.getDatabase({
      name: `instances/-/databases/${uid}`,
    });
    const [composed] = await upsertDatabaseMap([database]);

    return composed;
  };
  const getOrFetchDatabaseByUID = async (uid: string) => {
    if (uid === String(EMPTY_ID)) return emptyDatabase();
    if (uid === String(UNKNOWN_ID)) return unknownDatabase();

    const existed = databaseList.value.find((db) => db.uid === uid);
    if (existed) {
      return existed;
    }
    await fetchDatabaseByUID(uid);
    return getDatabaseByUID(uid);
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

  return {
    databaseList,
    fetchDatabaseList,
    searchDatabaseList,
    syncDatabase,
    databaseListByUser,
    databaseListByProject,
    databaseListByInstance,
    databaseListByEnvironment,
    getDatabaseByName,
    fetchDatabaseByName,
    getOrFetchDatabaseByName,
    fetchDatabaseByUID,
    getDatabaseByUID,
    getOrFetchDatabaseByUID,
    updateDatabase,
    fetchDatabaseSchema,
    updateDatabaseInstance,
  };
});

export const useSearchDatabaseV1List = (
  args: MaybeRef<Partial<ListDatabasesRequest>>
) => {
  const store = useDatabaseV1Store();
  const ready = ref(false);
  const databaseList = ref<ComposedDatabase[]>([]);
  watch(
    () => JSON.stringify(unref(args)),
    () => {
      ready.value = false;
      store.searchDatabaseList(unref(args)).then((list) => {
        databaseList.value = list;
        ready.value = true;
      });
    },
    { immediate: true }
  );

  return { databaseList, ready };
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

export const useDatabaseV1ByUID = (uid: MaybeRef<string>) => {
  const store = useDatabaseV1Store();
  const ready = ref(true);
  watch(
    () => unref(uid),
    (uid) => {
      if (uid !== String(UNKNOWN_ID)) {
        if (store.getDatabaseByUID(uid).uid === String(UNKNOWN_ID)) {
          ready.value = false;
          store.fetchDatabaseByUID(uid).then(() => {
            ready.value = true;
          });
        }
      }
    },
    { immediate: true }
  );
  const database = computed(() => store.getDatabaseByUID(unref(uid)));

  return {
    database,
    ready,
  };
};

const batchComposeDatabase = async (databaseList: Database[]) => {
  const projectV1Store = useProjectV1Store();
  const instanceV1Store = useInstanceV1Store();
  const environmentV1Store = useEnvironmentV1Store();

  const distinctProjectList = uniq(databaseList.map((db) => db.project));
  const distinctInstanceList = uniq(
    databaseList
      .map((db) => `instances/${extractDatabaseResourceName(db.name).instance}`)
      .filter((instance) => instance !== UNKNOWN_INSTANCE_NAME)
  );
  await Promise.all(
    distinctProjectList.map((project) =>
      projectV1Store.getOrFetchProjectByName(project)
    )
  );
  await Promise.all(
    distinctInstanceList.map((instance) =>
      instanceV1Store.getOrFetchInstanceByName(instance)
    )
  );
  return databaseList.map((db) => {
    const composed = db as ComposedDatabase;
    const extractedResourceNames = extractDatabaseResourceName(db.name);

    composed.databaseName = extractedResourceNames.database;
    composed.instance = `instances/${extractedResourceNames.instance}`;
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
    return composed;
  });
};
